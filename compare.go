// Copyright (c) the go-ruby-rational/rational authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rational

import (
	"math"
	"math/big"
)

// Cmp compares two Rationals, returning -1, 0 or +1 (Ruby's Rational#<=>).
func (a *Rational) Cmp(b *Rational) int {
	return a.r.Cmp(b.r)
}

// CmpInt compares with an Integer (Ruby's Rational#<=> Integer), returning -1, 0
// or +1.
func (a *Rational) CmpInt(n *big.Int) int {
	return a.r.Cmp(new(big.Rat).SetInt(n))
}

// CmpFloat compares with a Float (Ruby's Rational#<=> Float). The two booleans
// follow Ruby: when the float is NaN the comparison is undefined and ok is false
// (Ruby returns nil), otherwise ok is true and c is -1, 0 or +1.
func (a *Rational) CmpFloat(f float64) (c int, ok bool) {
	if math.IsNaN(f) { // big.Float.SetFloat64 panics on NaN; Ruby returns nil.
		return 0, false
	}
	if math.IsInf(f, 0) {
		// ±Inf: every finite Rational is below +Inf and above -Inf.
		if f > 0 {
			return -1, true
		}
		return 1, true
	}
	// A finite float64 is exactly representable as a Rational; comparing the two
	// rationals is exact and matches MRI, which compares the float's exact value.
	bf := new(big.Rat).SetFloat64(f)
	return a.r.Cmp(bf), true
}

// Eql reports a == b across two Rationals (Ruby's Rational#== with a Rational).
func (a *Rational) Eql(b *Rational) bool {
	return a.r.Cmp(b.r) == 0
}

// EqlInt reports a == n (Ruby's Rational#== Integer, e.g. (3/1) == 3 is true).
func (a *Rational) EqlInt(n *big.Int) bool {
	return a.CmpInt(n) == 0
}

// EqlFloat reports a == f (Ruby's Rational#== Float, e.g. (1/2) == 0.5 is true).
func (a *Rational) EqlFloat(f float64) bool {
	c, ok := a.CmpFloat(f)
	return ok && c == 0
}

// EqlStrict reports Ruby's Rational#eql? — true only for an equal Rational; an
// equal Integer or Float is not eql? to a Rational ((3/1).eql?(3) is false).
func (a *Rational) EqlStrict(b *Rational) bool {
	return a.Eql(b)
}

// Rationalize returns the Rational itself (Ruby's Rational#rationalize with no
// argument: a Rational is already exact).
func (a *Rational) Rationalize() *Rational {
	return wrap(a.r)
}

// ToS renders the value as Ruby's Rational#to_s — "num/den" with no parentheses,
// always showing the denominator (e.g. "3/4", "-3/4", "0/1", "2/1").
func (a *Rational) ToS() string {
	return a.r.Num().String() + "/" + a.r.Denom().String()
}

// Inspect renders the value as Ruby's Rational#inspect — the to_s form wrapped in
// parentheses (e.g. "(3/4)", "(-3/4)").
func (a *Rational) Inspect() string {
	return "(" + a.ToS() + ")"
}

// String makes Rational a fmt.Stringer; it returns the inspect form, matching
// how Ruby prints a Rational with p / inspect.
func (a *Rational) String() string {
	return a.Inspect()
}
