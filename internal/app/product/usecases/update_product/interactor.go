package update_product

import (
	"context"

	"github.com/google/uuid"

	contracts "github.com/murkotick/product-catalog-service/internal/app/product/contracts"
	"github.com/murkotick/product-catalog-service/internal/app/product/domain"
	shared "github.com/murkotick/product-catalog-service/internal/app/product/usecases/shared"
	"github.com/murkotick/product-catalog-service/internal/app/product/utils"
	"github.com/murkotick/product-catalog-service/internal/pkg/clock"
	commitplan "github.com/murkotick/product-catalog-service/internal/pkg/committer"
)

// Request represents the update product request (partial updates allowed).
type Request struct {
	ProductID   string
	Name        *string
	Description *string
	Category    *string
}

// Interactor applies partial updates using the Golden Mutation Pattern.
type Interactor struct {
	ProductRepo contracts.ProductRepo
	OutboxRepo  contracts.OutboxRepo
	Committer   contracts.Committer
	ReadModel   contracts.ReadModel
	Clock       clock.Clock
}

func NewInteractor(repo contracts.ProductRepo, outboxRepo contracts.OutboxRepo, committer contracts.Committer, readModel contracts.ReadModel, clk clock.Clock) *Interactor {
	return &Interactor{
		ProductRepo: repo,
		OutboxRepo:  outboxRepo,
		Committer:   committer,
		ReadModel:   readModel,
		Clock:       clk,
	}
}

func (it *Interactor) Execute(ctx context.Context, req Request) error {
	now := it.Clock.Now()

	// 1. Load aggregate via read model
	dtoOut, err := it.ReadModel.GetProduct(ctx, req.ProductID)
	if err != nil {
		return err
	}

	createdAtPtr := utils.ParseTimePtr(dtoOut.CreatedAt)
	updatedAtPtr := utils.ParseTimePtr(dtoOut.UpdatedAt)
	archivedAtPtr := utils.ParseTimePtr(dtoOut.ArchivedAt)

	description := ""
	if dtoOut.Description != nil {
		description = *dtoOut.Description
	}

	base := domain.NewMoney(dtoOut.BasePriceNum, dtoOut.BasePriceDen)
	product := domain.ReconstructProduct(
		dtoOut.ProductID,
		dtoOut.Name,
		description,
		dtoOut.Category,
		base,
		nil, // discount left nil for update details (we don't need discount to update name/desc/cat)
		domain.ProductStatus(dtoOut.Status),
		utils.TimeOrZero(createdAtPtr),
		utils.TimeOrZero(updatedAtPtr),
		archivedAtPtr,
	)

	// 2. Domain method: pass provided fields or empty strings (UpdateDetails uses non-empty to decide)
	updName := ""
	if req.Name != nil {
		updName = *req.Name
	}
	updDesc := ""
	if req.Description != nil {
		updDesc = *req.Description
	}
	updCategory := ""
	if req.Category != nil {
		updCategory = *req.Category
	}

	if err := product.UpdateDetails(updName, updDesc, updCategory, now); err != nil {
		return err
	}

	// 3. Collect mutations
	plan := commitplan.NewPlan()

	// 4. Repo update mutation
	plan.Add(it.ProductRepo.UpdateMut(product))

	// 5. Outbox events
	for _, ev := range product.DomainEvents() {
		eventID := uuid.New().String()
		payload, err := shared.MarshalDomainEventPayload(ev)
		if err != nil {
			return err
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

	// 6. Apply via committer
	if err := it.Committer.Apply(ctx, plan); err != nil {
		return err
	}

	return nil
}
