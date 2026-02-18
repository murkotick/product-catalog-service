package repo

import (
	"time"

	"cloud.google.com/go/spanner"
	domain "github.com/murkotick/product-catalog-service/internal/app/product/domain"
	"github.com/murkotick/product-catalog-service/internal/models/m_product"
)

// ProductRepo is the Spanner implementation of the write-side repository.
// It returns *spanner.Mutation objects but never applies them.
type ProductRepo struct{}

func NewProductRepo() *ProductRepo {
	return &ProductRepo{}
}

// buildInsertValues constructs the values map used for insertion.
// It's unexported so tests in the same package can inspect the map without
// relying on spanner.Mutation internals.
func buildInsertValues(p *domain.Product) map[string]interface{} {
	productID := p.ID()
	name := p.Name()
	var description *string
	if d := p.Description(); d != "" {
		desc := d
		description = &desc
	}
	category := p.Category()

	// base price assumed returned as numerator/denominator
	base := p.BasePrice() // *domain.Money
	baseNum := base.Numerator()
	baseDen := base.Denominator()

	var discountPct *string
	var discountStart *time.Time
	var discountEnd *time.Time
	if d := p.Discount(); d != nil {
		// Store the discount as a precise decimal fraction in [0,1].
		// Example: 20% => "0.2".
		discStr := d.PercentageRat().FloatString(10)
		discountPct = &discStr

		if !d.StartDate().IsZero() {
			s := d.StartDate().UTC()
			discountStart = &s
		}
		if !d.EndDate().IsZero() {
			e := d.EndDate().UTC()
			discountEnd = &e
		}
	}

	status := string(p.Status())

	values := m_product.BuildInsertMap(productID, name, description, category, baseNum, baseDen,
		discountPct, discountStart, discountEnd, status, p.CreatedAt().UTC(), p.UpdatedAt().UTC())

	return values
}

// InsertMut builds an Insert mutation for a new product.
func (r *ProductRepo) InsertMut(p *domain.Product) *spanner.Mutation {
	values := buildInsertValues(p)
	return m_product.InsertMutation(values)
}

// UpdateMut builds an Update mutation using the aggregate's ChangeTracker.
// It updates only dirty fields and always stamps updated_at when there are changes.
func (r *ProductRepo) UpdateMut(p *domain.Product) *spanner.Mutation {
	if p == nil || p.Changes() == nil || !p.Changes().HasChanges() {
		return nil
	}

	updates := map[string]interface{}{}

	if p.Changes().Dirty(domain.FieldName) {
		updates[m_product.ColName] = p.Name()
	}
	if p.Changes().Dirty(domain.FieldDescription) {
		if p.Description() == "" {
			updates[m_product.ColDescription] = nil
		} else {
			updates[m_product.ColDescription] = p.Description()
		}
	}
	if p.Changes().Dirty(domain.FieldCategory) {
		updates[m_product.ColCategory] = p.Category()
	}
	if p.Changes().Dirty(domain.FieldBasePrice) {
		updates[m_product.ColBasePriceNumerator] = p.BasePrice().Numerator()
		updates[m_product.ColBasePriceDenominator] = p.BasePrice().Denominator()
	}
	if p.Changes().Dirty(domain.FieldDiscount) {
		if d := p.Discount(); d != nil {
			updates[m_product.ColDiscountPercent] = d.PercentageRat().FloatString(10)
			s := d.StartDate().UTC()
			e := d.EndDate().UTC()
			updates[m_product.ColDiscountStartDate] = s
			updates[m_product.ColDiscountEndDate] = e
		} else {
			updates[m_product.ColDiscountPercent] = nil
			updates[m_product.ColDiscountStartDate] = nil
			updates[m_product.ColDiscountEndDate] = nil
		}
	}
	if p.Changes().Dirty(domain.FieldStatus) {
		updates[m_product.ColStatus] = string(p.Status())
	}
	if p.Changes().Dirty(domain.FieldArchivedAt) {
		if p.ArchivedAt() != nil {
			updates[m_product.ColArchivedAt] = p.ArchivedAt().UTC()
		} else {
			updates[m_product.ColArchivedAt] = nil
		}
	}

	if len(updates) == 0 {
		return nil
	}

	updates[m_product.ColUpdatedAt] = p.UpdatedAt().UTC()
	return m_product.UpdateMutation(p.ID(), updates)
}

// ArchiveMut returns a mutation to soft-delete the product (archive).
// The aggregate must already have been transitioned via p.Archive(now).
func (r *ProductRepo) ArchiveMut(p *domain.Product) *spanner.Mutation {
	return r.UpdateMut(p)
}
