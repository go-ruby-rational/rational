// Copyright (c) the go-ruby-rational/rational authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rational

import "math/big"

// Add returns a + b (Ruby's Rational#+). When both operands fit int64 the sum is
// computed in int64 and, only on overflow, promoted to the exact *big.Rat path.
func (a *Rational) Add(b *Rational) *Rational {
	if a.small && b.small {
		if r, ok := addSmall(a, b, false); ok {
			return r
		}
	}
	return wrap(new(big.Rat).Add(a.rat(), b.rat()))
}

// Sub returns a - b (Ruby's Rational#-), on the int64 fast path where possible.
func (a *Rational) Sub(b *Rational) *Rational {
	if a.small && b.small {
		if r, ok := addSmall(a, b, true); ok {
			return r
		}
	}
	return wrap(new(big.Rat).Sub(a.rat(), b.rat()))
}

// Mul returns a * b (Ruby's Rational#*), on the int64 fast path where possible.
func (a *Rational) Mul(b *Rational) *Rational {
	if a.small && b.small {
		if r, ok := mulSmall(a, b); ok {
			return r
		}
	}
	return wrap(new(big.Rat).Mul(a.rat(), b.rat()))
}

// Div returns a / b (Ruby's Rational#/). Dividing by zero returns
// ErrZeroDivision, mirroring Ruby's ZeroDivisionError. Non-zero divisions take
// the int64 fast path where possible.
func (a *Rational) Div(b *Rational) (*Rational, error) {
	if b.isZero() {
		return nil, ErrZeroDivision
	}
	if a.small && b.small {
		if r, ok := divSmall(a, b); ok {
			return r, nil
		}
	}
	return wrap(new(big.Rat).Quo(a.rat(), b.rat())), nil
}

// Neg returns -a (Ruby's Rational#-@).
func (a *Rational) Neg() *Rational {
	if a.small {
		if n, ok := checkedNeg(a.num); ok {
			return &Rational{small: true, num: n, den: a.den}
		}
	}
	return wrap(new(big.Rat).Neg(a.rat()))
}

// Abs returns |a| (Ruby's Rational#abs).
func (a *Rational) Abs() *Rational {
	if a.small {
		if a.num >= 0 {
			return &Rational{small: true, num: a.num, den: a.den}
		}
		if n, ok := checkedNeg(a.num); ok {
			return &Rational{small: true, num: n, den: a.den}
		}
	}
	return wrap(new(big.Rat).Abs(a.rat()))
}

// Reciprocal returns 1/a. A zero Rational has no reciprocal and returns
// ErrZeroDivision, mirroring Ruby's ZeroDivisionError for 0r ** -1.
func (a *Rational) Reciprocal() (*Rational, error) {
	if a.isZero() {
		return nil, ErrZeroDivision
	}
	if a.small {
		// 1/(num/den) = den/num; den is already > 0, so only a negative numerator
		// needs the sign moved onto the new denominator.
		num, den := a.den, a.num
		if den >= 0 {
			return &Rational{small: true, num: num, den: den}, nil
		}
		nn, ok1 := checkedNeg(num)
		nd, ok2 := checkedNeg(den)
		if ok1 && ok2 {
			return &Rational{small: true, num: nn, den: nd}, nil
		}
	}
	return wrap(new(big.Rat).Inv(a.rat())), nil
}

// Pow raises a to an integer power, staying exact (Ruby's Rational#** with an
// Integer exponent). A negative exponent of zero returns ErrZeroDivision,
// mirroring Ruby's ZeroDivisionError. For a Rational or Float exponent Ruby
// falls back to Float (a.to_f ** exp); use PowFloat for that.
func (a *Rational) Pow(exp *big.Int) (*Rational, error) {
	if exp.Sign() == 0 {
		// x ** 0 == 1 for every x, including 0 (Ruby: 0r ** 0 == 1).
		return FromInt64(1), nil
	}
	if a.isZero() && exp.Sign() < 0 {
		return nil, ErrZeroDivision
	}
	n := new(big.Int).Abs(exp)
	r := a.rat()
	num := new(big.Int).Exp(r.Num(), n, nil)
	den := new(big.Int).Exp(r.Denom(), n, nil)
	res := new(big.Rat).SetFrac(num, den)
	if exp.Sign() < 0 {
		res.Inv(res)
	}
	return wrap(res), nil
}

// PowFloat raises a to a real power and returns a float64, mirroring Ruby's
// Rational#** with a Rational or Float exponent (which yields a Float via
// a.to_f ** exp).
func (a *Rational) PowFloat(exp float64) float64 {
	return floatPow(a.ToF(), exp)
}
