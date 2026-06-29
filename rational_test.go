// Copyright (c) the go-ruby-rational/rational authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rational

import (
	"errors"
	"math"
	"math/big"
	"testing"
)

// r is a test helper building a Rational from two int64s; it fails the test on a
// zero denominator so call sites stay terse.
func r(t *testing.T, n, d int64) *Rational {
	t.Helper()
	x, err := New(big.NewInt(n), big.NewInt(d))
	if err != nil {
		t.Fatalf("New(%d,%d): %v", n, d, err)
	}
	return x
}

func TestNewNormalises(t *testing.T) {
	cases := []struct {
		n, d int64
		want string
	}{
		{2, 4, "(1/2)"},
		{-1, 2, "(-1/2)"},
		{1, -2, "(-1/2)"},
		{-1, -2, "(1/2)"},
		{0, 5, "(0/1)"},
		{6, 3, "(2/1)"},
		{-6, 4, "(-3/2)"},
	}
	for _, c := range cases {
		if got := r(t, c.n, c.d).Inspect(); got != c.want {
			t.Errorf("New(%d,%d).Inspect() = %q, want %q", c.n, c.d, got, c.want)
		}
	}
}

func TestNewZeroDen(t *testing.T) {
	_, err := New(big.NewInt(1), big.NewInt(0))
	if !errors.Is(err, ErrZeroDivision) {
		t.Fatalf("New(1,0) err = %v, want ErrZeroDivision", err)
	}
}

func TestFromInt(t *testing.T) {
	if got := FromInt(big.NewInt(5)).Inspect(); got != "(5/1)" {
		t.Errorf("FromInt(5) = %q", got)
	}
	if got := FromInt64(-3).Inspect(); got != "(-3/1)" {
		t.Errorf("FromInt64(-3) = %q", got)
	}
}

func TestNumeratorDenominator(t *testing.T) {
	x := r(t, -6, 4) // (-3/2)
	if x.Numerator().Int64() != -3 {
		t.Errorf("Numerator = %v", x.Numerator())
	}
	if x.Denominator().Int64() != 2 {
		t.Errorf("Denominator = %v", x.Denominator())
	}
	// The returned ints are copies; mutating them must not change the Rational.
	x.Numerator().SetInt64(99)
	x.Denominator().SetInt64(99)
	if x.Inspect() != "(-3/2)" {
		t.Errorf("Rational mutated through returned ints: %q", x.Inspect())
	}
}

func TestParse(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"3/4", "(3/4)"},
		{"3", "(3/1)"},
		{"-3/4", "(-3/4)"},
		{"  3/4  ", "(3/4)"},
		{"0.5", "(1/2)"},
		{"1.25", "(5/4)"},
		{"-1.25", "(-5/4)"},
		{"10/4", "(5/2)"},
		{"12", "(12/1)"},
		{"3e2", "(300/1)"},
		{"  -2 / 6 ", "(-1/3)"},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil {
			t.Errorf("Parse(%q) err = %v", c.in, err)
			continue
		}
		if got.Inspect() != c.want {
			t.Errorf("Parse(%q) = %q, want %q", c.in, got.Inspect(), c.want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, in := range []string{"", "  ", "abc", "3/x", "x/3", "1.2.3", "3/0"} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) expected error", in)
		}
	}
	// A zero-denominator fraction surfaces ErrZeroDivision specifically.
	if _, err := Parse("3/0"); !errors.Is(err, ErrZeroDivision) {
		t.Errorf("Parse(3/0) err = %v, want ErrZeroDivision", err)
	}
}

func TestArith(t *testing.T) {
	if got := r(t, 1, 3).Add(r(t, 1, 6)).Inspect(); got != "(1/2)" {
		t.Errorf("1/3 + 1/6 = %q", got)
	}
	if got := r(t, 1, 2).Sub(r(t, 1, 3)).Inspect(); got != "(1/6)" {
		t.Errorf("1/2 - 1/3 = %q", got)
	}
	if got := r(t, 2, 3).Mul(r(t, 3, 4)).Inspect(); got != "(1/2)" {
		t.Errorf("2/3 * 3/4 = %q", got)
	}
	q, err := r(t, 3, 4).Div(r(t, 1, 2))
	if err != nil || q.Inspect() != "(3/2)" {
		t.Errorf("3/4 / 1/2 = %q, %v", q, err)
	}
}

