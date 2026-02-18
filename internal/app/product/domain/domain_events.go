package domain

import "time"

// DomainEvent is a marker interface for all domain events.
// Domain events represent facts about things that have happened in the domain.
type DomainEvent interface {
	EventType() string
	AggregateID() string
	OccurredAt() time.Time
}

// ProductCreatedEvent is raised when a new product is created.
type ProductCreatedEvent struct {
	ProductID string
	Name      string
	Category  string
	BasePrice *Money
	CreatedAt time.Time
}

func (e *ProductCreatedEvent) EventType() string {
	return "product.created"
}

func (e *ProductCreatedEvent) AggregateID() string {
	return e.ProductID
}

func (e *ProductCreatedEvent) OccurredAt() time.Time {
	return e.CreatedAt
}

// ProductUpdatedEvent is raised when product details are updated.
type ProductUpdatedEvent struct {
	ProductID string
	UpdatedAt time.Time
	Changes   map[string]interface{} // Map of field name to new value
}

func (e *ProductUpdatedEvent) EventType() string {
	return "product.updated"
}

func (e *ProductUpdatedEvent) AggregateID() string {
	return e.ProductID
}

func (e *ProductUpdatedEvent) OccurredAt() time.Time {
	return e.UpdatedAt
}

// ProductActivatedEvent is raised when a product is activated.
type ProductActivatedEvent struct {
	ProductID   string
	ActivatedAt time.Time
}

func (e *ProductActivatedEvent) EventType() string {
	return "product.activated"
}

func (e *ProductActivatedEvent) AggregateID() string {
	return e.ProductID
}

func (e *ProductActivatedEvent) OccurredAt() time.Time {
	return e.ActivatedAt
}

// ProductDeactivatedEvent is raised when a product is deactivated.
type ProductDeactivatedEvent struct {
	ProductID     string
	DeactivatedAt time.Time
}

func (e *ProductDeactivatedEvent) EventType() string {
	return "product.deactivated"
}

func (e *ProductDeactivatedEvent) AggregateID() string {
	return e.ProductID
}

func (e *ProductDeactivatedEvent) OccurredAt() time.Time {
	return e.DeactivatedAt
}

// ProductArchivedEvent is raised when a product is archived (soft deleted).
type ProductArchivedEvent struct {
	ProductID  string
	ArchivedAt time.Time
}

func (e *ProductArchivedEvent) EventType() string {
	return "product.archived"
}

func (e *ProductArchivedEvent) AggregateID() string {
	return e.ProductID
}

func (e *ProductArchivedEvent) OccurredAt() time.Time {
	return e.ArchivedAt
}

// DiscountAppliedEvent is raised when a discount is applied to a product.
type DiscountAppliedEvent struct {
	ProductID         string
	DiscountPercent   float64
	DiscountStartDate time.Time
	DiscountEndDate   time.Time
	AppliedAt         time.Time
}

func (e *DiscountAppliedEvent) EventType() string {
	return "product.discount_applied"
}

func (e *DiscountAppliedEvent) AggregateID() string {
	return e.ProductID
}

func (e *DiscountAppliedEvent) OccurredAt() time.Time {
	return e.AppliedAt
}

// DiscountRemovedEvent is raised when a discount is removed from a product.
type DiscountRemovedEvent struct {
	ProductID string
	RemovedAt time.Time
}

func (e *DiscountRemovedEvent) EventType() string {
	return "product.discount_removed"
}

func (e *DiscountRemovedEvent) AggregateID() string {
	return e.ProductID
}

func (e *DiscountRemovedEvent) OccurredAt() time.Time {
	return e.RemovedAt
}

// PriceChangedEvent is raised when the base price of a product changes.
type PriceChangedEvent struct {
	ProductID string
	OldPrice  *Money
	NewPrice  *Money
	ChangedAt time.Time
}

func (e *PriceChangedEvent) EventType() string {
	return "price.changed"
}

func (e *PriceChangedEvent) AggregateID() string {
	return e.ProductID
}

func (e *PriceChangedEvent) OccurredAt() time.Time {
	return e.ChangedAt
}
