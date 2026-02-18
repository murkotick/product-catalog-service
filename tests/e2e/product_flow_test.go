package e2e

import (
	"context"
	"math/big"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/murkotick/product-catalog-service/internal/app/product/domain"
	"github.com/murkotick/product-catalog-service/internal/app/product/queries/get_product"
	"github.com/murkotick/product-catalog-service/internal/app/product/queries/list_products"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/activate_product"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/apply_discount"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/create_product"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/update_product"
)

func TestProductCreationFlow(t *testing.T) {
	requireEmulator(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	productID, err := createUC.Execute(ctx, create_product.Request{
		Name:         "Test Product",
		Description:  "A product for E2E tests",
		Category:     "books",
		BasePriceNum: 1999,
		BasePriceDen: 100,
	})
	require.NoError(t, err)
	require.NotEmpty(t, productID)

	getQ := get_product.NewHandler(readModel)
	prod, err := getQ.Execute(ctx, productID)
	require.NoError(t, err)

	assert.Equal(t, "Test Product", prod.Name)
	assert.Equal(t, "books", prod.Category)
	assert.Equal(t, "draft", prod.Status)
	assert.Equal(t, "19.9900000000", prod.EffectivePrice)

	events := mustFetchOutboxEvents(ctx, t, spClient, productID)
	require.Len(t, events, 1)
	assert.Equal(t, "product.created", events[0].EventType)
	assert.Equal(t, "pending", events[0].Status)
}

func TestDiscountApplicationFlow(t *testing.T) {
	requireEmulator(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	productID, err := createUC.Execute(ctx, create_product.Request{
		Name:         "Discounted Product",
		Description:  "",
		Category:     "electronics",
		BasePriceNum: 10000,
		BasePriceDen: 100,
	})
	require.NoError(t, err)

	// Activate first (discounts only allowed on active products).
	require.NoError(t, activateUC.Execute(ctx, activate_product.Request{ProductID: productID}))

	// Make discount active at real "now" (queries compute effective price using time.Now()).
	now := time.Now().UTC()
	start := now.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)

	err = applyDisUC.Execute(ctx, apply_discount.Request{
		ProductID:  productID,
		Percentage: 20, // 20% off
		StartDate:  start,
		EndDate:    end,
	})
	require.NoError(t, err)

	// Verify effective price.
	getQ := get_product.NewHandler(readModel)
	prod, err := getQ.Execute(ctx, productID)
	require.NoError(t, err)

	// 100.00 - 20% = 80.00
	assert.Equal(t, "80.0000000000", prod.EffectivePrice)

	// Also verify via list query (active products).
	listQ := list_products.NewHandler(readModel)
	items, err := listQ.Execute(ctx, nil, 10, 0)
	require.NoError(t, err)
	found := false
	for _, it := range items {
		if it.ProductID == productID {
			found = true
			assert.Equal(t, "80.0000000000", it.EffectivePrice)
			break
		}
	}
	require.True(t, found, "created product must be present in list")
}

func TestBusinessRuleValidation_CannotApplyDiscountToInactiveProduct(t *testing.T) {
	requireEmulator(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	productID, err := createUC.Execute(ctx, create_product.Request{
		Name:         "Inactive Product",
		Description:  "",
		Category:     "books",
		BasePriceNum: 5000,
		BasePriceDen: 100,
	})
	require.NoError(t, err)

	now := time.Now().UTC()
	err = applyDisUC.Execute(ctx, apply_discount.Request{
		ProductID:  productID,
		Percentage: 10,
		StartDate:  now.Add(-1 * time.Hour),
		EndDate:    now.Add(1 * time.Hour),
	})
	assert.ErrorIs(t, err, domain.ErrProductNotActive)

	// Ensure no discount fields were persisted.
	stmt := spanner.Statement{
		SQL:    "SELECT CAST(discount_percent AS STRING) FROM products WHERE product_id = @id",
		Params: map[string]interface{}{"id": productID},
	}
	iter := spClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	require.NoError(t, err)
	var dp spanner.NullString
	require.NoError(t, row.Columns(&dp))
	assert.False(t, dp.Valid)
}

func TestProductUpdateFlow_CreatesOutboxEvent(t *testing.T) {
	requireEmulator(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	productID, err := createUC.Execute(ctx, create_product.Request{
		Name:         "Old Name",
		Description:  "Old Desc",
		Category:     "books",
		BasePriceNum: 2500,
		BasePriceDen: 100,
	})
	require.NoError(t, err)

	newName := "New Name"
	newCat := "stationery"
	require.NoError(t, updateUC.Execute(ctx, update_product.Request{
		ProductID: productID,
		Name:      &newName,
		Category:  &newCat,
	}))

	getQ := get_product.NewHandler(readModel)
	prod, err := getQ.Execute(ctx, productID)
	require.NoError(t, err)
	assert.Equal(t, "New Name", prod.Name)
	assert.Equal(t, "stationery", prod.Category)

	events := mustFetchOutboxEvents(ctx, t, spClient, productID)
	// product.created + product.updated
	require.GreaterOrEqual(t, len(events), 2)
	assert.Equal(t, "product.created", events[0].EventType)
	assert.Equal(t, "product.updated", events[1].EventType)
}

func TestEffectivePriceMathMatchesBigRat(t *testing.T) {
	requireEmulator(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	productID, err := createUC.Execute(ctx, create_product.Request{
		Name:         "Rat Math Product",
		Description:  "",
		Category:     "books",
		BasePriceNum: 1999,
		BasePriceDen: 100,
	})
	require.NoError(t, err)
	require.NoError(t, activateUC.Execute(ctx, activate_product.Request{ProductID: productID}))

	now := time.Now().UTC()
	start := now.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)
	require.NoError(t, applyDisUC.Execute(ctx, apply_discount.Request{
		ProductID:  productID,
		Percentage: 20,
		StartDate:  start,
		EndDate:    end,
	}))

	getQ := get_product.NewHandler(readModel)
	prod, err := getQ.Execute(ctx, productID)
	require.NoError(t, err)

	base := new(big.Rat).SetFrac(big.NewInt(1999), big.NewInt(100))
	percent := new(big.Rat).SetFrac(big.NewInt(20), big.NewInt(100))
	expected := new(big.Rat).Sub(base, new(big.Rat).Mul(base, percent))
	assert.Equal(t, expected.FloatString(10), prod.EffectivePrice)
}