func TestDivByZero(t *testing.T) {
	_, err := r(t, 1, 2).Div(r(t, 0, 1))
	if !errors.Is(err, ErrZeroDivision) {
		t.Fatalf("Div by zero err = %v", err)
	}
}

func TestNegAbsReciprocal(t *testing.T) {
	if got := r(t, 3, 4).Neg().Inspect(); got != "(-3/4)" {
		t.Errorf("Neg = %q", got)
	}
	if got := r(t, -3, 4).Abs().Inspect(); got != "(3/4)" {
		t.Errorf("Abs = %q", got)
	}
	recip, err := r(t, 2, 3).Reciprocal()
	if err != nil || recip.Inspect() != "(3/2)" {
		t.Errorf("Reciprocal = %q, %v", recip, err)
	}
	if _, err := r(t, 0, 1).Reciprocal(); !errors.Is(err, ErrZeroDivision) {
		t.Errorf("Reciprocal(0) err = %v", err)
	}
}

func TestPow(t *testing.T) {
	cases := []struct {
		n, d, e int64
		want    string
	}{
		{3, 4, 2, "(9/16)"},
		{3, 4, -1, "(4/3)"},
		{2, 1, 3, "(8/1)"},
		{1, 2, 0, "(1/1)"},
		{2, 3, -2, "(9/4)"},
		{-2, 3, 3, "(-8/27)"},
		{-2, 3, 2, "(4/9)"},
		{0, 1, 0, "(1/1)"},
		{0, 1, 3, "(0/1)"},
		{2, 3, -3, "(27/8)"},
	}
	for _, c := range cases {
		got, err := r(t, c.n, c.d).Pow(big.NewInt(c.e))
		if err != nil {
			t.Errorf("(%d/%d)**%d err = %v", c.n, c.d, c.e, err)
			continue
		}
		if got.Inspect() != c.want {
			t.Errorf("(%d/%d)**%d = %q, want %q", c.n, c.d, c.e, got.Inspect(), c.want)
		}
	}
}

func TestPowZeroNegative(t *testing.T) {
	if _, err := r(t, 0, 1).Pow(big.NewInt(-1)); !errors.Is(err, ErrZeroDivision) {
		t.Errorf("0 ** -1 err = %v, want ErrZeroDivision", err)
	}
}

func TestPowFloat(t *testing.T) {
	// 0.25 ** 0.5 == 0.5 and 2 ** 0.5 == sqrt2 are libm-stable on every target.
	if got := r(t, 1, 4).PowFloat(0.5); got != 0.5 {
		t.Errorf("(1/4)**0.5 = %v, want 0.5", got)
	}
	if got := r(t, 2, 1).PowFloat(0.5); math.Abs(got-math.Sqrt2) > 1e-15 {
		t.Errorf("(2/1)**0.5 = %v, want ~sqrt2", got)
	}
}

func TestConversions(t *testing.T) {
	if r(t, 7, 2).ToI().Int64() != 3 {
		t.Errorf("ToI 7/2")
	}
	if r(t, -7, 2).ToI().Int64() != -3 {
		t.Errorf("ToI -7/2")
	}
	if r(t, 7, 2).ToF() != 3.5 {
		t.Errorf("ToF 7/2")
	}
	if r(t, 3, 4).ToR().Inspect() != "(3/4)" {
		t.Errorf("ToR")
	}
}

func TestTruncateFloorCeilRound(t *testing.T) {
	cases := []struct {
		n, d                      int64
		trunc, floor, ceil, round int64
	}{
		{7, 2, 3, 3, 4, 4},
		{-7, 2, -3, -4, -3, -4},
		{5, 2, 2, 2, 3, 3},
		{-5, 2, -2, -3, -2, -3},
		{1, 2, 0, 0, 1, 1},
		{3, 2, 1, 1, 2, 2},
		{-3, 2, -1, -2, -1, -2},
	}
	for _, c := range cases {
		x := r(t, c.n, c.d)
		if got := x.Truncate().Int64(); got != c.trunc {
			t.Errorf("(%d/%d).Truncate = %d, want %d", c.n, c.d, got, c.trunc)
		}
		if got := x.Floor().Int64(); got != c.floor {
			t.Errorf("(%d/%d).Floor = %d, want %d", c.n, c.d, got, c.floor)
		}
		if got := x.Ceil().Int64(); got != c.ceil {
			t.Errorf("(%d/%d).Ceil = %d, want %d", c.n, c.d, got, c.ceil)
		}
		if got := x.Round().Int64(); got != c.round {
			t.Errorf("(%d/%d).Round = %d, want %d", c.n, c.d, got, c.round)
		}
	}
}

