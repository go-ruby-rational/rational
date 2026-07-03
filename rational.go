// Copyright (c) the go-ruby-rational/rational authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package rational is a pure-Go (no cgo), MRI-4.0.5-byte-exact reimplementation
// of Ruby's Rational — the exact rational number type backing literals such as
// 3r and the Rational() conversion method.
//
// A Rational holds an exact ratio numerator/denominator of arbitrary-precision
// integers, always stored in lowest terms with a positive denominator (Ruby's
// normalisation: the sign lives on the numerator). The arithmetic, comparison
// and conversion methods mirror MRI's results exactly, including the inspect /
// to_s distinction ("(3/4)" vs "3/4"), round-half-away-from-zero rounding, the
// digit-aware floor/ceil/round/truncate that return a Rational for n >= 1 and an
// Integer for n <= 0, and the rule that ** with an integer exponent stays exact
// while a Rational or Float exponent falls back to Float (to_f ** exp).
//
// It is the Rational backend for go-embedded-ruby's numeric tower, but is a
// standalone, reusable module with no dependency on the Ruby runtime — a sibling
// of go-ruby-bigdecimal. It is MRI-faithful and therefore distinct from
// go-composites/rational, which models a generic mathematical rational.
package rational

import (
	"errors"
	"math/big"
	"strings"
)

// ErrZeroDivision is returned (wrapped) when a Rational would be formed with a
// zero denominator or a non-positive power of zero is requested. It mirrors
// Ruby's ZeroDivisionError ("divided by 0").
var ErrZeroDivision = errors.New("divided by 0")

// ErrInvalidArgument mirrors Ruby's ArgumentError for an unparseable Rational
// string ("invalid value for convert()").
var ErrInvalidArgument = errors.New("invalid value for convert()")

// Rational is an exact ratio of two arbitrary-precision integers, normalised to
// lowest terms with a positive denominator. The zero value is not valid; build a
// Rational with New, FromInt or Parse.
//
// It carries two representations. Values whose numerator AND denominator both fit
// in an int64 use the machine-word "fast path" (small == true, num/den held
// directly, big == nil): arithmetic on two such values is done in int64 with
// overflow detection and, on overflow, promotes to the *big.Rat path — this is
// the MRI fixnum-num/den optimisation, and it never loses precision. Values that
// do not fit int64 use the *big.Rat "slow path" (small == false). Both forms hold
// the value in lowest terms with a positive denominator, exactly matching Ruby's
// normalisation and MRI's byte-exact results.
type Rational struct {
	small    bool     // true: value is num/den; false: value is *big
	num, den int64    // valid when small; den > 0, gcd(|num|, den) == 1
	big      *big.Rat // valid when !small; nil otherwise
}

// wrap builds a Rational from a *big.Rat. big.Rat already keeps lowest terms with
// a positive denominator (matching Ruby's normalisation), so when both numerator
// and denominator fit int64 the value is demoted to the fast int64 path; larger
// values keep a private copy of the *big.Rat so callers cannot mutate it.
func wrap(r *big.Rat) *Rational {
	if n, d := r.Num(), r.Denom(); n.IsInt64() && d.IsInt64() {
		return &Rational{small: true, num: n.Int64(), den: d.Int64()}
	}
	return &Rational{big: new(big.Rat).Set(r)}
}

// rat returns the value as a *big.Rat. For a fast-path value it builds a fresh
// *big.Rat (den > 0 is guaranteed, so SetFrac64 never panics); for a slow-path
// value it returns the private *big.Rat, which callers must treat as read-only.
func (a *Rational) rat() *big.Rat {
	if a.small {
		return new(big.Rat).SetFrac64(a.num, a.den)
	}
	return a.big
}

// clone returns an independent copy sharing no mutable state with a.
func (a *Rational) clone() *Rational {
	if a.small {
		return &Rational{small: true, num: a.num, den: a.den}
	}
	return &Rational{big: new(big.Rat).Set(a.big)}
}

// isZero reports whether the value is 0, on either representation.
func (a *Rational) isZero() bool {
	if a.small {
		return a.num == 0
	}
	return a.big.Sign() == 0
}

// New returns the Rational num/den reduced to lowest terms with a positive
// denominator (Ruby's Rational(num, den)). A zero denominator yields
// ErrZeroDivision, exactly as MRI raises ZeroDivisionError.
func New(num, den *big.Int) (*Rational, error) {
	if den.Sign() == 0 {
		return nil, ErrZeroDivision
	}
	return wrap(new(big.Rat).SetFrac(num, den)), nil
}

// FromInt returns the Rational n/1 (Ruby's Integer#to_r / Rational(n)).
func FromInt(n *big.Int) *Rational {
	if n.IsInt64() {
		return &Rational{small: true, num: n.Int64(), den: 1}
	}
	return &Rational{big: new(big.Rat).SetInt(n)}
}

// FromInt64 is a convenience constructor for FromInt(big.NewInt(n)); the value
// n/1 always fits the fast int64 path.
func FromInt64(n int64) *Rational {
	return &Rational{small: true, num: n, den: 1}
}

// Parse converts a string to a Rational the way Ruby's Rational(String) does:
// it accepts surrounding whitespace, an optional sign, a bare integer ("3"), a
// fraction ("3/4", "-3/4", "10/4" → 5/2) and a decimal ("1.25", "1.5e2"). An
// unparseable string (or a fraction with a zero denominator) returns an error
// mirroring Ruby's ArgumentError / ZeroDivisionError.
func Parse(s string) (*Rational, error) {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil, ErrInvalidArgument
	}
	if i := strings.IndexByte(t, '/'); i >= 0 {
		numS, denS := t[:i], t[i+1:]
		num, ok1 := new(big.Int).SetString(strings.TrimSpace(numS), 10)
		den, ok2 := new(big.Int).SetString(strings.TrimSpace(denS), 10)
		if !ok1 || !ok2 {
			return nil, ErrInvalidArgument
		}
		return New(num, den)
	}
	// Bare integer or plain decimal: an int64 fast path avoids allocating a
	// *big.Rat for the common small case ("3", "3.14159"), matching MRI's exact
	// result. Scientific notation and out-of-int64 magnitudes fall through to the
	// big.Rat parser, which parses both forms exactly.
	if num, den, ok := parseDecimalFast(t); ok {
		return newFrac64(num, den), nil
	}
	r, ok := new(big.Rat).SetString(t)
	if !ok {
		return nil, ErrInvalidArgument
	}
	return wrap(r), nil
}

// Numerator returns the numerator (Ruby's Rational#numerator); the sign of the
// Rational lives here.
func (a *Rational) Numerator() *big.Int {
	if a.small {
		return big.NewInt(a.num)
	}
	return new(big.Int).Set(a.big.Num())
}

// Denominator returns the (always positive) denominator (Ruby's
// Rational#denominator).
func (a *Rational) Denominator() *big.Int {
	if a.small {
		return big.NewInt(a.den)
	}
	return new(big.Int).Set(a.big.Denom())
}
