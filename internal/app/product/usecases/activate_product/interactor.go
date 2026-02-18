package activate_product

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

// Request for activating a product
type Request struct {
	ProductID string
}

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

	// 1. Load aggregate via ReadModel and reconstruct
	dto, err := it.ReadModel.GetProduct(ctx, req.ProductID)
	if err != nil {
		return err
	}

	createdAtPtr := utils.ParseTimePtr(dto.CreatedAt)
	updatedAtPtr := utils.ParseTimePtr(dto.UpdatedAt)
	archivedAtPtr := utils.ParseTimePtr(dto.ArchivedAt)

	base := domain.NewMoney(dto.BasePriceNum, dto.BasePriceDen)
	product := domain.ReconstructProduct(
		dto.ProductID,
		dto.Name,
		"",
		dto.Category,
		base,
		nil,
		domain.ProductStatus(dto.Status),
		utils.TimeOrZero(createdAtPtr),
		utils.TimeOrZero(updatedAtPtr),
		archivedAtPtr,
	)

	// 2. Domain call
	if err := product.Activate(now); err != nil {
		return err
	}

	// 3. Build commit plan
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

	// 6. Apply plan
	return it.Committer.Apply(ctx, plan)
}