func TestDigitRoundingRational(t *testing.T) {
	// n >= 1 returns a Rational (isRat true).
	cases := []struct {
		op       op
		n, d, dg int64
		want     string
	}{
		{opRound, 1, 3, 2, "(33/100)"},
		{opRound, 2, 3, 2, "(67/100)"},
		{opRound, -2, 3, 2, "(-67/100)"},
		{opRound, 355, 113, 4, "(3927/1250)"},
		{opFloor, 1, 3, 2, "(33/100)"},
		{opCeil, 1, 3, 2, "(17/50)"},
		{opFloor, -1, 3, 2, "(-17/50)"},
		{opCeil, -1, 3, 2, "(-33/100)"},
		{opTrunc, 355, 113, 2, "(157/50)"},
		{opTrunc, -1, 3, 2, "(-33/100)"},
	}
	for _, c := range cases {
		rat, _, isRat := r(t, c.n, c.d).digit(c.op, int(c.dg))
		if !isRat {
			t.Errorf("op %d (%d/%d) digit %d: isRat=false", c.op, c.n, c.d, c.dg)
			continue
		}
		if rat.Inspect() != c.want {
			t.Errorf("op %d (%d/%d) digit %d = %q, want %q", c.op, c.n, c.d, c.dg, rat.Inspect(), c.want)
		}
	}
}

func TestDigitRoundingInteger(t *testing.T) {
	// n <= 0 returns an Integer (isRat false).
	cases := []struct {
		op    op
		n, dg int64
		want  int64
	}{
		{opRound, 127, -1, 130},
		{opFloor, 127, -1, 120},
		{opCeil, 127, -1, 130},
		{opTrunc, 127, -1, 120},
		{opFloor, 123, -1, 120},
		{opRound, 7, 0, 7}, // n == 0 path → integer (round of whole 7 is 7)
	}
	for _, c := range cases {
		_, i, isRat := FromInt64(c.n).digit(c.op, int(c.dg))
		if isRat {
			t.Errorf("op %d %d digit %d: isRat=true", c.op, c.n, c.dg)
			continue
		}
		if i.Int64() != c.want {
			t.Errorf("op %d %d digit %d = %v, want %d", c.op, c.n, c.dg, i, c.want)
		}
	}
	// n == 0 round of 7/2 → 4 (away from zero) via FloorN/CeilN/RoundN/TruncateN.
	if _, i, isRat := r(t, 7, 2).RoundN(0); isRat || i.Int64() != 4 {
		t.Errorf("RoundN(0) 7/2 = %v isRat=%v", i, isRat)
	}
}

func TestDigitWrappers(t *testing.T) {
	rat, _, isRat := r(t, 1, 3).FloorN(2)
	if !isRat || rat.Inspect() != "(33/100)" {
		t.Errorf("FloorN(2) = %v", rat)
	}
	rat, _, isRat = r(t, 1, 3).CeilN(2)
	if !isRat || rat.Inspect() != "(17/50)" {
		t.Errorf("CeilN(2) = %v", rat)
	}
	rat, _, isRat = r(t, 1, 3).TruncateN(2)
	if !isRat || rat.Inspect() != "(33/100)" {
		t.Errorf("TruncateN(2) = %v", rat)
	}
	_, i, isRat := r(t, 127, 1).CeilN(-1)
	if isRat || i.Int64() != 130 {
		t.Errorf("CeilN(-1) 127 = %v", i)
	}
	_, i, isRat = r(t, 127, 1).TruncateN(-1)
	if isRat || i.Int64() != 120 {
		t.Errorf("TruncateN(-1) 127 = %v", i)
	}
}

