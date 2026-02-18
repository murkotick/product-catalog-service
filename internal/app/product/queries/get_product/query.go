package get_product

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/murkotick/product-catalog-service/internal/app/product/dto"
)

// SpannerGetProductQuery is a concrete query implementation that reads from Spanner directly.
type SpannerGetProductQuery struct {
	Client *spanner.Client
}

func NewSpannerGetProductQuery(client *spanner.Client) *SpannerGetProductQuery {
	return &SpannerGetProductQuery{Client: client}
}

// GetProduct executes a SQL query to fetch a product row and compute the effective price.
func (q *SpannerGetProductQuery) GetProduct(ctx context.Context, productID string) (*dto.ProductDTO, error) {
	stmt := spanner.Statement{
		SQL: `SELECT product_id, name, description, category,
		             base_price_numerator, base_price_denominator,
		             discount_percent, discount_start_date, discount_end_date,
		             status, created_at, updated_at, archived_at
		      FROM products
		      WHERE product_id = @id`,
		Params: map[string]interface{}{"id": productID},
	}

	iter := q.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, spanner.ErrRowNotFound
	}
	if err != nil {
		return nil, err
	}

	var (
		id                         string
		name                       string
		description                spanner.NullString
		category                   string
		baseNum                    int64
		baseDen                    int64
		discountPercent            spanner.NullString
		discountStart, discountEnd spanner.NullTime
		status                     string
		createdAt, updatedAt       time.Time
		archivedAt                 spanner.NullTime
	)

	if err := row.Columns(&id, &name, &description, &category, &baseNum, &baseDen,
		&discountPercent, &discountStart, &discountEnd, &status, &createdAt, &updatedAt, &archivedAt); err != nil {
		return nil, err
	}

	dtoOut := &dto.ProductDTO{
		ProductID:    id,
		Name:         name,
		Category:     category,
		BasePriceNum: baseNum,
		BasePriceDen: baseDen,
		Status:       status,
	}

	if description.Valid {
		desc := description.StringVal
		dtoOut.Description = &desc
	}

	if discountPercent.Valid {
		dp := discountPercent.StringVal
		dtoOut.DiscountPct = &dp
	}

	if discountStart.Valid {
		ds := discountStart.Time.UTC().Format(time.RFC3339)
		dtoOut.DiscountStart = &ds
	}
	if discountEnd.Valid {
		de := discountEnd.Time.UTC().Format(time.RFC3339)
		dtoOut.DiscountEnd = &de
	}

	// timestamps
	c := createdAt.UTC().Format(time.RFC3339)
	dtoOut.CreatedAt = &c
	u := updatedAt.UTC().Format(time.RFC3339)
	dtoOut.UpdatedAt = &u
	if archivedAt.Valid {
		aa := archivedAt.Time.UTC().Format(time.RFC3339)
		dtoOut.ArchivedAt = &aa
	}

	// Compute effective price based on discount validity now (UTC).
	effective, err := computeEffectivePrice(baseNum, baseDen, discountPercent, discountStart, discountEnd, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	dtoOut.EffectivePrice = effective.FloatString(10)

	return dtoOut, nil
}

// computeEffectivePrice returns the effective price as *big.Rat
func computeEffectivePrice(baseNum, baseDen int64, discountPercent spanner.NullString, start, end spanner.NullTime, now time.Time) (*big.Rat, error) {
	base := new(big.Rat).SetFrac(big.NewInt(baseNum), big.NewInt(baseDen))

	// no discount present
	if !discountPercent.Valid || discountPercent.StringVal == "" {
		return base, nil
	}

	// check validity window (start inclusive, end inclusive)
	if start.Valid && now.Before(start.Time) {
		return base, nil
	}
	if end.Valid && now.After(end.Time) { // now > end => expired
		return base, nil
	}

	// discountPercent.StringVal is held as decimal string (NUMERIC) or percentage string.
	// Try big.Rat parse first (handles "0.25" or "0.20"), if that fails, try to parse as float percentage "25" -> 0.25
	discRat := new(big.Rat)
	if _, ok := discRat.SetString(discountPercent.StringVal); ok {
		// If discount is > 1 (e.g., "25"), treat as percent and divide by 100
		one := new(big.Rat).SetInt64(1)
		if discRat.Cmp(one) == 1 { // discRat > 1
			discRat = new(big.Rat).Quo(discRat, new(big.Rat).SetInt64(100))
		}
	} else {
		// fallback: try parse float
		var f float64
		_, err := fmt.Sscanf(discountPercent.StringVal, "%f", &f)
		if err != nil {
			return nil, fmt.Errorf("invalid discount percent format: %s", discountPercent.StringVal)
		}
		discRat = new(big.Rat).SetFloat64(f)
		if discRat.Cmp(new(big.Rat).SetInt64(1)) == 1 {
			discRat = new(big.Rat).Quo(discRat, new(big.Rat).SetInt64(100))
		}
	}

	discountAmount := new(big.Rat).Mul(base, discRat)
	final := new(big.Rat).Sub(base, discountAmount)
	return final, nil
}
