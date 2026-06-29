// Copyright (c) the go-ruby-rational/rational authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rational

import (
	"math/big"
	"os/exec"
	"strings"
	"testing"
)

// rubyBin locates a usable `ruby` (>= 4.0) once. The oracle tests skip themselves
// when it is absent (the qemu cross-arch lanes and the Windows lane) or when it
// is older than the MRI 4.0 reference, so the deterministic suite alone drives
// the 100% gate there.
func rubyBin(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping MRI oracle")
	}
	// Gate on RUBY_VERSION >= "4.0" so the byte-exact reference is MRI 4.x.
	out, err := exec.Command(path, "-e", "print RUBY_VERSION").Output()
	if err != nil {
		t.Skipf("cannot query RUBY_VERSION: %v", err)
	}
	if !rubyAtLeast(string(out), "4.0") {
		t.Skipf("ruby %s < 4.0; skipping MRI oracle", strings.TrimSpace(string(out)))
	}
	return path
}

// rubyAtLeast reports whether a dotted RUBY_VERSION string is >= a "major.minor"
// floor. Only the first two components are compared, which is all the gate needs.
func rubyAtLeast(version, floor string) bool {
	va := strings.Split(strings.TrimSpace(version), ".")
	fa := strings.Split(floor, ".")
	for i := 0; i < len(fa); i++ {
		if i >= len(va) {
			return false
		}
		v := atoiSafe(va[i])
		f := atoiSafe(fa[i])
		if v != f {
			return v > f
		}
	}
	return true
}

// atoiSafe parses a leading run of digits, ignoring any suffix (e.g. "5+PRISM").
func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// rubyEval runs a Ruby script and returns its trimmed stdout. The script
// $stdout.binmode + STDIN.binmode itself so Windows text-mode never pollutes the
// bytes (the go-ruby-erb lesson); ruby is absent on the Windows CI lane anyway,
// but the binmode keeps the contract uniform.
func rubyEval(t *testing.T, bin, script string) string {
	t.Helper()
	cmd := exec.Command(bin, "-e", "$stdout.binmode\nSTDIN.binmode\n"+script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error: %v\nscript:\n%s\noutput:\n%s", err, script, out)
	}
	return strings.TrimRight(string(out), "\r\n")
}

// TestOracleInspectToS checks inspect / to_s against MRI for a spread of values.
func TestOracleInspectToS(t *testing.T) {
	bin := rubyBin(t)
	cases := []struct{ n, d int64 }{
		{2, 4}, {3, 4}, {-3, 4}, {1, -2}, {-1, -2}, {0, 5}, {6, 3}, {-6, 4}, {7, 2},
	}
	for _, c := range cases {
		mri := rubyEval(t, bin, scriptF("r=Rational(%d,%d); puts r.inspect; puts r.to_s", c.n, c.d))
		lines := strings.Split(mri, "\n")
		got := r(t, c.n, c.d)
		if lines[0] != got.Inspect() {
			t.Errorf("(%d/%d) inspect: MRI %q, got %q", c.n, c.d, lines[0], got.Inspect())
		}
		if lines[1] != got.ToS() {
			t.Errorf("(%d/%d) to_s: MRI %q, got %q", c.n, c.d, lines[1], got.ToS())
		}
	}
}

// TestOracleArith checks +, -, *, / against MRI.
func TestOracleArith(t *testing.T) {
	bin := rubyBin(t)
	type pair struct{ n, d int64 }
	ops := []struct {
		sym string
		fn  func(a, b *Rational) string
	}{
		{"+", func(a, b *Rational) string { return a.Add(b).Inspect() }},
		{"-", func(a, b *Rational) string { return a.Sub(b).Inspect() }},
		{"*", func(a, b *Rational) string { return a.Mul(b).Inspect() }},
		{"/", func(a, b *Rational) string { q, _ := a.Div(b); return q.Inspect() }},
	}
	pairs := []struct{ a, b pair }{
		{pair{1, 3}, pair{1, 6}}, {pair{3, 4}, pair{1, 2}}, {pair{2, 3}, pair{3, 4}},
		{pair{-1, 2}, pair{1, 3}}, {pair{5, 6}, pair{7, 8}},
	}
	for _, p := range pairs {
		for _, op := range ops {
			mri := rubyEval(t, bin, scriptF(
				"puts (Rational(%d,%d) %s Rational(%d,%d)).inspect",
				p.a.n, p.a.d, op.sym, p.b.n, p.b.d))
			got := op.fn(r(t, p.a.n, p.a.d), r(t, p.b.n, p.b.d))
			if mri != got {
				t.Errorf("(%d/%d) %s (%d/%d): MRI %q, got %q",
					p.a.n, p.a.d, op.sym, p.b.n, p.b.d, mri, got)
			}
		}
	}
}

