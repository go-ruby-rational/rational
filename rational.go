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
type Rational struct {
	r *big.Rat
}

// wrap builds a Rational around a *big.Rat, copying so callers cannot mutate the
// shared value. big.Rat already keeps lowest terms with a positive denominator,
// matching Ruby's normalisation.
func wrap(r *big.Rat) *Rational {
	c := new(big.Rat).Set(r)
	return &Rational{r: c}
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
	return wrap(new(big.Rat).SetInt(n))
}

// FromInt64 is a convenience constructor for FromInt(big.NewInt(n)).
func FromInt64(n int64) *Rational {
	return wrap(new(big.Rat).SetInt64(n))
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
	// Bare integer or decimal/scientific form: big.Rat parses both exactly.
	r, ok := new(big.Rat).SetString(t)
	if !ok {
		return nil, ErrInvalidArgument
	}
	return wrap(r), nil
}

// Numerator returns the numerator (Ruby's Rational#numerator); the sign of the
// Rational lives here.
func (a *Rational) Numerator() *big.Int {
	return new(big.Int).Set(a.r.Num())
}

// Denominator returns the (always positive) denominator (Ruby's
// Rational#denominator).
func (a *Rational) Denominator() *big.Int {
	return new(big.Int).Set(a.r.Denom())
}
