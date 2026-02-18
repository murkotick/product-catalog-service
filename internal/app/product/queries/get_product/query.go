package get_product

import (
	"context"
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
		discountPercent            spanner.NullNumeric
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
		dp := new(big.Rat).Set(&discountPercent.Numeric).FloatString(10)
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
func computeEffectivePrice(baseNum, baseDen int64, discountPercent spanner.NullNumeric, start, end spanner.NullTime, now time.Time) (*big.Rat, error) {
	base := new(big.Rat).SetFrac(big.NewInt(baseNum), big.NewInt(baseDen))

	// no discount present
	if !discountPercent.Valid {
		return base, nil
	}

	// check validity window (start inclusive, end inclusive)
	if start.Valid && now.Before(start.Time) {
		return base, nil
	}
	if end.Valid && now.After(end.Time) { // now > end => expired
		return base, nil
	}

	// discount_percent is stored as a NUMERIC (0.0-1.0 scale) and decoded into big.Rat.
	discRat := new(big.Rat).Set(&discountPercent.Numeric)
	// Defensive: if stored as "20" rather than "0.20", normalize to 0-1 scale.
	if discRat.Cmp(big.NewRat(1, 1)) == 1 {
		discRat.Quo(discRat, big.NewRat(100, 1))
	}

	discountAmount := new(big.Rat).Mul(base, discRat)
	final := new(big.Rat).Sub(base, discountAmount)
	return final, nil
}
