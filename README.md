<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-rational/brand/main/social/go-ruby-rational-rational.png" alt="go-ruby-rational/rational" width="720"></p>

# rational — go-ruby-rational

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-rational.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of Ruby's
[`Rational`](https://docs.ruby-lang.org/en/master/Rational.html)** — the exact
`numerator/denominator` number type backing literals such as `3r` and the
`Rational()` conversion method — **byte-exact against MRI 4.0.5**. A `Rational`
holds an arbitrary-precision ratio in lowest terms with a positive denominator
(Ruby's normalisation: the sign lives on the numerator), and every arithmetic,
comparison and conversion method reproduces MRI's result exactly.

It is the `Rational` backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** module with no dependency on the Ruby runtime — a sibling
of [go-ruby-bigdecimal](https://github.com/go-ruby-bigdecimal/bigdecimal). Being
**MRI-faithful**, it is deliberately distinct from
[go-composites/rational](https://github.com/go-composites/rational), which models
a generic mathematical rational rather than Ruby's exact semantics.

> **Why a separate, MRI-faithful type.** The exact rational core (reduce to
> lowest terms, exact `+ - * /`, integer `**`) is the easy part; matching MRI is
> in the details — `inspect` prints `(3/4)` while `to_s` prints `3/4`; rounding is
> **half-away-from-zero** (`(5/2).round == 3`, `(-5/2).round == -3`); the
> digit-aware `floor`/`ceil`/`round`/`truncate` return a **Rational** for `n >= 1`
> but an **Integer** for `n <= 0`; and `**` stays exact for an Integer exponent
> yet falls back to **Float** for a Rational/Float exponent. This package encodes
> those rules and pins them with a differential MRI oracle.

## Features

Faithful port of `Rational`, validated against the `ruby` binary (>= 4.0) on every
supported platform:

- **Exact arithmetic** — `Add` / `Sub` / `Mul` / `Div` over arbitrary-precision
  integers (`math/big.Rat` core), always reduced to lowest terms with a positive
  denominator; `Div` by zero raises `ErrZeroDivision` (MRI `ZeroDivisionError`).
- **`Neg` / `Abs` / `Reciprocal`** — `Reciprocal` of zero raises `ErrZeroDivision`.
- **`Pow`** — integer exponent stays exact (`(3/4) ** -1 == (4/3)`, `0r ** -1`
  raises); **`PowFloat`** for the Rational/Float-exponent Float fallback.
- **Normalisation** — `Rational(2, 4) → (1/2)`, negative-denominator folding
  (`Rational(1, -2) → (-1/2)`), `Rational(0, n) → (0/1)`.
- **`ToS` vs `Inspect`** — MRI's split: `to_s → "3/4"`, `inspect → "(3/4)"`
  (`String()` returns the inspect form).
- **Conversions** — `ToI` (truncate toward zero), `ToF` (nearest-even, matching
  MRI), `ToR`, and `Truncate` / `Floor` / `Ceil` / `Round` to an Integer.
- **Digit-aware rounding** — `FloorN` / `CeilN` / `RoundN` / `TruncateN` return a
  **Rational** for `n >= 1` and an **Integer** for `n <= 0`, with MRI's
  half-away-from-zero rounding.
- **Comparison** — `Cmp` (`<=>`), `Eql` (`==`) and `EqlStrict` (`eql?`, which is
  true only for an equal Rational) against Rational, Integer (`CmpInt`/`EqlInt`)
  and Float (`CmpFloat`/`EqlFloat`, with `NaN` → undefined like MRI's `nil`).
- **`Parse`** — `Rational(String)` semantics: surrounding whitespace, a sign, a
  bare integer (`"3"`), a fraction (`"10/4" → (5/2)`) or a decimal/scientific
  literal (`"1.25" → (5/4)`, `"3e2" → (300/1)`).
- **`Rationalize`** — returns the Rational itself (already exact).

CGO-free, dependency-free, **100% test coverage**, `gofmt` + `go vet` clean, and
green across the six 64-bit Go targets (amd64, arm64, riscv64, loong64, ppc64le,
s390x) and three OSes (Linux, macOS, Windows).

## Install

```sh
go get github.com/go-ruby-rational/rational
```

## Usage

```go
package main

import (
	"fmt"
	"math/big"

	"github.com/go-ruby-rational/rational"
)

func main() {
	a, _ := rational.New(big.NewInt(1), big.NewInt(3)) // (1/3)
	b, _ := rational.New(big.NewInt(1), big.NewInt(6)) // (1/6)

	fmt.Println(a.Add(b).Inspect()) // (1/2)      — Rational#inspect
	fmt.Println(a.Add(b).ToS())     // 1/2        — Rational#to_s

	r, _ := rational.Parse("10/4")  // Rational("10/4")
	fmt.Println(r.Inspect())        // (5/2)

	p, _ := r.Pow(big.NewInt(-1))   // (5/2) ** -1
	fmt.Println(p.Inspect())        // (2/5)

	fmt.Println(rational.FromInt64(5).
		Mul(r).Inspect())           // (25/2)

	// Digit-aware rounding mirrors MRI: n >= 1 → Rational, n <= 0 → Integer.
	q, _, _ := r.Mul(a).RoundN(2)   // (5/6).round(2)
	fmt.Println(q.Inspect())        // (83/100)
}
```

## API

```go
// Construction
func New(num, den *big.Int) (*Rational, error) // Rational(num, den); den==0 → ErrZeroDivision
func FromInt(n *big.Int) *Rational              // Rational(n)
func FromInt64(n int64) *Rational
func Parse(s string) (*Rational, error)         // Rational(String)

// Arithmetic
func (a *Rational) Add(b *Rational) *Rational
func (a *Rational) Sub(b *Rational) *Rational
func (a *Rational) Mul(b *Rational) *Rational
func (a *Rational) Div(b *Rational) (*Rational, error) // /0 → ErrZeroDivision
func (a *Rational) Neg() *Rational
func (a *Rational) Abs() *Rational
func (a *Rational) Reciprocal() (*Rational, error)     // 1/a; 1/0 → ErrZeroDivision
func (a *Rational) Pow(exp *big.Int) (*Rational, error) // exact; 0**(-n) → ErrZeroDivision
func (a *Rational) PowFloat(exp float64) float64        // Float fallback (a.to_f ** exp)

// Components
func (a *Rational) Numerator() *big.Int   // sign lives here
func (a *Rational) Denominator() *big.Int // always positive

// Conversion & rounding
func (a *Rational) ToI() *big.Int   // truncate toward zero
func (a *Rational) ToF() float64
func (a *Rational) ToR() *Rational
func (a *Rational) Truncate() *big.Int
func (a *Rational) Floor() *big.Int
func (a *Rational) Ceil() *big.Int
func (a *Rational) Round() *big.Int // half away from zero
func (a *Rational) FloorN(n int) (rat *Rational, i *big.Int, isRat bool)    // n>=1 → Rational
func (a *Rational) CeilN(n int) (rat *Rational, i *big.Int, isRat bool)
func (a *Rational) RoundN(n int) (rat *Rational, i *big.Int, isRat bool)
func (a *Rational) TruncateN(n int) (rat *Rational, i *big.Int, isRat bool)
func (a *Rational) Rationalize() *Rational

// Comparison
func (a *Rational) Cmp(b *Rational) int             // <=>
func (a *Rational) CmpInt(n *big.Int) int
func (a *Rational) CmpFloat(f float64) (c int, ok bool) // ok=false for NaN (MRI nil)
func (a *Rational) Eql(b *Rational) bool            // ==
func (a *Rational) EqlInt(n *big.Int) bool          // (3/1) == 3
func (a *Rational) EqlFloat(f float64) bool         // (1/2) == 0.5
func (a *Rational) EqlStrict(b *Rational) bool      // eql? — Rational only

// Rendering
func (a *Rational) ToS() string    // "3/4"
func (a *Rational) Inspect() string // "(3/4)"
func (a *Rational) String() string  // inspect form (fmt.Stringer)

var ErrZeroDivision  error // ZeroDivisionError ("divided by 0")
var ErrInvalidArgument error // ArgumentError (unparseable Rational string)
```

## Notes on MRI fidelity

`Pow` with an **integer** exponent is computed exactly. For a **Rational or
Float** exponent MRI returns a `Float` (`a.to_f ** exp`); `PowFloat` provides that
fallback via `math.Pow`. Because Go's `math.Pow` and MRI's C `pow` are independent
libm implementations, a perfect-root case (e.g. `8.0 ** (1.0/3.0)`) can differ in
the last ULP across platforms — the oracle therefore exercises `Pow` (exact) and
only the libm-stable `PowFloat` cases.

## Tests & coverage

The suite pairs deterministic, ruby-free tests (which alone hold coverage at
**100%**, so the qemu cross-arch and Windows lanes pass the gate) with a
**differential MRI oracle**: a wide corpus of values is evaluated both here and by
the system `ruby` and the inspected results compared byte-for-byte. The oracle
gates on `RUBY_VERSION >= "4.0"` and `$stdout.binmode` + `STDIN.binmode` so Windows
text-mode never pollutes the bytes; it skips itself where `ruby` is absent.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-rational/rational authors.
