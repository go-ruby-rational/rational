// Copyright (c) the go-ruby-rational/rational authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rational

import (
	"math"
	"math/big"
	"testing"
)

// bigInt parses a base-10 integer for a test, failing on error.
func bigInt(t *testing.T, s string) *big.Int {
	t.Helper()
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		t.Fatalf("bad int literal %q", s)
	}
	return n
}

// newBig builds a Rational from decimal-string numerator/denominator.
func newBig(t *testing.T, num, den string) *Rational {
	t.Helper()
	x, err := New(bigInt(t, num), bigInt(t, den))
	if err != nil {
		t.Fatalf("New(%s,%s): %v", num, den, err)
	}
	return x
}

// wantRat is the exact expected big.Rat result, formatted as canonical to_s.
func wantRat(r *big.Rat) string { return r.Num().String() + "/" + r.Denom().String() }

// --- Overflow-checked int64 primitives -------------------------------------

func TestCheckedMul(t *testing.T) {
	cases := []struct {
		a, b int64
		want int64
		ok   bool
	}{
		{0, 5, 0, true},                         // zero, non-negative branch
		{6, 7, 42, true},                        // positive
		{-6, 7, -42, true},                      // negative (small magnitude)
		{6, -7, -42, true},                      // negative (b<0)
		{-6, -7, 42, true},                      // both negative → positive
		{math.MaxInt64, 1, math.MaxInt64, true}, // positive boundary lo==MaxInt64
		{math.MinInt64, 1, math.MinInt64, true}, // negative boundary lo==1<<63
		{math.MaxInt64, 2, 0, false},            // positive overflow (hi==0, lo>MaxInt64)
		{-2, 6148914691236517206, 0, false},     // negative overflow (lo>MaxInt64+1)
		{math.MinInt64, 2, 0, false},            // high-word overflow (hi!=0)
		{1 << 40, 1 << 40, 0, false},            // high-word overflow (hi!=0)
	}
	for _, c := range cases {
		got, ok := checkedMul(c.a, c.b)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("checkedMul(%d,%d) = (%d,%v), want (%d,%v)", c.a, c.b, got, ok, c.want, c.ok)
		}
	}
}

func TestCheckedAdd(t *testing.T) {
	cases := []struct {
		a, b int64
		want int64
		ok   bool
	}{
		{2, 3, 5, true},
		{-2, 3, 1, true},              // mixed sign, no overflow
		{math.MaxInt64, 1, 0, false},  // positive overflow
		{math.MinInt64, -1, 0, false}, // negative overflow
	}
	for _, c := range cases {
		got, ok := checkedAdd(c.a, c.b)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("checkedAdd(%d,%d) = (%d,%v), want (%d,%v)", c.a, c.b, got, ok, c.want, c.ok)
		}
	}
}

func TestCheckedNeg(t *testing.T) {
	if v, ok := checkedNeg(5); !ok || v != -5 {
		t.Errorf("checkedNeg(5) = (%d,%v)", v, ok)
	}
	if _, ok := checkedNeg(math.MinInt64); ok {
		t.Errorf("checkedNeg(MinInt64) ok should be false")
	}
}

func TestMagUGcdU(t *testing.T) {
	if magU(5) != 5 || magU(-5) != 5 || magU(math.MinInt64) != 1<<63 {
		t.Errorf("magU broken")
	}
	if gcdU(12, 8) != 4 || gcdU(0, 5) != 5 {
		t.Errorf("gcdU broken")
	}
}

// --- parseDecimalFast (unit) -----------------------------------------------

func TestParseDecimalFast(t *testing.T) {
	cases := []struct {
		in       string
		num, den int64
		ok       bool
	}{
		{"3", 3, 1, true},
		{"-3", -3, 1, true},
		{"+3", 3, 1, true}, // leading '+'
		{"3.14159", 314159, 100000, true},
		{"0.5", 5, 10, true},                   // returned unreduced (newFrac64 reduces)
		{"1.2.3", 0, 0, false},                 // two dots
		{"3e2", 0, 0, false},                   // exponent letter
		{"", 0, 0, false},                      // no digit
		{"+", 0, 0, false},                     // sign only, no digit
		{"99999999999999999999", 0, 0, false},  // numerator overflow
		{"0.0000000000000000001", 0, 0, false}, // 10^scale overflow
	}
	for _, c := range cases {
		num, den, ok := parseDecimalFast(c.in)
		if ok != c.ok || (ok && (num != c.num || den != c.den)) {
			t.Errorf("parseDecimalFast(%q) = (%d,%d,%v), want (%d,%d,%v)",
				c.in, num, den, ok, c.num, c.den, c.ok)
		}
	}
}

// --- Fast-path reduction & representation -----------------------------------