// TestOraclePow checks ** with integer exponents against MRI.
func TestOraclePow(t *testing.T) {
	bin := rubyBin(t)
	cases := []struct{ n, d, e int64 }{
		{3, 4, 2}, {3, 4, -1}, {2, 1, 3}, {1, 2, 0}, {2, 3, -2},
		{-2, 3, 3}, {-2, 3, 2}, {0, 1, 0}, {0, 1, 3}, {2, 3, -3},
	}
	for _, c := range cases {
		mri := rubyEval(t, bin, scriptF("puts (Rational(%d,%d) ** %d).inspect", c.n, c.d, c.e))
		got, err := r(t, c.n, c.d).Pow(big.NewInt(c.e))
		if err != nil {
			t.Errorf("(%d/%d)**%d err %v", c.n, c.d, c.e, err)
			continue
		}
		if mri != got.Inspect() {
			t.Errorf("(%d/%d)**%d: MRI %q, got %q", c.n, c.d, c.e, mri, got.Inspect())
		}
	}
}

// TestOracleConversions checks to_i / to_f / truncate / floor / ceil / round.
func TestOracleConversions(t *testing.T) {
	bin := rubyBin(t)
	cases := []struct{ n, d int64 }{
		{7, 2}, {-7, 2}, {5, 2}, {-5, 2}, {1, 2}, {3, 2}, {-3, 2}, {1, 3}, {-1, 3},
	}
	for _, c := range cases {
		mri := rubyEval(t, bin, scriptF(
			"r=Rational(%d,%d); puts r.to_i; puts r.to_f; puts r.truncate; puts r.floor; puts r.ceil; puts r.round",
			c.n, c.d))
		l := strings.Split(mri, "\n")
		x := r(t, c.n, c.d)
		check := func(idx int, label, got string) {
			if l[idx] != got {
				t.Errorf("(%d/%d).%s: MRI %q, got %q", c.n, c.d, label, l[idx], got)
			}
		}
		check(0, "to_i", x.ToI().String())
		check(1, "to_f", formatFloatLikeRuby(x.ToF()))
		check(2, "truncate", x.Truncate().String())
		check(3, "floor", x.Floor().String())
		check(4, "ceil", x.Ceil().String())
		check(5, "round", x.Round().String())
	}
}

// TestOracleDigitRounding checks the digit-aware floor/ceil/round/truncate.
func TestOracleDigitRounding(t *testing.T) {
	bin := rubyBin(t)
	type tc struct {
		n, d, dg int64
		meth     string
		fn       func(x *Rational, n int) (rat *Rational, i *big.Int, isRat bool)
	}
	cases := []tc{
		{1, 3, 2, "round", (*Rational).RoundN},
		{2, 3, 2, "round", (*Rational).RoundN},
		{-2, 3, 2, "round", (*Rational).RoundN},
		{355, 113, 4, "round", (*Rational).RoundN},
		{1, 3, 2, "floor", (*Rational).FloorN},
		{1, 3, 2, "ceil", (*Rational).CeilN},
		{-1, 3, 2, "floor", (*Rational).FloorN},
		{355, 113, 2, "truncate", (*Rational).TruncateN},
		{127, 1, -1, "round", (*Rational).RoundN},
		{127, 1, -1, "floor", (*Rational).FloorN},
		{127, 1, -1, "ceil", (*Rational).CeilN},
		{127, 1, -1, "truncate", (*Rational).TruncateN},
	}
	for _, c := range cases {
		mri := rubyEval(t, bin, scriptF(
			"r=Rational(%d,%d); v=r.%s(%d); puts v.class; puts v.inspect",
			c.n, c.d, c.meth, c.dg))
		l := strings.Split(mri, "\n")
		rat, i, isRat := c.fn(r(t, c.n, c.d), int(c.dg))
		var gotClass, gotInspect string
		if isRat {
			gotClass, gotInspect = "Rational", rat.Inspect()
		} else {
			gotClass, gotInspect = "Integer", i.String()
		}
		if l[0] != gotClass {
			t.Errorf("(%d/%d).%s(%d) class: MRI %q, got %q", c.n, c.d, c.meth, c.dg, l[0], gotClass)
		}
		if l[1] != gotInspect {
			t.Errorf("(%d/%d).%s(%d): MRI %q, got %q", c.n, c.d, c.meth, c.dg, l[1], gotInspect)
		}
	}
}

