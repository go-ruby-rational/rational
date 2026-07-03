// Copyright (c) the go-ruby-rational/rational authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rational

import (
	"math"
	"math/bits"
)

// This file holds the int64 "fast path": the machine-word representation and the
// overflow-checked primitives the arithmetic operators use before promoting to
// *big.Rat. Every helper detects overflow exactly (via math/bits 128-bit
// products and sign-aware sum checks) so a fast-path result is either bit-exact
// or the caller falls back to the big.Rat path — precision is never lost.

// magU returns the magnitude |x| as a uint64. It is correct even for
// math.MinInt64, whose magnitude (1<<63) does not fit an int64.
func magU(x int64) uint64 {
	u := uint64(x)
	if x < 0 {
		u = -u
	}
	return u
}

// gcdU is the binary-free Euclidean gcd on magnitudes. gcdU(0, d) == d, so a
// zero numerator reduces cleanly against its denominator.
func gcdU(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// checkedMul returns a*b and whether it fits an int64. It multiplies magnitudes
// with a full 128-bit product (bits.Mul64) and rejects any result whose high word
// is non-zero or whose low word exceeds the signed range for its sign.
func checkedMul(a, b int64) (int64, bool) {
	hi, lo := bits.Mul64(magU(a), magU(b))
	if hi != 0 {
		return 0, false
	}
	if (a < 0) != (b < 0) { // negative result
		if lo > uint64(math.MaxInt64)+1 {
			return 0, false
		}
		return -int64(lo), true // lo == 1<<63 negates to math.MinInt64, which is correct
	}
	if lo > uint64(math.MaxInt64) {
		return 0, false
	}
	return int64(lo), true
}

// checkedAdd returns a+b and whether it fits an int64. Overflow can only occur
// when the operands share a sign and the sum's sign flips.
func checkedAdd(a, b int64) (int64, bool) {
	c := a + b
	if (a > 0 && b > 0 && c < 0) || (a < 0 && b < 0 && c >= 0) {
		return 0, false
	}
	return c, true
}

// checkedNeg returns -a and whether it fits an int64 (only math.MinInt64 fails).
func checkedNeg(a int64) (int64, bool) {
	if a == math.MinInt64 {
		return 0, false
	}
	return -a, true
}

// newFrac64 builds a fast-path Rational from num/den (den > 0, possibly not in
// lowest terms), reducing by their gcd so the result is canonical (Ruby always
// stores a Rational in lowest terms). num/den only shrink under reduction, so no
// overflow is possible here. It is the shared tail of the fast arithmetic paths
// and of the fast decimal parser.
func newFrac64(num, den int64) *Rational {
	g := int64(gcdU(magU(num), uint64(den)))
	if g > 1 {
		num /= g
		den /= g
	}
	return &Rational{small: true, num: num, den: den}
}

// addSmall computes a±b entirely in int64 (sub selects subtraction), returning
// false on any overflow so the caller can fall back to *big.Rat. It follows
// MRI's scheme: reduce the denominators by their gcd first (bounding the
// intermediate products), combine, then reduce the result to lowest terms.
func addSmall(a, b *Rational, sub bool) (*Rational, bool) {
	g := int64(gcdU(uint64(a.den), uint64(b.den)))
	da := a.den / g // a.den / gcd
	db := b.den / g // b.den / gcd
	t1, ok1 := checkedMul(a.num, db)
	t2, ok2 := checkedMul(b.num, da)
	if !ok1 || !ok2 {
		return nil, false
	}
	if sub {
		var ok bool
		if t2, ok = checkedNeg(t2); !ok {
			return nil, false
		}
	}
	num, ok := checkedAdd(t1, t2)
	if !ok {
		return nil, false
	}
	den, ok := checkedMul(a.den, db) // == lcm(a.den, b.den) > 0
	if !ok {
		return nil, false
	}
	return newFrac64(num, den), true
}

// mulSmall computes a*b in int64 using MRI's cross-cancellation (gcd the
// numerators against the opposite denominators before multiplying, which keeps
// the products small and yields an already-reduced result). It returns false on
// overflow. Both denominators are positive, so the product denominator is too.
func mulSmall(a, b *Rational) (*Rational, bool) {
	g1 := int64(gcdU(magU(a.num), uint64(b.den)))
	g2 := int64(gcdU(magU(b.num), uint64(a.den)))
	num, ok1 := checkedMul(a.num/g1, b.num/g2)
	den, ok2 := checkedMul(a.den/g2, b.den/g1)
	if !ok1 || !ok2 {
		return nil, false
	}
	return newFrac64(num, den), true
}

// divSmall computes a/b in int64 (b already known non-zero). It is a*reciprocal(b)
// with the same cross-cancellation as mulSmall, then normalises the sign onto the
// numerator (den > 0). It returns false on overflow.
func divSmall(a, b *Rational) (*Rational, bool) {
	g1 := int64(gcdU(magU(a.num), magU(b.num)))
	g2 := int64(gcdU(uint64(a.den), uint64(b.den)))
	num, ok1 := checkedMul(a.num/g1, b.den/g2)
	den, ok2 := checkedMul(a.den/g2, b.num/g1)
	if !ok1 || !ok2 {
		return nil, false
	}
	if den < 0 { // b was negative: move the sign to the numerator
		var okn, okd bool
		if num, okn = checkedNeg(num); !okn {
			return nil, false
		}
		if den, okd = checkedNeg(den); !okd {
			return nil, false
		}
	}
	return newFrac64(num, den), true
}

// parseDecimalFast parses a trimmed bare integer or plain decimal ("3", "-3",
// "3.14159") into num/den (den > 0) in int64, returning false — so the caller
// falls back to the big.Rat parser — on any exponent, non-digit, or magnitude
// that would overflow int64.
func parseDecimalFast(s string) (num, den int64, ok bool) {
	i := 0
	neg := false
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	var n int64
	scale := 0
	seenDot := false
	seenDigit := false
	for ; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			if seenDot {
				return 0, 0, false
			}
			seenDot = true
			continue
		}
		if c < '0' || c > '9' { // letters, 'e'/'E', signs mid-number: give up
			return 0, 0, false
		}
		seenDigit = true
		d := int64(c - '0')
		if n > (math.MaxInt64-d)/10 { // n*10 + d would overflow
			return 0, 0, false
		}
		n = n*10 + d
		if seenDot {
			scale++
		}
	}
	if !seenDigit {
		return 0, 0, false
	}
	den = 1
	for k := 0; k < scale; k++ {
		if den > math.MaxInt64/10 { // 10^scale would overflow
			return 0, 0, false
		}
		den *= 10
	}
	if neg {
		n = -n
	}
	return n, den, true
}