func TestFastPathReduces(t *testing.T) {
	// Results land on the fast int64 path AND are reduced to lowest terms.
	if got := r(t, 1, 6).Add(r(t, 1, 6)); !got.small || got.Inspect() != "(1/3)" {
		t.Errorf("1/6 + 1/6 = %q small=%v", got.Inspect(), got.small)
	}
	if got := r(t, 2, 3).Mul(r(t, 3, 4)); !got.small || got.Inspect() != "(1/2)" {
		t.Errorf("2/3 * 3/4 = %q small=%v", got.Inspect(), got.small)
	}
	q, _ := r(t, 1, 2).Div(r(t, -3, 4)) // sign normalisation: -2/3
	if !q.small || q.Inspect() != "(-2/3)" {
		t.Errorf("1/2 / (-3/4) = %q small=%v", q.Inspect(), q.small)
	}
	// FromInt64 and a small decimal Parse are on the fast path.
	if !FromInt64(7).small {
		t.Errorf("FromInt64 not small")
	}
	d, _ := Parse("3.14159")
	if !d.small || d.ToS() != "314159/100000" {
		t.Errorf("Parse(3.14159) = %q small=%v", d.ToS(), d.small)
	}
	// A zero numerator normalises to 0/1 on both mul and div fast paths.
	if got := r(t, 0, 1).Mul(r(t, 2, 3)); !got.small || got.Inspect() != "(0/1)" {
		t.Errorf("0 * 2/3 = %q", got.Inspect())
	}
	z, _ := r(t, 0, 1).Div(r(t, -2, 3))
	if !z.small || z.Inspect() != "(0/1)" {
		t.Errorf("0 / (-2/3) = %q", z.Inspect())
	}
}

// TestOverflowPromotesToBig checks the fast→big boundary: operands that fit
// int64 but whose cross-multiply / reduced form does not must fall back to the
// big.Rat path and equal the exact big result bit-for-bit.
func TestOverflowPromotesToBig(t *testing.T) {
	check := func(name string, got *Rational, want *big.Rat) {
		t.Helper()
		if got.small {
			t.Errorf("%s: expected big representation (fallback), got fast path", name)
		}
		if got.ToS() != wantRat(want) {
			t.Errorf("%s: got %q, want %q", name, got.ToS(), wantRat(want))
		}
	}
	br := func(num, den string) *big.Rat {
		return new(big.Rat).SetFrac(bigInt(t, num), bigInt(t, den))
	}

	// Add: denominator product overflows (t1/t2 fit, den checkedMul overflows).
	a := newBig(t, "1", "3037000500")
	b := newBig(t, "1", "3037000501")
	check("add-den-overflow", a.Add(b), new(big.Rat).Add(a.rat(), b.rat()))

	// Add: a numerator cross-product (t1) overflows.
	a = newBig(t, "5000000000000000000", "1")
	check("add-t1-overflow", a.Add(r(t, 1, 2)),
		new(big.Rat).Add(a.rat(), r(t, 1, 2).rat()))

	// Add: the sum itself overflows (positive).
	a = newBig(t, "5000000000000000000", "1")
	b = newBig(t, "5000000000000000000", "1")
	check("add-sum-overflow-pos", a.Add(b), br("10000000000000000000", "1"))

	// Add: the sum itself overflows (negative).
	a = newBig(t, "-5000000000000000000", "1")
	b = newBig(t, "-5000000000000000000", "1")
	check("add-sum-overflow-neg", a.Add(b), br("-10000000000000000000", "1"))

	// Sub: negating the second cross-product overflows (b.num == MinInt64).
	a = FromInt64(1)
	b = FromInt64(math.MinInt64)
	check("sub-neg-overflow", a.Sub(b), new(big.Rat).Sub(a.rat(), b.rat()))

	// Mul: numerator product overflows.
	a = newBig(t, "4000000000", "1")
	b = newBig(t, "4000000000", "1")
	check("mul-overflow", a.Mul(b), br("16000000000000000000", "1"))

	// Div: numerator product overflows.
	a = newBig(t, "4000000000", "1")
	b = newBig(t, "1", "4000000000")
	q, _ := a.Div(b)
	check("div-num-overflow", q, br("16000000000000000000", "1"))

	// Div: denominator product overflows.
	a = newBig(t, "1", "4000000000")
	b = newBig(t, "4000000000", "1")
	q, _ = a.Div(b)
	check("div-den-overflow", q, br("1", "16000000000000000000"))

	// Div: sign-normalisation negation of the numerator overflows.
	a = FromInt64(math.MinInt64)
	b = FromInt64(-1)
	q, _ = a.Div(b)
	check("div-neg-num-overflow", q, br("9223372036854775808", "1"))

	// Div: sign-normalisation negation of the denominator overflows.
	a = r(t, 1, 2)
	b = FromInt64(-4611686018427387904)
	q, _ = a.Div(b)
	check("div-neg-den-overflow", q, br("-1", "9223372036854775808"))

	// Cmp: cross-products overflow (both, and each side alone).
	x := newBig(t, "9000000000000000000", "1")
	y := newBig(t, "1", "9000000000000000000")
	if x.Cmp(y) != 1 {
		t.Errorf("cmp x>y overflow fallback wrong")
	}
	if y.Cmp(x) != -1 {
		t.Errorf("cmp y<x overflow fallback wrong")
	}
}

