package domain

import (
	"fmt"
	"math/big"
	"time"
)

// Discount represents a percentage-based discount with a validity period.
// Discount is immutable once created.
type Discount struct {
	percentage *big.Rat
	startDate  time.Time
	endDate    time.Time
}

// NewDiscount creates a new Discount with the given percentage and date range.
// percentage should be between 0 and 100 (e.g., 20 for 20% off).
// Returns an error if the percentage is invalid or date range is invalid.
func NewDiscount(percentage float64, startDate, endDate time.Time) (*Discount, error) {
	if percentage < 0 || percentage > 100 {
		return nil, ErrInvalidDiscountPercentage
	}

	if endDate.Before(startDate) {
		return nil, ErrInvalidDiscountPeriod
	}

	if startDate.Equal(endDate) {
		return nil, ErrInvalidDiscountPeriod
	}

	return &Discount{
		percentage: big.NewRat(int64(percentage*100), 10000), // Store as precise fraction
		startDate:  startDate,
		endDate:    endDate,
	}, nil
}

// NewDiscountFromRat creates a Discount with percentage as a big.Rat (0.0 to 1.0).
// For example: 0.20 for 20% off.
func NewDiscountFromRat(percentageRat *big.Rat, startDate, endDate time.Time) (*Discount, error) {
	// Convert to 0-100 scale for validation
	hundred := big.NewRat(100, 1)
	percentage := new(big.Rat).Mul(percentageRat, hundred)
	percentageFloat, _ := percentage.Float64()

	if percentageFloat < 0 || percentageFloat > 100 {
		return nil, ErrInvalidDiscountPercentage
	}

	if endDate.Before(startDate) {
		return nil, ErrInvalidDiscountPeriod
	}

	if startDate.Equal(endDate) {
		return nil, ErrInvalidDiscountPeriod
	}

	return &Discount{
		percentage: new(big.Rat).Set(percentageRat),
		startDate:  startDate,
		endDate:    endDate,
	}, nil
}

// IsValidAt checks if the discount is valid at the given time.
// A discount is valid if the time is within [startDate, endDate).
func (d *Discount) IsValidAt(now time.Time) bool {
	return !now.Before(d.startDate) && now.Before(d.endDate)
}

// IsActive is an alias for IsValidAt for better readability in some contexts.
func (d *Discount) IsActive(now time.Time) bool {
	return d.IsValidAt(now)
}

// Percentage returns the discount percentage as a float64 (0-100 scale).
// For example: 20.0 for 20% off.
func (d *Discount) Percentage() float64 {
	hundred := big.NewRat(100, 1)
	percentage := new(big.Rat).Mul(d.percentage, hundred)
	result, _ := percentage.Float64()
	return result
}

// PercentageRat returns the discount percentage as a big.Rat (0.0-1.0 scale).
// For example: 0.20 for 20% off.
// Returns a copy to maintain immutability.
func (d *Discount) PercentageRat() *big.Rat {
	return new(big.Rat).Set(d.percentage)
}

// StartDate returns the start date of the discount validity period.
func (d *Discount) StartDate() time.Time {
	return d.startDate
}

// EndDate returns the end date of the discount validity period.
func (d *Discount) EndDate() time.Time {
	return d.endDate
}

// CalculateDiscountAmount calculates the discount amount for a given price.
// Returns a new Money instance representing the discount amount.
func (d *Discount) CalculateDiscountAmount(price *Money) *Money {
	return price.Multiply(NewMoneyFromRat(d.percentage))
}

// ApplyTo applies the discount to a given price and returns the final price.
// Returns a new Money instance representing the discounted price.
func (d *Discount) ApplyTo(price *Money) *Money {
	discountAmount := d.CalculateDiscountAmount(price)
	return price.Subtract(discountAmount)
}

// String returns a string representation of the discount.
func (d *Discount) String() string {
	return fmt.Sprintf("%.2f%% off (valid from %s to %s)",
		d.Percentage(),
		d.startDate.Format("2006-01-02"),
		d.endDate.Format("2006-01-02"))
}
