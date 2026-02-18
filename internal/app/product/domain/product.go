package domain

import (
	"strings"
	"time"
)

// Field constants for change tracking
const (
	FieldName        = "name"
	FieldDescription = "description"
	FieldCategory    = "category"
	FieldBasePrice   = "base_price"
	FieldDiscount    = "discount"
	FieldStatus      = "status"
	FieldArchivedAt  = "archived_at"
)

// ProductStatus represents the lifecycle state of a product.
type ProductStatus string

const (
	// ProductStatusDraft indicates a product that is being prepared.
	ProductStatusDraft ProductStatus = "draft"

	// ProductStatusActive indicates a product that is available for sale.
	ProductStatusActive ProductStatus = "active"

	// ProductStatusInactive indicates a product that is temporarily unavailable.
	ProductStatusInactive ProductStatus = "inactive"

	// ProductStatusArchived indicates a product that has been soft-deleted.
	ProductStatusArchived ProductStatus = "archived"
)

// Product is the aggregate root for the product catalog domain.
// It encapsulates all business rules related to products and pricing.
type Product struct {
	id          string
	name        string
	description string
	category    string
	basePrice   *Money
	discount    *Discount
	status      ProductStatus
	createdAt   time.Time
	updatedAt   time.Time
	archivedAt  *time.Time
	changes     *ChangeTracker
	events      []DomainEvent
}

// NewProduct creates a new Product with the given details.
// The product starts in Draft status.
func NewProduct(id, name, description, category string, basePrice *Money, now time.Time) (*Product, error) {
	// Validate inputs
	if err := validateProductName(name); err != nil {
		return nil, err
	}
	if err := validateProductCategory(category); err != nil {
		return nil, err
	}
	if err := validatePrice(basePrice); err != nil {
		return nil, err
	}

	p := &Product{
		id:          id,
		name:        strings.TrimSpace(name),
		description: strings.TrimSpace(description),
		category:    strings.TrimSpace(category),
		basePrice:   basePrice,
		status:      ProductStatusDraft,
		createdAt:   now,
		updatedAt:   now,
		changes:     NewChangeTracker(),
		events:      make([]DomainEvent, 0),
	}

	// Capture creation event
	p.events = append(p.events, &ProductCreatedEvent{
		ProductID: p.id,
		Name:      p.name,
		Category:  p.category,
		BasePrice: p.basePrice,
		CreatedAt: now,
	})

	return p, nil
}

// ReconstructProduct reconstructs a Product from persisted state.
// Used by repositories when loading from the database.
func ReconstructProduct(
	id, name, description, category string,
	basePrice *Money,
	discount *Discount,
	status ProductStatus,
	createdAt, updatedAt time.Time,
	archivedAt *time.Time,
) *Product {
	return &Product{
		id:          id,
		name:        name,
		description: description,
		category:    category,
		basePrice:   basePrice,
		discount:    discount,
		status:      status,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		archivedAt:  archivedAt,
		changes:     NewChangeTracker(),
		events:      make([]DomainEvent, 0),
	}
}

// Getters

func (p *Product) ID() string {
	return p.id
}

func (p *Product) Name() string {
	return p.name
}

func (p *Product) Description() string {
	return p.description
}

func (p *Product) Category() string {
	return p.category
}

func (p *Product) BasePrice() *Money {
	return p.basePrice
}

func (p *Product) Discount() *Discount {
	return p.discount
}

func (p *Product) Status() ProductStatus {
	return p.status
}

func (p *Product) CreatedAt() time.Time {
	return p.createdAt
}

func (p *Product) UpdatedAt() time.Time {
	return p.updatedAt
}

func (p *Product) ArchivedAt() *time.Time {
	return p.archivedAt
}

func (p *Product) Changes() *ChangeTracker {
	return p.changes
}

func (p *Product) DomainEvents() []DomainEvent {
	return p.events
}

// Business Methods

// UpdateDetails updates the product's name, description, and/or category.
// Only updates fields that are provided (non-empty).
func (p *Product) UpdateDetails(name, description, category string, now time.Time) error {
	if p.status == ProductStatusArchived {
		return ErrProductArchived
	}

	changes := make(map[string]interface{})

	// Update name if provided
	if name != "" {
		if err := validateProductName(name); err != nil {
			return err
		}
		trimmedName := strings.TrimSpace(name)
		if trimmedName != p.name {
			p.name = trimmedName
			p.changes.MarkDirty(FieldName)
			changes["name"] = p.name
		}
	}

	// Update description if provided
	if description != "" {
		trimmedDesc := strings.TrimSpace(description)
		if trimmedDesc != p.description {
			p.description = trimmedDesc
			p.changes.MarkDirty(FieldDescription)
			changes["description"] = p.description
		}
	}

	// Update category if provided
	if category != "" {
		if err := validateProductCategory(category); err != nil {
			return err
		}
		trimmedCat := strings.TrimSpace(category)
		if trimmedCat != p.category {
			p.category = trimmedCat
			p.changes.MarkDirty(FieldCategory)
			changes["category"] = p.category
		}
	}

	// Only emit event if something changed
	if len(changes) > 0 {
		p.updatedAt = now
		p.events = append(p.events, &ProductUpdatedEvent{
			ProductID: p.id,
			UpdatedAt: now,
			Changes:   changes,
		})
	}

	return nil
}

