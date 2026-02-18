package product

import (
	"fmt"

	productv1 "github.com/murkotick/product-catalog-service/proto/product/v1"
)

func validateCreateProduct(req *productv1.CreateProductRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}
	if req.GetName() == "" {
		return fmt.Errorf("name is required")
	}
	if req.GetCategory() == "" {
		return fmt.Errorf("category is required")
	}
	if req.BasePrice == nil {
		return fmt.Errorf("base_price is required")
	}
	if req.BasePrice.Denominator == 0 {
		return fmt.Errorf("base_price.denominator must be non-zero")
	}
	return nil
}

func validateUpdateProduct(req *productv1.UpdateProductRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}
	if req.GetProductId() == "" {
		return fmt.Errorf("product_id is required")
	}
	// At least one field should be present
	if req.Name == nil && req.Description == nil && req.Category == nil {
		return fmt.Errorf("at least one field must be provided")
	}
	return nil
}

func validateApplyDiscount(req *productv1.ApplyDiscountRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}
	if req.GetProductId() == "" {
		return fmt.Errorf("product_id is required")
	}
	if req.Discount == nil {
		return fmt.Errorf("discount is required")
	}
	if req.Discount.GetPercentage() == "" {
		return fmt.Errorf("discount.percentage is required")
	}
	if req.Discount.StartDate == nil {
		return fmt.Errorf("discount.start_date is required")
	}
	if req.Discount.EndDate == nil {
		return fmt.Errorf("discount.end_date is required")
	}
	return nil
}
