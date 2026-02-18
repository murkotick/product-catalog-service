package product

import (
	"context"
	"errors"

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/murkotick/product-catalog-service/internal/app/product/domain"
)

// mapError translates domain sentinel errors into proper gRPC status codes.
// Unknown errors become codes.Internal.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, err.Error())
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, err.Error())
	}

	// Not found
	if errors.Is(err, domain.ErrProductNotFound) || errors.Is(err, spanner.ErrRowNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}

	// Invalid argument (validation)
	switch {
	case errors.Is(err, domain.ErrEmptyProductName),
		errors.Is(err, domain.ErrEmptyProductCategory),
		errors.Is(err, domain.ErrProductNameTooLong),
		errors.Is(err, domain.ErrProductCategoryTooLong),
		errors.Is(err, domain.ErrProductDescriptionTooLong),
		errors.Is(err, domain.ErrInvalidDiscountPercentage),
		errors.Is(err, domain.ErrInvalidDiscountPeriod),
		errors.Is(err, domain.ErrNegativePrice),
		errors.Is(err, domain.ErrZeroPrice):
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Failed precondition (business rules / state)
	switch {
	case errors.Is(err, domain.ErrProductNotActive),
		errors.Is(err, domain.ErrProductArchived),
		errors.Is(err, domain.ErrProductAlreadyActive),
		errors.Is(err, domain.ErrProductAlreadyInactive),
		errors.Is(err, domain.ErrCannotArchiveActiveProduct),
		errors.Is(err, domain.ErrDiscountNotValid),
		errors.Is(err, domain.ErrDiscountAlreadyExists):
		return status.Error(codes.FailedPrecondition, err.Error())
	}

	return status.Error(codes.Internal, err.Error())
}
