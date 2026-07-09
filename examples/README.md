# rational examples

Runnable pure-Ruby usage of the built-in `Rational` exact numerator/denominator number type, verified under the [rbgo](https://github.com/go-embedded-ruby) interpreter.

```sh
rbgo examples/rational_usage.rb
```

| File | Shows |
| --- | --- |
| `rational_usage.rb` | Build values with `Rational()` (normalised to lowest terms, sign on the numerator), exact `+ - * / % **` arithmetic that stays a `Rational` while a `Float` operand promotes to `Float`, mixing with `Integer`, and `numerator` / `denominator` / `<=>` / `==` / `abs` / `to_f` / `to_i`. |
