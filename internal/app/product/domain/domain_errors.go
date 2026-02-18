package domain

import "errors"

// Domain errors for Product aggregate
var (
	// ErrProductNotFound indicates that a product with the given ID does not exist.
	ErrProductNotFound = errors.New("product not found")

	// ErrProductNotActive indicates an operation that requires an active product
	// was attempted on an inactive product.
	ErrProductNotActive = errors.New("product is not active")

	// ErrProductAlreadyActive indicates an attempt to activate an already active product.
	ErrProductAlreadyActive = errors.New("product is already active")

	// ErrProductAlreadyInactive indicates an attempt to deactivate an already inactive product.
	ErrProductAlreadyInactive = errors.New("product is already inactive")

	// ErrProductArchived indicates an operation on an archived product that is not allowed.
	ErrProductArchived = errors.New("product is archived")

	// ErrCannotArchiveActiveProduct indicates an attempt to archive an active product.
	ErrCannotArchiveActiveProduct = errors.New("cannot archive an active product")
)

// Domain errors for Discount value object
var (
	// ErrInvalidDiscountPercentage indicates the discount percentage is outside valid range (0-100).
	ErrInvalidDiscountPercentage = errors.New("discount percentage must be between 0 and 100")

	// ErrInvalidDiscountPeriod indicates the discount date range is invalid.
	ErrInvalidDiscountPeriod = errors.New("discount end date must be after start date")

	// ErrDiscountNotValid indicates the discount is not valid at the current time.
	ErrDiscountNotValid = errors.New("discount is not valid at this time")

	// ErrDiscountAlreadyExists indicates an attempt to apply a discount when one already exists.
	ErrDiscountAlreadyExists = errors.New("product already has an active discount")
)

// Domain errors for Money value object
var (
	// ErrNegativePrice indicates an attempt to set a negative price.
	ErrNegativePrice = errors.New("price cannot be negative")

	// ErrZeroPrice indicates an attempt to set a zero price.
	ErrZeroPrice = errors.New("price cannot be zero")
)

// Domain errors for Product validation
var (
	// ErrEmptyProductName indicates an attempt to create/update a product with an empty name.
	ErrEmptyProductName = errors.New("product name cannot be empty")

	// ErrEmptyProductCategory indicates an attempt to create/update a product with an empty category.
	ErrEmptyProductCategory = errors.New("product category cannot be empty")

	// ErrProductNameTooLong indicates the product name exceeds maximum length.
	ErrProductNameTooLong = errors.New("product name exceeds maximum length of 255 characters")

	// ErrProductDescriptionTooLong indicates the product description exceeds maximum length.
	ErrProductDescriptionTooLong = errors.New("product description exceeds maximum length")

	// ErrProductCategoryTooLong indicates the product category exceeds maximum length.
	ErrProductCategoryTooLong = errors.New("product category exceeds maximum length of 100 characters")
)