// UpdatePrice changes the base price of the product.
func (p *Product) UpdatePrice(newPrice *Money, now time.Time) error {
	if p.status == ProductStatusArchived {
		return ErrProductArchived
	}

	if err := validatePrice(newPrice); err != nil {
		return err
	}

	if !newPrice.Equals(p.basePrice) {
		oldPrice := p.basePrice
		p.basePrice = newPrice
		p.changes.MarkDirty(FieldBasePrice)
		p.updatedAt = now

		p.events = append(p.events, &PriceChangedEvent{
			ProductID: p.id,
			OldPrice:  oldPrice,
			NewPrice:  newPrice,
			ChangedAt: now,
		})
	}

	return nil
}

// Activate transitions the product to Active status, making it available for sale.
func (p *Product) Activate(now time.Time) error {
	if p.status == ProductStatusArchived {
		return ErrProductArchived
	}

	if p.status == ProductStatusActive {
		return ErrProductAlreadyActive
	}

	p.status = ProductStatusActive
	p.changes.MarkDirty(FieldStatus)
	p.updatedAt = now

	p.events = append(p.events, &ProductActivatedEvent{
		ProductID:   p.id,
		ActivatedAt: now,
	})

	return nil
}

// Deactivate transitions the product to Inactive status, temporarily removing it from sale.
func (p *Product) Deactivate(now time.Time) error {
	if p.status == ProductStatusArchived {
		return ErrProductArchived
	}

	if p.status == ProductStatusInactive {
		return ErrProductAlreadyInactive
	}

	p.status = ProductStatusInactive
	p.changes.MarkDirty(FieldStatus)
	p.updatedAt = now

	p.events = append(p.events, &ProductDeactivatedEvent{
		ProductID:     p.id,
		DeactivatedAt: now,
	})

	return nil
}

// Archive soft-deletes the product.
// Active products cannot be archived.
func (p *Product) Archive(now time.Time) error {
	if p.status == ProductStatusActive {
		return ErrCannotArchiveActiveProduct
	}

	if p.status == ProductStatusArchived {
		return ErrProductArchived
	}

	p.status = ProductStatusArchived
	p.archivedAt = &now
	p.changes.MarkDirty(FieldStatus)
	p.changes.MarkDirty(FieldArchivedAt)
	p.updatedAt = now

	p.events = append(p.events, &ProductArchivedEvent{
		ProductID:  p.id,
		ArchivedAt: now,
	})

	return nil
}

// ApplyDiscount applies a discount to the product.
// Only active products can have discounts applied.
// Only one discount can be active at a time.
func (p *Product) ApplyDiscount(discount *Discount, now time.Time) error {
	if p.status != ProductStatusActive {
		return ErrProductNotActive
	}

	if !discount.IsValidAt(now) {
		return ErrDiscountNotValid
	}

	if p.discount != nil {
		return ErrDiscountAlreadyExists
	}

	p.discount = discount
	p.changes.MarkDirty(FieldDiscount)
	p.updatedAt = now

	p.events = append(p.events, &DiscountAppliedEvent{
		ProductID:         p.id,
		DiscountPercent:   discount.Percentage(),
		DiscountStartDate: discount.StartDate(),
		DiscountEndDate:   discount.EndDate(),
		AppliedAt:         now,
	})

	return nil
}

// RemoveDiscount removes any existing discount from the product.
func (p *Product) RemoveDiscount(now time.Time) error {
	if p.status == ProductStatusArchived {
		return ErrProductArchived
	}

	if p.discount == nil {
		return nil // No discount to remove
	}

	p.discount = nil
	p.changes.MarkDirty(FieldDiscount)
	p.updatedAt = now

	p.events = append(p.events, &DiscountRemovedEvent{
		ProductID: p.id,
		RemovedAt: now,
	})

	return nil
}

// CalculateEffectivePrice calculates the current effective price considering any active discount.
func (p *Product) CalculateEffectivePrice(now time.Time) *Money {
	if p.discount != nil && p.discount.IsValidAt(now) {
		return p.discount.ApplyTo(p.basePrice)
	}
	return p.basePrice
}

// IsActive returns true if the product is in Active status.
func (p *Product) IsActive() bool {
	return p.status == ProductStatusActive
}

// IsArchived returns true if the product is archived.
func (p *Product) IsArchived() bool {
	return p.status == ProductStatusArchived
}

// HasActiveDiscount returns true if the product has a discount that is valid at the given time.
func (p *Product) HasActiveDiscount(now time.Time) bool {
	return p.discount != nil && p.discount.IsValidAt(now)
}

// ClearEvents clears the accumulated domain events.
// Should be called after events have been published.
func (p *Product) ClearEvents() {
	p.events = make([]DomainEvent, 0)
}

// Validation helpers

func validateProductName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ErrEmptyProductName
	}
	if len(trimmed) > 255 {
		return ErrProductNameTooLong
	}
	return nil
}

func validateProductCategory(category string) error {
	trimmed := strings.TrimSpace(category)
	if trimmed == "" {
		return ErrEmptyProductCategory
	}
	if len(trimmed) > 100 {
		return ErrProductCategoryTooLong
	}
	return nil
}

func validatePrice(price *Money) error {
	if price == nil {
		return ErrZeroPrice
	}
	if price.IsNegative() {
		return ErrNegativePrice
	}
	if price.IsZero() {
		return ErrZeroPrice
	}
	return nil
}
