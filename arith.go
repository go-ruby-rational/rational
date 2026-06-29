// Copyright (c) the go-ruby-rational/rational authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rational

import "math/big"

// Add returns a + b (Ruby's Rational#+).
func (a *Rational) Add(b *Rational) *Rational {
	return wrap(new(big.Rat).Add(a.r, b.r))
}

// Sub returns a - b (Ruby's Rational#-).
func (a *Rational) Sub(b *Rational) *Rational {
	return wrap(new(big.Rat).Sub(a.r, b.r))
}

// Mul returns a * b (Ruby's Rational#*).
func (a *Rational) Mul(b *Rational) *Rational {
	return wrap(new(big.Rat).Mul(a.r, b.r))
}

// Div returns a / b (Ruby's Rational#/). Dividing by zero returns
// ErrZeroDivision, mirroring Ruby's ZeroDivisionError.
func (a *Rational) Div(b *Rational) (*Rational, error) {
	if b.r.Sign() == 0 {
		return nil, ErrZeroDivision
	}
	return wrap(new(big.Rat).Quo(a.r, b.r)), nil
}

// Neg returns -a (Ruby's Rational#-@).
func (a *Rational) Neg() *Rational {
	return wrap(new(big.Rat).Neg(a.r))
}

// Abs returns |a| (Ruby's Rational#abs).
func (a *Rational) Abs() *Rational {
	return wrap(new(big.Rat).Abs(a.r))
}

// Reciprocal returns 1/a. A zero Rational has no reciprocal and returns
// ErrZeroDivision, mirroring Ruby's ZeroDivisionError for 0r ** -1.
func (a *Rational) Reciprocal() (*Rational, error) {
	if a.r.Sign() == 0 {
		return nil, ErrZeroDivision
	}
	return wrap(new(big.Rat).Inv(a.r)), nil
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
	if a.r.Sign() == 0 && exp.Sign() < 0 {
		return nil, ErrZeroDivision
	}
	n := new(big.Int).Abs(exp)
	num := new(big.Int).Exp(a.r.Num(), n, nil)
	den := new(big.Int).Exp(a.r.Denom(), n, nil)
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
