package services

import (
	"time"

	"github.com/murkotick/product-catalog-service/internal/app/product/domain"
)

// PricingCalculator is a domain service that handles complex pricing calculations.
// Domain services are used when business logic doesn't naturally fit within a single aggregate.
type PricingCalculator struct{}

// NewPricingCalculator creates a new PricingCalculator instance.
func NewPricingCalculator() *PricingCalculator {
	return &PricingCalculator{}
}

// CalculateEffectivePrice calculates the final price for a product considering discounts.
// This is a simple implementation for the current requirements, but could be extended
// to handle more complex scenarios like:
// - Multiple discount tiers
// - Quantity-based pricing
// - Customer-specific pricing
// - Seasonal pricing rules
func (pc *PricingCalculator) CalculateEffectivePrice(
	basePrice *domain.Money,
	discount *domain.Discount,
	now time.Time,
) *domain.Money {
	// If no discount exists, return base price
	if discount == nil {
		return basePrice
	}

	// If discount exists but is not valid at the current time, return base price
	if !discount.IsValidAt(now) {
		return basePrice
	}

	// Apply the discount
	return discount.ApplyTo(basePrice)
}

// CalculateSavings calculates how much money is saved with a discount.
func (pc *PricingCalculator) CalculateSavings(
	basePrice *domain.Money,
	discount *domain.Discount,
	now time.Time,
) *domain.Money {
	if discount == nil || !discount.IsValidAt(now) {
		return domain.Zero()
	}

	effectivePrice := discount.ApplyTo(basePrice)
	return basePrice.Subtract(effectivePrice)
}

// CalculateSavingsPercentage calculates the percentage saved with a discount.
// Returns a value between 0.0 and 1.0 (e.g., 0.20 for 20% savings).
func (pc *PricingCalculator) CalculateSavingsPercentage(
	basePrice *domain.Money,
	discount *domain.Discount,
	now time.Time,
) float64 {
	if discount == nil || !discount.IsValidAt(now) {
		return 0.0
	}

	// The savings percentage is simply the discount percentage
	return discount.Percentage() / 100.0
}
