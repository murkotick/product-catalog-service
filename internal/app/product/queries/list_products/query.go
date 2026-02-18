package list_products

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/murkotick/product-catalog-service/internal/app/product/dto"
)

// SpannerListProductsQuery lists active products with optional category filter.
type SpannerListProductsQuery struct {
	Client *spanner.Client
}

func NewSpannerListProductsQuery(client *spanner.Client) *SpannerListProductsQuery {
	return &SpannerListProductsQuery{Client: client}
}

func (q *SpannerListProductsQuery) ListActiveProducts(ctx context.Context, category *string, limit, offset int) ([]*dto.ProductSummaryDTO, error) {
	baseSQL := `SELECT product_id, name, category,
					  base_price_numerator, base_price_denominator,
					  discount_percent, discount_start_date, discount_end_date
		FROM products
		WHERE status = 'active'`
	params := map[string]interface{}{}
	if category != nil {
		baseSQL += " AND category = @category"
		params["category"] = *category
	}
	baseSQL += " ORDER BY name ASC LIMIT @limit OFFSET @offset"
	params["limit"] = limit
	params["offset"] = offset

	stmt := spanner.Statement{SQL: baseSQL, Params: params}
	iter := q.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var out []*dto.ProductSummaryDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}

		var (
			id                         string
			name                       string
			categoryStr                string
			baseNum                    int64
			baseDen                    int64
			discountPct                spanner.NullString
			discountStart, discountEnd spanner.NullTime
		)
		if err := row.Columns(&id, &name, &categoryStr, &baseNum, &baseDen, &discountPct, &discountStart, &discountEnd); err != nil {
			return nil, err
		}

		priceRat, err := computeEffectivePrice(baseNum, baseDen, discountPct, discountStart, discountEnd, time.Now().UTC())
		if err != nil {
			return nil, err
		}

		out = append(out, &dto.ProductSummaryDTO{
			ProductID:      id,
			Name:           name,
			Category:       categoryStr,
			EffectivePrice: priceRat.FloatString(10),
			BasePriceNum:   baseNum,
			BasePriceDen:   baseDen,
			Status:         "active",
		})
	}
}

// computeEffectivePrice mirrors the helper from get_product.
func computeEffectivePrice(baseNum, baseDen int64, discountPercent spanner.NullString, start, end spanner.NullTime, now time.Time) (*big.Rat, error) {
	base := new(big.Rat).SetFrac(big.NewInt(baseNum), big.NewInt(baseDen))

	if !discountPercent.Valid || discountPercent.StringVal == "" {
		return base, nil
	}
	if start.Valid && now.Before(start.Time) {
		return base, nil
	}
	if end.Valid && now.After(end.Time) {
		return base, nil
	}

	discRat := new(big.Rat)
	if _, ok := discRat.SetString(discountPercent.StringVal); !ok {
		return nil, fmt.Errorf("invalid discount_percent: %q", discountPercent.StringVal)
	}
	if discRat.Cmp(new(big.Rat).SetInt64(1)) == 1 {
		discRat = new(big.Rat).Quo(discRat, new(big.Rat).SetInt64(100))
	}

	discountAmount := new(big.Rat).Mul(base, discRat)
	final := new(big.Rat).Sub(base, discountAmount)
	return final, nil
}
