package product

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	productv1 "github.com/murkotick/product-catalog-service/proto/product/v1"

	"github.com/murkotick/product-catalog-service/internal/app/product/queries/get_product"
	"github.com/murkotick/product-catalog-service/internal/app/product/queries/list_products"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/activate_product"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/apply_discount"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/create_product"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/update_product"
)

// Commands groups write interactors.
// Keep transport layer depending on application layer only.
type Commands struct {
	Create   *create_product.Interactor
	Update   *update_product.Interactor
	Activate *activate_product.Interactor
	ApplyDis *apply_discount.Interactor
}

// Queries groups read handlers.
type Queries struct {
	Get  *get_product.Handler
	List *list_products.Handler
}

// Handler is a thin gRPC transport adapter.
// It validates input, maps proto <-> application DTOs and delegates to CQRS handlers.
type Handler struct {
	productv1.UnimplementedProductServiceServer

	commands Commands
	queries  Queries
}

func NewHandler(cmd Commands, qry Queries) *Handler {
	return &Handler{commands: cmd, queries: qry}
}

func (h *Handler) CreateProduct(ctx context.Context, req *productv1.CreateProductRequest) (*productv1.CreateProductReply, error) {
	if err := validateCreateProduct(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	appReq, err := mapCreateProductRequest(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	id, err := h.commands.Create.Execute(ctx, appReq)
	if err != nil {
		return nil, mapError(err)
	}

	return &productv1.CreateProductReply{ProductId: id}, nil
}

func (h *Handler) UpdateProduct(ctx context.Context, req *productv1.UpdateProductRequest) (*productv1.UpdateProductReply, error) {
	if err := validateUpdateProduct(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	appReq := mapUpdateProductRequest(req)
	if err := h.commands.Update.Execute(ctx, appReq); err != nil {
		return nil, mapError(err)
	}
	return &productv1.UpdateProductReply{}, nil
}

func (h *Handler) ActivateProduct(ctx context.Context, req *productv1.ActivateProductRequest) (*productv1.ActivateProductReply, error) {
	if req == nil || req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	if err := h.commands.Activate.Execute(ctx, activate_product.Request{ProductID: req.ProductId}); err != nil {
		return nil, mapError(err)
	}
	return &productv1.ActivateProductReply{}, nil
}

func (h *Handler) DeactivateProduct(ctx context.Context, req *productv1.DeactivateProductRequest) (*productv1.DeactivateProductReply, error) {
	// NOTE: application-layer interactor not implemented yet in Phase 4.
	return nil, status.Error(codes.Unimplemented, "DeactivateProduct not implemented")
}

func (h *Handler) ApplyDiscount(ctx context.Context, req *productv1.ApplyDiscountRequest) (*productv1.ApplyDiscountReply, error) {
	if err := validateApplyDiscount(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	appReq, err := mapApplyDiscountRequest(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := h.commands.ApplyDis.Execute(ctx, appReq); err != nil {
		return nil, mapError(err)
	}
	return &productv1.ApplyDiscountReply{}, nil
}

func (h *Handler) RemoveDiscount(ctx context.Context, req *productv1.RemoveDiscountRequest) (*productv1.RemoveDiscountReply, error) {
	// NOTE: application-layer interactor not implemented yet in Phase 4.
	return nil, status.Error(codes.Unimplemented, "RemoveDiscount not implemented")
}

func (h *Handler) GetProduct(ctx context.Context, req *productv1.GetProductRequest) (*productv1.GetProductReply, error) {
	if req == nil || req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	dtoOut, err := h.queries.Get.Execute(ctx, req.ProductId)
	if err != nil {
		return nil, mapError(err)
	}

	pbProd, err := mapProductDTOToProto(dtoOut)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &productv1.GetProductReply{Product: pbProd}, nil
}

func (h *Handler) ListProducts(ctx context.Context, req *productv1.ListProductsRequest) (*productv1.ListProductsReply, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	offset, err := decodePageToken(req.PageToken)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid page_token")
	}

	var category *string
	if req.Category != nil {
		c := req.GetCategory()
		if c != "" {
			category = &c
		}
	}

	items, err := h.queries.List.Execute(ctx, category, limit, offset)
	if err != nil {
		return nil, mapError(err)
	}

	products, err := mapProductSummariesToProto(items)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	next := ""
	if len(items) == limit {
		next = encodePageToken(offset + len(items))
	}

	return &productv1.ListProductsReply{Products: products, NextPageToken: next}, nil
}
