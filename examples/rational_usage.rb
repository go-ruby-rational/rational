# frozen_string_literal: true
#
# Usage of Rational — the exact numerator/denominator number type built into
# the numeric tower (no require needed). Every value is reduced to lowest terms
# with a positive denominator. Runs under go-embedded-ruby (rbgo); see
# examples/README.md.

# Construct with the Rational() kernel method; components are normalised.
a = Rational(3, 4)
b = Rational(2, 4)
p a                            # => (3/4)
p b                            # => (1/2)
p Rational(1, -2)              # => (-1/2)   sign folds onto the numerator

# Exact arithmetic stays a Rational; a Float operand promotes to Float.
p a + b                        # => (5/4)
p a - b                        # => (1/4)
p a * b                        # => (3/8)
p a / b                        # => (3/2)
p a % b                        # => (1/4)
p a ** 2                       # => (9/16)
p a + 0.25                     # => 1.0

# Mixed with Integer keeps it exact.
p a + 1                        # => (7/4)
p 2 * a                        # => (3/2)

# Parts, comparison and conversions.
p a.numerator                  # => 3
p a.denominator                # => 4
p a <=> b                      # => 1
p a == Rational(6, 8)          # => true
p Rational(-3, 4).abs          # => (3/4)
p a.to_f                       # => 0.75
p a.to_i                       # => 0
