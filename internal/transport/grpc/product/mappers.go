package product

import (
	"fmt"
	"math/big"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	productv1 "github.com/murkotick/product-catalog-service/proto/product/v1"

	"github.com/murkotick/product-catalog-service/internal/app/product/dto"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/apply_discount"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/create_product"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/update_product"
)

func mapCreateProductRequest(req *productv1.CreateProductRequest) (create_product.Request, error) {
	money := req.GetBasePrice()
	if money == nil {
		return create_product.Request{}, fmt.Errorf("base_price is required")
	}
	if money.Denominator == 0 {
		return create_product.Request{}, fmt.Errorf("base_price.denominator must be non-zero")
	}

	return create_product.Request{
		Name:         req.GetName(),
		Description:  req.GetDescription(),
		Category:     req.GetCategory(),
		BasePriceNum: money.Numerator,
		BasePriceDen: money.Denominator,
	}, nil
}

func mapUpdateProductRequest(req *productv1.UpdateProductRequest) update_product.Request {
	out := update_product.Request{ProductID: req.GetProductId()}
	if req.Name != nil {
		v := req.GetName()
		out.Name = &v
	}
	if req.Description != nil {
		v := req.GetDescription()
		out.Description = &v
	}
	if req.Category != nil {
		v := req.GetCategory()
		out.Category = &v
	}
	return out
}

func mapApplyDiscountRequest(req *productv1.ApplyDiscountRequest) (apply_discount.Request, error) {
	if req.GetDiscount() == nil {
		return apply_discount.Request{}, fmt.Errorf("discount is required")
	}
	d := req.GetDiscount()

	start := time.Time{}
	end := time.Time{}
	if d.StartDate != nil {
		start = d.StartDate.AsTime()
	}
	if d.EndDate != nil {
		end = d.EndDate.AsTime()
	}
	if start.IsZero() {
		return apply_discount.Request{}, fmt.Errorf("discount.start_date is required")
	}
	if end.IsZero() {
		return apply_discount.Request{}, fmt.Errorf("discount.end_date is required")
	}

	pct, err := parseDiscountPercentageToFloat(d.GetPercentage())
	if err != nil {
		return apply_discount.Request{}, err
	}

	return apply_discount.Request{
		ProductID:  req.GetProductId(),
		Percentage: pct,
		StartDate:  start.UTC(),
		EndDate:    end.UTC(),
	}, nil
}

// parseDiscountPercentageToFloat accepts either "20" (20%) or "0.2" (20%).
// The application-layer interactor expects 0-100 scale.
func parseDiscountPercentageToFloat(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("discount.percentage is required")
	}

	r := new(big.Rat)
	if _, ok := r.SetString(s); !ok {
		return 0, fmt.Errorf("invalid discount.percentage: %q", s)
	}

	// If <= 1, treat as fraction (0.2 => 20%). Otherwise treat as percent.
	if r.Cmp(big.NewRat(1, 1)) <= 0 {
		r = new(big.Rat).Mul(r, big.NewRat(100, 1))
	}

	f, _ := r.Float64()
	return f, nil
}

func mapProductDTOToProto(in *dto.ProductDTO) (*productv1.Product, error) {
	if in == nil {
		return nil, fmt.Errorf("nil product")
	}

	out := &productv1.Product{
		Id:        in.ProductID,
		Name:      in.Name,
		Category:  in.Category,
		Status:    mapStatusToProto(in.Status),
		BasePrice: &productv1.Money{Numerator: in.BasePriceNum, Denominator: in.BasePriceDen},
	}

	if in.Description != nil {
		out.Description = *in.Description
	}

	// Timestamps (DTO uses RFC3339 strings)
	if ts, err := parseRFC3339Ptr(in.CreatedAt); err != nil {
		return nil, err
	} else if ts != nil {
		out.CreatedAt = timestamppb.New(*ts)
	}
	if ts, err := parseRFC3339Ptr(in.UpdatedAt); err != nil {
		return nil, err
	} else if ts != nil {
		out.UpdatedAt = timestamppb.New(*ts)
	}
	if ts, err := parseRFC3339Ptr(in.ArchivedAt); err != nil {
		return nil, err
	} else if ts != nil {
		out.ArchivedAt = timestamppb.New(*ts)
	}

	// Effective price
	if in.EffectivePrice != "" {
		rat := new(big.Rat)
		if _, ok := rat.SetString(in.EffectivePrice); ok {
			m, err := ratToProtoMoney(rat)
			if err != nil {
				return nil, err
			}
			out.EffectivePrice = m
		}
	}

	// Discount (only if present; whether it is currently active is handled by read side)
	if in.DiscountPct != nil && in.DiscountStart != nil && in.DiscountEnd != nil {
		ds, err := parseRFC3339Ptr(in.DiscountStart)
		if err != nil {
			return nil, err
		}
		de, err := parseRFC3339Ptr(in.DiscountEnd)
		if err != nil {
			return nil, err
		}
		if ds != nil && de != nil {
			out.ActiveDiscount = &productv1.Discount{
				Percentage: *in.DiscountPct,
				StartDate:  timestamppb.New(*ds),
				EndDate:    timestamppb.New(*de),
			}
		}
	}

	return out, nil
}

func mapProductSummariesToProto(items []*dto.ProductSummaryDTO) ([]*productv1.Product, error) {
	out := make([]*productv1.Product, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}

		p := &productv1.Product{
			Id:       it.ProductID,
			Name:     it.Name,
			Category: it.Category,
			Status:   mapStatusToProto(it.Status),
		}

		if it.EffectivePrice != "" {
			rat := new(big.Rat)
			if _, ok := rat.SetString(it.EffectivePrice); ok {
				m, err := ratToProtoMoney(rat)
				if err != nil {
					return nil, err
				}
				p.EffectivePrice = m
			}
		}

		// Best-effort base price if available (added in phase 5 for better API responses).
		if it.BasePriceDen != 0 {
			p.BasePrice = &productv1.Money{Numerator: it.BasePriceNum, Denominator: it.BasePriceDen}
		}

		out = append(out, p)
	}
	return out, nil
}

func ratToProtoMoney(r *big.Rat) (*productv1.Money, error) {
	if r == nil {
		return &productv1.Money{Numerator: 0, Denominator: 1}, nil
	}
	n := r.Num()
	d := r.Denom()
	if !n.IsInt64() || !d.IsInt64() {
		return nil, fmt.Errorf("money value out of int64 range")
	}
	return &productv1.Money{Numerator: n.Int64(), Denominator: d.Int64()}, nil
}

func parseRFC3339Ptr(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, err
	}
	tt := t.UTC()
	return &tt, nil
}

func mapStatusToProto(status string) productv1.ProductStatus {
	switch status {
	case "active":
		return productv1.ProductStatus_PRODUCT_STATUS_ACTIVE
	case "draft":
		return productv1.ProductStatus_PRODUCT_STATUS_INACTIVE
	case "inactive":
		return productv1.ProductStatus_PRODUCT_STATUS_INACTIVE
	case "archived":
		return productv1.ProductStatus_PRODUCT_STATUS_ARCHIVED
	default:
		return productv1.ProductStatus_PRODUCT_STATUS_UNSPECIFIED
	}
}
