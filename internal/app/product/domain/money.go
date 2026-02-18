package domain

import (
	"fmt"
	"math/big"
)

// Money represents a monetary value with precise decimal arithmetic.
// It uses big.Rat internally to avoid floating-point precision issues.
// Money is immutable - all operations return new instances.
type Money struct {
	amount *big.Rat
}

// NewMoney creates a new Money instance from numerator and denominator.
// For example: NewMoney(1999, 100) represents $19.99
func NewMoney(numerator, denominator int64) *Money {
	if denominator == 0 {
		panic("money: denominator cannot be zero")
	}
	return &Money{
		amount: big.NewRat(numerator, denominator),
	}
}

// NewMoneyFromDecimal creates Money from a decimal string.
// For example: "19.99", "100.00", "0.01"
func NewMoneyFromDecimal(decimal string) (*Money, error) {
	rat := new(big.Rat)
	if _, ok := rat.SetString(decimal); !ok {
		return nil, fmt.Errorf("invalid decimal format: %s", decimal)
	}
	return &Money{amount: rat}, nil
}

// NewMoneyFromRat creates Money from an existing big.Rat.
// The rat is copied to ensure immutability.
func NewMoneyFromRat(rat *big.Rat) *Money {
	if rat == nil {
		return &Money{amount: big.NewRat(0, 1)}
	}
	return &Money{
		amount: new(big.Rat).Set(rat),
	}
}

// Zero returns a Money instance representing zero.
func Zero() *Money {
	return &Money{amount: big.NewRat(0, 1)}
}

// Add returns a new Money that is the sum of m and other.
func (m *Money) Add(other *Money) *Money {
	result := new(big.Rat).Add(m.amount, other.amount)
	return &Money{amount: result}
}

// Subtract returns a new Money that is the difference of m and other.
func (m *Money) Subtract(other *Money) *Money {
	result := new(big.Rat).Sub(m.amount, other.amount)
	return &Money{amount: result}
}

// Multiply returns a new Money that is the product of m and other.
func (m *Money) Multiply(other *Money) *Money {
	result := new(big.Rat).Mul(m.amount, other.amount)
	return &Money{amount: result}
}

// MultiplyByDecimal multiplies Money by a decimal value (e.g., for percentage calculations).
// For example: money.MultiplyByDecimal(0.20) calculates 20% of the amount.
func (m *Money) MultiplyByDecimal(decimal float64) *Money {
	multiplier := new(big.Rat).SetFloat64(decimal)
	result := new(big.Rat).Mul(m.amount, multiplier)
	return &Money{amount: result}
}

// MultiplyByFraction multiplies Money by a fraction (numerator/denominator).
// This is more precise than MultiplyByDecimal for exact fractions.
func (m *Money) MultiplyByFraction(numerator, denominator int64) *Money {
	multiplier := big.NewRat(numerator, denominator)
	result := new(big.Rat).Mul(m.amount, multiplier)
	return &Money{amount: result}
}

// IsZero returns true if the money amount is zero.
func (m *Money) IsZero() bool {
	return m.amount.Cmp(big.NewRat(0, 1)) == 0
}

// IsNegative returns true if the money amount is negative.
func (m *Money) IsNegative() bool {
	return m.amount.Cmp(big.NewRat(0, 1)) < 0
}

// IsPositive returns true if the money amount is positive.
func (m *Money) IsPositive() bool {
	return m.amount.Cmp(big.NewRat(0, 1)) > 0
}

// GreaterThan returns true if m is greater than other.
func (m *Money) GreaterThan(other *Money) bool {
	return m.amount.Cmp(other.amount) > 0
}

// LessThan returns true if m is less than other.
func (m *Money) LessThan(other *Money) bool {
	return m.amount.Cmp(other.amount) < 0
}

// Equals returns true if m equals other.
func (m *Money) Equals(other *Money) bool {
	if other == nil {
		return false
	}
	return m.amount.Cmp(other.amount) == 0
}

// Numerator returns the numerator of the internal rational representation.
// Used for database persistence.
func (m *Money) Numerator() int64 {
	return m.amount.Num().Int64()
}

// Denominator returns the denominator of the internal rational representation.
// Used for database persistence.
func (m *Money) Denominator() int64 {
	return m.amount.Denom().Int64()
}

// Rat returns a copy of the internal big.Rat.
// The returned value is a copy to maintain immutability.
func (m *Money) Rat() *big.Rat {
	return new(big.Rat).Set(m.amount)
}

// Float64 returns the money amount as a float64.
// Note: This may lose precision and should only be used for display purposes.
func (m *Money) Float64() float64 {
	f, _ := m.amount.Float64()
	return f
}

// String returns a string representation of the money amount.
// Format: "numerator/denominator" (e.g., "1999/100" for $19.99)
func (m *Money) String() string {
	return m.amount.FloatString(2)
}

// FloatString returns a decimal string representation with the specified precision.
// For example: FloatString(2) returns "19.99" for $19.99
func (m *Money) FloatString(precision int) string {
	return m.amount.FloatString(precision)
}
