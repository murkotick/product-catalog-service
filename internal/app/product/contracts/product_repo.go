package contracts

import (
	"cloud.google.com/go/spanner"
	domain "github.com/murkotick/product-catalog-service/internal/app/product/domain"
)

// ProductRepo is the write-side repository interface for products.
// Methods return Spanner mutations; they do not apply them.
type ProductRepo interface {
	// InsertMut returns a mutation that inserts the product (or nil if none).
	InsertMut(p *domain.Product) *spanner.Mutation

	// UpdateMut returns a mutation that updates the product according to its ChangeTracker (or nil).
	UpdateMut(p *domain.Product) *spanner.Mutation

	// ArchiveMut returns a mutation to soft-delete (archive) the product (or nil).
	ArchiveMut(p *domain.Product) *spanner.Mutation
}