// TestOracleCompare checks <=> and == against MRI for Rational, Integer, Float.
func TestOracleCompare(t *testing.T) {
	bin := rubyBin(t)
	// Rational <=> Rational and ==.
	pairs := []struct{ an, ad, bn, bd int64 }{
		{1, 2, 1, 3}, {1, 2, 1, 2}, {1, 3, 1, 2}, {2, 4, 1, 2},
	}
	for _, p := range pairs {
		mri := rubyEval(t, bin, scriptF(
			"puts (Rational(%d,%d) <=> Rational(%d,%d)); puts (Rational(%d,%d) == Rational(%d,%d))",
			p.an, p.ad, p.bn, p.bd, p.an, p.ad, p.bn, p.bd))
		l := strings.Split(mri, "\n")
		a, b := r(t, p.an, p.ad), r(t, p.bn, p.bd)
		if l[0] != itoa(a.Cmp(b)) {
			t.Errorf("(%d/%d)<=>(%d/%d): MRI %q got %d", p.an, p.ad, p.bn, p.bd, l[0], a.Cmp(b))
		}
		if l[1] != boolStr(a.Eql(b)) {
			t.Errorf("(%d/%d)==(%d/%d): MRI %q got %v", p.an, p.ad, p.bn, p.bd, l[1], a.Eql(b))
		}
	}
	// Rational <=> Float and ==.
	floatCases := []struct {
		n, d int64
		f    string
	}{{1, 2, "0.5"}, {1, 2, "0.6"}, {1, 2, "0.4"}, {3, 1, "3.0"}}
	for _, c := range floatCases {
		mri := rubyEval(t, bin, scriptF(
			"r=Rational(%d,%d); puts (r <=> %s); puts (r == %s)", c.n, c.d, c.f, c.f))
		l := strings.Split(mri, "\n")
		x := r(t, c.n, c.d)
		f := mustFloat(t, c.f)
		cmp, _ := x.CmpFloat(f)
		if l[0] != itoa(cmp) {
			t.Errorf("(%d/%d)<=>%s: MRI %q got %d", c.n, c.d, c.f, l[0], cmp)
		}
		if l[1] != boolStr(x.EqlFloat(f)) {
			t.Errorf("(%d/%d)==%s: MRI %q got %v", c.n, c.d, c.f, l[1], x.EqlFloat(f))
		}
	}
}

// TestOracleParse checks Rational(String) against MRI.
func TestOracleParse(t *testing.T) {
	bin := rubyBin(t)
	for _, in := range []string{"3/4", "3", "-3/4", "0.5", "1.25", "10/4", "12"} {
		mri := rubyEval(t, bin, scriptF("puts Rational(%q).inspect", in))
		got, err := Parse(in)
		if err != nil {
			t.Errorf("Parse(%q) err %v", in, err)
			continue
		}
		if mri != got.Inspect() {
			t.Errorf("Parse(%q): MRI %q got %q", in, mri, got.Inspect())
		}
	}
}

// TestOracleNumeratorDenominator checks numerator / denominator against MRI.
func TestOracleNumeratorDenominator(t *testing.T) {
	bin := rubyBin(t)
	cases := []struct{ n, d int64 }{{6, 4}, {-6, 4}, {3, 1}, {0, 5}}
	for _, c := range cases {
		mri := rubyEval(t, bin, scriptF(
			"r=Rational(%d,%d); puts r.numerator; puts r.denominator", c.n, c.d))
		l := strings.Split(mri, "\n")
		x := r(t, c.n, c.d)
		if l[0] != x.Numerator().String() {
			t.Errorf("(%d/%d).numerator: MRI %q got %v", c.n, c.d, l[0], x.Numerator())
		}
		if l[1] != x.Denominator().String() {
			t.Errorf("(%d/%d).denominator: MRI %q got %v", c.n, c.d, l[1], x.Denominator())
		}
	}
}

// --- small oracle helpers (kept ruby-free so they need no separate gate) ---

func scriptF(format string, a ...any) string { return sprintf(format, a...) }
