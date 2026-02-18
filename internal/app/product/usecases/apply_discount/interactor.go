package apply_discount

import (
	"context"
	"math/big"
	"time"

	"github.com/google/uuid"

	contracts "github.com/murkotick/product-catalog-service/internal/app/product/contracts"
	"github.com/murkotick/product-catalog-service/internal/app/product/domain"
	shared "github.com/murkotick/product-catalog-service/internal/app/product/usecases/shared"
	"github.com/murkotick/product-catalog-service/internal/app/product/utils"
	"github.com/murkotick/product-catalog-service/internal/pkg/clock"
	commitplan "github.com/murkotick/product-catalog-service/internal/pkg/committer"
)

// Request to apply a discount
type Request struct {
	ProductID  string
	Percentage float64 // 0-100 scale as domain.NewDiscount expects
	StartDate  time.Time
	EndDate    time.Time
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

	// 1. Load aggregate
	dto, err := it.ReadModel.GetProduct(ctx, req.ProductID)
	if err != nil {
		return err
	}

	createdAtPtr := utils.ParseTimePtr(dto.CreatedAt)
	updatedAtPtr := utils.ParseTimePtr(dto.UpdatedAt)
	archivedAtPtr := utils.ParseTimePtr(dto.ArchivedAt)

	description := ""
	if dto.Description != nil {
		description = *dto.Description
	}

	base := domain.NewMoney(dto.BasePriceNum, dto.BasePriceDen)

	// Reconstruct existing discount (if any) so the domain can enforce
	// "only one active discount" properly.
	var existingDiscount *domain.Discount
	if dto.DiscountPct != nil && dto.DiscountStart != nil && dto.DiscountEnd != nil {
		pct := new(big.Rat)
		if _, ok := pct.SetString(*dto.DiscountPct); ok {
			// If the stored value is > 1, treat it as percent (e.g. 25 => 0.25)
			if pct.Cmp(big.NewRat(1, 1)) == 1 {
				pct = new(big.Rat).Quo(pct, big.NewRat(100, 1))
			}
			start := utils.ParseTimePtr(dto.DiscountStart)
			end := utils.ParseTimePtr(dto.DiscountEnd)
			if start != nil && end != nil {
				d, err := domain.NewDiscountFromRat(pct, *start, *end)
				if err != nil {
					return err
				}
				existingDiscount = d
			}
		}
	}
	product := domain.ReconstructProduct(
		dto.ProductID,
		dto.Name,
		description,
		dto.Category,
		base,
		existingDiscount,
		domain.ProductStatus(dto.Status),
		utils.TimeOrZero(createdAtPtr),
		utils.TimeOrZero(updatedAtPtr),
		archivedAtPtr,
	)

	// 2. Create discount domain object
	discount, err := domain.NewDiscount(req.Percentage, req.StartDate, req.EndDate)
	if err != nil {
		return err
	}

	// 2b. Domain call
	if err := product.ApplyDiscount(discount, now); err != nil {
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
