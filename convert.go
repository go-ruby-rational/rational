// Copyright (c) the go-ruby-rational/rational authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rational

import (
	"math"
	"math/big"
)

// floatPow is the float fallback for Pow with a non-integer exponent. It is a
// thin wrapper over math.Pow so the dependency is isolated and testable; note
// that, like any libm, perfect-root cases can differ from MRI's C pow in the
// last ULP.
func floatPow(base, exp float64) float64 { return math.Pow(base, exp) }

// ToF returns the nearest float64 (Ruby's Rational#to_f). big.Rat.Float64 rounds
// to nearest-even, matching MRI.
func (a *Rational) ToF() float64 {
	f, _ := a.r.Float64()
	return f
}

// ToI truncates toward zero to an Integer (Ruby's Rational#to_i, which is an
// alias of truncate with no argument).
func (a *Rational) ToI() *big.Int {
	return a.truncInt()
}

// ToR returns a copy of the Rational (Ruby's Rational#to_r — it is already a
// Rational and returns itself).
func (a *Rational) ToR() *Rational {
	return wrap(a.r)
}

// truncInt returns the value truncated toward zero, as a *big.Int. Quo on
// big.Int truncates toward zero, matching Ruby's to_i / truncate.
func (a *Rational) truncInt() *big.Int {
	q := new(big.Int)
	q.Quo(a.r.Num(), a.r.Denom())
	return q
}

// floorInt returns the greatest integer <= a (Ruby's Rational#floor).
func (a *Rational) floorInt() *big.Int {
	num, den := a.r.Num(), a.r.Denom()
	q, r := new(big.Int).QuoRem(num, den, new(big.Int))
	if r.Sign() < 0 {
		q.Sub(q, big.NewInt(1))
	}
	return q
}

// ceilInt returns the least integer >= a (Ruby's Rational#ceil).
func (a *Rational) ceilInt() *big.Int {
	num, den := a.r.Num(), a.r.Denom()
	q, r := new(big.Int).QuoRem(num, den, new(big.Int))
	if r.Sign() > 0 {
		q.Add(q, big.NewInt(1))
	}
	return q
}

// roundInt rounds half away from zero to an integer (Ruby's Rational#round with
// no argument: 1/2 → 1, 5/2 → 3, -5/2 → -3).
func (a *Rational) roundInt() *big.Int {
	// floor(a + 1/2) is wrong at .5 for negatives; do it explicitly: scale the
	// fractional comparison. Compute q = trunc, rem = a - q over den, then compare
	// 2*|rem_num| with den.
	num, den := a.r.Num(), a.r.Denom()
	q, r := new(big.Int).QuoRem(num, den, new(big.Int)) // r has sign of num
	twice := new(big.Int).Mul(new(big.Int).Abs(r), big.NewInt(2))
	if twice.Cmp(den) >= 0 { // halfway or beyond rounds away from zero
		if num.Sign() >= 0 {
			q.Add(q, big.NewInt(1))
		} else {
			q.Sub(q, big.NewInt(1))
		}
	}
	return q
}

// Truncate truncates toward zero (Ruby's Rational#truncate).
func (a *Rational) Truncate() *big.Int { return a.truncInt() }

// Floor returns the greatest integer <= a (Ruby's Rational#floor).
func (a *Rational) Floor() *big.Int { return a.floorInt() }

// Ceil returns the least integer >= a (Ruby's Rational#ceil).
func (a *Rational) Ceil() *big.Int { return a.ceilInt() }

// Round rounds half away from zero (Ruby's Rational#round).
func (a *Rational) Round() *big.Int { return a.roundInt() }

// op selects which integer rounding rule a digit-aware call applies.
type op int

const (
	opTrunc op = iota
	opFloor
	opCeil
	opRound
)

// applyInt applies the chosen rounding rule to a Rational, returning an integer.
func (a *Rational) applyInt(o op) *big.Int {
	switch o {
	case opFloor:
		return a.floorInt()
	case opCeil:
		return a.ceilInt()
	case opRound:
		return a.roundInt()
	default:
		return a.truncInt()
	}
}

// digit performs the digit-aware floor/ceil/round/truncate that Ruby's Rational
// exposes via the optional argument. For n >= 1 it returns a Rational (scaled to
// n decimal places); for n <= 0 it returns an Integer, rounding off the low
// |n| decimal digits. ok reports whether the Rational branch applies.
//
//	r.floor(2)  → Rational            r.floor(-1) → Integer            r.floor → Integer
func (a *Rational) digit(o op, n int) (rat *Rational, integer *big.Int, isRat bool) {
	if n == 0 {
		return nil, a.applyInt(o), false
	}
	pow := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(abs(n))), nil)
	if n > 0 {
		// Scale up by 10^n, apply the integer rule, divide back: a Rational.
		scaled := wrap(new(big.Rat).Mul(a.r, new(big.Rat).SetInt(pow)))
		i := scaled.applyInt(o)
		return wrap(new(big.Rat).SetFrac(i, pow)), nil, true
	}
	// n < 0: scale down by 10^|n|, apply the rule, scale back up: an Integer.
	scaled := wrap(new(big.Rat).Quo(a.r, new(big.Rat).SetInt(pow)))
	i := scaled.applyInt(o)
	return nil, i.Mul(i, pow), false
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// FloorN is Ruby's Rational#floor(n): a Rational for n >= 1, an Integer otherwise.
func (a *Rational) FloorN(n int) (*Rational, *big.Int, bool) { return a.digit(opFloor, n) }

// CeilN is Ruby's Rational#ceil(n).
func (a *Rational) CeilN(n int) (*Rational, *big.Int, bool) { return a.digit(opCeil, n) }

// RoundN is Ruby's Rational#round(n).
func (a *Rational) RoundN(n int) (*Rational, *big.Int, bool) { return a.digit(opRound, n) }

// TruncateN is Ruby's Rational#truncate(n).
func (a *Rational) TruncateN(n int) (*Rational, *big.Int, bool) { return a.digit(opTrunc, n) }
