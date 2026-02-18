package contracts

import (
	"context"

	"github.com/murkotick/product-catalog-service/internal/app/product/dto"
)

type ReadModel interface {
	GetProduct(ctx context.Context, productID string) (*dto.ProductDTO, error)
	ListActiveProducts(ctx context.Context, category *string, limit, offset int) ([]*dto.ProductSummaryDTO, error)
}