// TestFastNegAbsRecipOverflow covers the unary MinInt64 promotions.
func TestFastNegAbsRecipOverflow(t *testing.T) {
	mn := FromInt64(math.MinInt64)
	if got := mn.Neg(); got.small || got.ToS() != "9223372036854775808/1" {
		t.Errorf("Neg(MinInt64) = %q small=%v", got.ToS(), got.small)
	}
	if got := mn.Abs(); got.small || got.ToS() != "9223372036854775808/1" {
		t.Errorf("Abs(MinInt64) = %q small=%v", got.ToS(), got.small)
	}
	recip, err := mn.Reciprocal()
	if err != nil || recip.small || recip.ToS() != "-1/9223372036854775808" {
		t.Errorf("Reciprocal(MinInt64) = %q small=%v err=%v", recip.ToS(), recip.small, err)
	}
	// Abs of a non-negative fast-path value returns it unchanged (fast path).
	if got := r(t, 3, 4).Abs(); !got.small || got.Inspect() != "(3/4)" {
		t.Errorf("Abs(3/4) = %q small=%v", got.Inspect(), got.small)
	}
	// Negative-numerator reciprocal stays on the fast path (sign moved to denom).
	rr, err := r(t, -2, 3).Reciprocal()
	if err != nil || !rr.small || rr.Inspect() != "(-3/2)" {
		t.Errorf("Reciprocal(-2/3) = %q small=%v", rr.Inspect(), rr.small)
	}
}

// TestBigRepresentation exercises every method on the *big.Rat slow path,
// with operands whose reduced form exceeds int64.
func TestBigRepresentation(t *testing.T) {
	// Coprime big numerator/denominator so reduction cannot demote to int64.
	big1 := newBig(t, "12345678901234567890", "9876543210987654323")
	if big1.small {
		t.Fatalf("expected big representation")
	}
	// FromInt with an out-of-int64 magnitude.
	fi := FromInt(bigInt(t, "99999999999999999999"))
	if fi.small || fi.ToS() != "99999999999999999999/1" {
		t.Errorf("FromInt(big) = %q small=%v", fi.ToS(), fi.small)
	}
	// Numerator / Denominator on the big path (returned copies).
	if big1.Numerator().String() != "12345678901234567890" {
		t.Errorf("big Numerator = %v", big1.Numerator())
	}
	if big1.Denominator().String() != "9876543210987654323" {
		t.Errorf("big Denominator = %v", big1.Denominator())
	}
	// ToS / ToF / ToR(clone) / rat on big.
	if big1.ToS() != "12345678901234567890/9876543210987654323" {
		t.Errorf("big ToS = %q", big1.ToS())
	}
	if f := big1.ToF(); f < 1.24 || f > 1.26 {
		t.Errorf("big ToF = %v", f)
	}
	if cl := big1.ToR(); cl.small || cl.ToS() != big1.ToS() {
		t.Errorf("big ToR clone = %q small=%v", cl.ToS(), cl.small)
	}
	// isZero on a big (non-zero) value, via Reciprocal.
	if rec, err := big1.Reciprocal(); err != nil || !rec.small && rec.ToS() == "" {
		t.Errorf("big Reciprocal err=%v", err)
	}
	// Arithmetic that keeps the value big (add two bigs).
	sum := big1.Add(fi)
	exp := new(big.Rat).Add(big1.rat(), fi.rat())
	if sum.ToS() != wantRat(exp) {
		t.Errorf("big Add = %q, want %q", sum.ToS(), wantRat(exp))
	}
	// Cmp / CmpInt / CmpFloat with a big operand.
	if big1.Cmp(fi) != -1 {
		t.Errorf("big Cmp wrong")
	}
	if big1.CmpInt(bigInt(t, "2")) != -1 {
		t.Errorf("big CmpInt wrong")
	}
	if c, ok := big1.CmpFloat(2.0); !ok || c != -1 {
		t.Errorf("big CmpFloat = %d,%v", c, ok)
	}
	// Rounding family on a big value.
	if big1.ToI().String() != "1" || big1.Floor().String() != "1" ||
		big1.Ceil().String() != "2" || big1.Round().String() != "1" ||
		big1.Truncate().String() != "1" {
		t.Errorf("big rounding family wrong")
	}
	// digit() Rational and Integer branches on a big value.
	if rat, _, isRat := big1.RoundN(2); !isRat || rat == nil {
		t.Errorf("big RoundN(2) not Rational")
	}
	if _, i, isRat := big1.RoundN(-1); isRat || i == nil {
		t.Errorf("big RoundN(-1) not Integer")
	}
	// Pow on a big value stays exact.
	if p, err := big1.Pow(big.NewInt(2)); err != nil || p.ToS() == "" {
		t.Errorf("big Pow err=%v", err)
	}
	// Neg / Abs on a big value.
	if big1.Neg().ToS() != "-12345678901234567890/9876543210987654323" {
		t.Errorf("big Neg wrong")
	}
	if big1.Neg().Abs().ToS() != big1.ToS() {
		t.Errorf("big Abs wrong")
	}
}
