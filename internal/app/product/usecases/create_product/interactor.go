package create_product

import (
	"context"

	"github.com/google/uuid"

	contracts "github.com/murkotick/product-catalog-service/internal/app/product/contracts"
	"github.com/murkotick/product-catalog-service/internal/app/product/domain"
	shared "github.com/murkotick/product-catalog-service/internal/app/product/usecases/shared"
	"github.com/murkotick/product-catalog-service/internal/pkg/clock"
	commitplan "github.com/murkotick/product-catalog-service/internal/pkg/committer"
)

// Request is the application-level create-product request.
type Request struct {
	Name         string
	Description  string
	Category     string
	BasePriceNum int64 // numerator
	BasePriceDen int64 // denominator
}

// Interactor implements the create-product usecase following the Golden Mutation pattern.
type Interactor struct {
	ProductRepo contracts.ProductRepo
	OutboxRepo  contracts.OutboxRepo
	Committer   contracts.Committer
	Clock       clock.Clock
}

// NewInteractor constructs the interactor.
func NewInteractor(prodRepo contracts.ProductRepo, outboxRepo contracts.OutboxRepo, committer contracts.Committer, clk clock.Clock) *Interactor {
	return &Interactor{
		ProductRepo: prodRepo,
		OutboxRepo:  outboxRepo,
		Committer:   committer,
		Clock:       clk,
	}
}

// Execute creates a new product, persists it and writes outbox events in a single commit.
func (it *Interactor) Execute(ctx context.Context, req Request) (string, error) {
	now := it.Clock.Now()

	// 1. Build domain aggregate
	id := uuid.New().String()
	baseMoney := domain.NewMoney(req.BasePriceNum, req.BasePriceDen)
	product, err := domain.NewProduct(id, req.Name, req.Description, req.Category, baseMoney, now)
	if err != nil {
		return "", err
	}

	// 2. Domain validation done in constructor

	// 3. Build commit plan
	plan := commitplan.NewPlan()

	// 4. Repo insert mutation
	plan.Add(it.ProductRepo.InsertMut(product))

	// 5. Add outbox events (enriched)
	for _, ev := range product.DomainEvents() {
		eventID := uuid.New().String()
		payload, err := shared.MarshalDomainEventPayload(ev)
		if err != nil {
			return "", err
		}
		plan.Add(it.OutboxRepo.InsertMut(&contracts.OutboxEvent{
			EventID:      eventID,
			EventType:    ev.EventType(),
			AggregateID:  ev.AggregateID(),
			PayloadJSON:  payload,
			Status:       "pending",
			CreatedAtUTC: now,
		}))
	}

	// 6. Apply plan via Committer
	if err := it.Committer.Apply(ctx, plan); err != nil {
		return "", err
	}

	return product.ID(), nil
}
