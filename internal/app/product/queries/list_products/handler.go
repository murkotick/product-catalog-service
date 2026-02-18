package list_products

import (
	"context"

	contracts "github.com/murkotick/product-catalog-service/internal/app/product/contracts"
	"github.com/murkotick/product-catalog-service/internal/app/product/dto"
)

type Handler struct {
	readModel contracts.ReadModel
}

func NewHandler(r contracts.ReadModel) *Handler {
	return &Handler{readModel: r}
}

func (h *Handler) Execute(ctx context.Context, category *string, limit, offset int) ([]*dto.ProductSummaryDTO, error) {
	return h.readModel.ListActiveProducts(ctx, category, limit, offset)
}
