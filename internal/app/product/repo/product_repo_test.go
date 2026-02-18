package repo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/murkotick/product-catalog-service/internal/app/product/domain"
	"github.com/murkotick/product-catalog-service/internal/models/m_product"
)

// TestInsertMut_NoDiscount verifies InsertMut for a product without a discount.
func TestInsertMut_NoDiscount(t *testing.T) {
	r := NewProductRepo()

	now := time.Now().UTC()
	base := domain.NewMoney(1999, 100) // $19.99

	// NewProduct signature: NewProduct(id, name, description, category string, basePrice *Money, now time.Time)
	p, err := domain.NewProduct("prod-no-discount", "Test Product", "a description", "electronics", base, now)
	require.NoError(t, err)

	// Inspect values map (test-friendly)
	values := buildInsertValues(p)
	require.NotNil(t, values)

	// base price fields should be present and equal to Money Numerator/Denominator
	numVal, ok := values[m_product.ColBasePriceNumerator]
	require.True(t, ok, "base price numerator missing")
	denVal, ok := values[m_product.ColBasePriceDenominator]
	require.True(t, ok, "base price denominator missing")

	assert.Equal(t, base.Numerator(), numVal)
	assert.Equal(t, base.Denominator(), denVal)

	// Discount columns should be present in map and be nil (no discount)
	if v, ok := values[m_product.ColDiscountPercent]; ok {
		assert.Nil(t, v)
	} else {
		// BuildInsertMap ensures keys exist; but just in case assert fail
		t.Fatalf("expected key %s in insert map", m_product.ColDiscountPercent)
	}

	if v, ok := values[m_product.ColDiscountStartDate]; ok {
		assert.Nil(t, v)
	} else {
		t.Fatalf("expected key %s in insert map", m_product.ColDiscountStartDate)
	}

	if v, ok := values[m_product.ColDiscountEndDate]; ok {
		assert.Nil(t, v)
	} else {
		t.Fatalf("expected key %s in insert map", m_product.ColDiscountEndDate)
	}

	// Also sanity-check InsertMut returns a non-nil mutation
	mut := r.InsertMut(p)
	require.NotNil(t, mut)
}

// TestInsertMut_WithDiscount verifies InsertMut when the product contains a Discount.
func TestInsertMut_WithDiscount(t *testing.T) {
	r := NewProductRepo()

	now := time.Now().UTC()
	base := domain.NewMoney(2000, 100) // $20.00

	// Create a discount: NewDiscount(percentage float64 (0-100), start, end)
	start := now.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)
	discount, err := domain.NewDiscount(25.0, start, end) // 25%
	require.NoError(t, err)

	// Reconstruct a product with discount present; use status active for realism
	p := domain.ReconstructProduct("prod-with-discount", "Discounted", "desc", "gadgets", base, discount, domain.ProductStatusActive, now, now, nil)

	values := buildInsertValues(p)
	require.NotNil(t, values)

	// discount percent stored by InsertMut uses precise [0,1] fraction string
	expectedDiscountStr := discount.PercentageRat().FloatString(10)

	actual, ok := values[m_product.ColDiscountPercent]
	require.True(t, ok, "discount_percent missing in insert values")
	require.NotNil(t, actual, "expected non-nil discount_percent when discount present")

	assert.Equal(t, expectedDiscountStr, actual)

	// Sanity-check InsertMut returns a non-nil mutation
	mut := r.InsertMut(p)
	require.NotNil(t, mut)
}
