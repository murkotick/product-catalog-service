package get_product

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

func (h *Handler) Execute(ctx context.Context, productID string) (*dto.ProductDTO, error) {
	return h.readModel.GetProduct(ctx, productID)
}
