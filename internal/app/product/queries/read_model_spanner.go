package queries

import (
	"context"

	"cloud.google.com/go/spanner"

	"github.com/murkotick/product-catalog-service/internal/app/product/dto"
	"github.com/murkotick/product-catalog-service/internal/app/product/queries/get_product"
	"github.com/murkotick/product-catalog-service/internal/app/product/queries/list_products"
)

// SpannerReadModel is an infrastructure adapter that satisfies contracts.ReadModel.
// It composes the individual query implementations.
type SpannerReadModel struct {
	getQ  *get_product.SpannerGetProductQuery
	listQ *list_products.SpannerListProductsQuery
}

func NewSpannerReadModel(client *spanner.Client) *SpannerReadModel {
	return &SpannerReadModel{
		getQ:  get_product.NewSpannerGetProductQuery(client),
		listQ: list_products.NewSpannerListProductsQuery(client),
	}
}

func (rm *SpannerReadModel) GetProduct(ctx context.Context, productID string) (*dto.ProductDTO, error) {
	return rm.getQ.GetProduct(ctx, productID)
}

func (rm *SpannerReadModel) ListActiveProducts(ctx context.Context, category *string, limit, offset int) ([]*dto.ProductSummaryDTO, error) {
	return rm.listQ.ListActiveProducts(ctx, category, limit, offset)
}