func TestCompare(t *testing.T) {
	if r(t, 1, 2).Cmp(r(t, 1, 3)) != 1 {
		t.Errorf("1/2 <=> 1/3")
	}
	if r(t, 1, 2).Cmp(r(t, 1, 2)) != 0 {
		t.Errorf("1/2 <=> 1/2")
	}
	if r(t, 1, 3).Cmp(r(t, 1, 2)) != -1 {
		t.Errorf("1/3 <=> 1/2")
	}
	if r(t, 1, 2).CmpInt(big.NewInt(1)) != -1 {
		t.Errorf("1/2 <=> 1")
	}
	if r(t, 3, 1).CmpInt(big.NewInt(3)) != 0 {
		t.Errorf("3/1 <=> 3")
	}
	if r(t, 1, 2).CmpInt(big.NewInt(0)) != 1 {
		t.Errorf("1/2 <=> 0")
	}
}

func TestCmpFloat(t *testing.T) {
	c, ok := r(t, 1, 2).CmpFloat(0.5)
	if !ok || c != 0 {
		t.Errorf("1/2 <=> 0.5 = %d,%v", c, ok)
	}
	c, ok = r(t, 1, 2).CmpFloat(0.6)
	if !ok || c != -1 {
		t.Errorf("1/2 <=> 0.6 = %d,%v", c, ok)
	}
	c, ok = r(t, 1, 2).CmpFloat(0.4)
	if !ok || c != 1 {
		t.Errorf("1/2 <=> 0.4 = %d,%v", c, ok)
	}
	if _, ok := r(t, 1, 2).CmpFloat(math.NaN()); ok {
		t.Errorf("1/2 <=> NaN ok should be false")
	}
	if c, ok := r(t, 1, 2).CmpFloat(math.Inf(1)); !ok || c != -1 {
		t.Errorf("1/2 <=> +Inf = %d,%v", c, ok)
	}
	if c, ok := r(t, 1, 2).CmpFloat(math.Inf(-1)); !ok || c != 1 {
		t.Errorf("1/2 <=> -Inf = %d,%v", c, ok)
	}
}

func TestEql(t *testing.T) {
	if !r(t, 1, 2).Eql(r(t, 2, 4)) {
		t.Errorf("1/2 == 2/4")
	}
	if r(t, 1, 2).Eql(r(t, 1, 3)) {
		t.Errorf("1/2 == 1/3 should be false")
	}
	if !r(t, 3, 1).EqlInt(big.NewInt(3)) {
		t.Errorf("3/1 == 3")
	}
	if r(t, 1, 2).EqlInt(big.NewInt(1)) {
		t.Errorf("1/2 == 1 should be false")
	}
	if !r(t, 1, 2).EqlFloat(0.5) {
		t.Errorf("1/2 == 0.5")
	}
	if r(t, 1, 2).EqlFloat(0.6) {
		t.Errorf("1/2 == 0.6 should be false")
	}
	if r(t, 1, 2).EqlFloat(math.NaN()) {
		t.Errorf("1/2 == NaN should be false")
	}
	if !r(t, 1, 2).EqlStrict(r(t, 1, 2)) {
		t.Errorf("1/2 eql? 1/2")
	}
	if r(t, 1, 2).EqlStrict(r(t, 1, 3)) {
		t.Errorf("1/2 eql? 1/3 should be false")
	}
}

func TestRationalize(t *testing.T) {
	if got := r(t, 3, 4).Rationalize().Inspect(); got != "(3/4)" {
		t.Errorf("Rationalize 3/4 = %q", got)
	}
}

func TestToSInspectString(t *testing.T) {
	x := r(t, -3, 4)
	if x.ToS() != "-3/4" {
		t.Errorf("ToS = %q", x.ToS())
	}
	if x.Inspect() != "(-3/4)" {
		t.Errorf("Inspect = %q", x.Inspect())
	}
	if x.String() != "(-3/4)" {
		t.Errorf("String = %q", x.String())
	}
	if r(t, 0, 1).ToS() != "0/1" {
		t.Errorf("ToS 0/1")
	}
}

func TestFloatPowHelper(t *testing.T) {
	if got := floatPow(4, 0.5); got != 2 {
		t.Errorf("floatPow(4,0.5) = %v", got)
	}
}

func TestAbsHelper(t *testing.T) {
	if abs(-3) != 3 || abs(3) != 3 || abs(0) != 0 {
		t.Errorf("abs broken")
	}
}
