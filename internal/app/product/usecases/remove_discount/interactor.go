package remove_discount

import (
	"context"
	"math/big"

	"github.com/google/uuid"

	contracts "github.com/murkotick/product-catalog-service/internal/app/product/contracts"
	"github.com/murkotick/product-catalog-service/internal/app/product/domain"
	shared "github.com/murkotick/product-catalog-service/internal/app/product/usecases/shared"
	"github.com/murkotick/product-catalog-service/internal/app/product/utils"
	"github.com/murkotick/product-catalog-service/internal/pkg/clock"
	commitplan "github.com/murkotick/product-catalog-service/internal/pkg/committer"
)

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
	return &Interactor{ProductRepo: repo, OutboxRepo: outboxRepo, Committer: committer, ReadModel: readModel, Clock: clk}
}

func (it *Interactor) Execute(ctx context.Context, req Request) error {
	now := it.Clock.Now()

	dto, err := it.ReadModel.GetProduct(ctx, req.ProductID)
	if err != nil {
		return err
	}

	createdAtPtr := utils.ParseTimePtr(dto.CreatedAt)
	updatedAtPtr := utils.ParseTimePtr(dto.UpdatedAt)
	archivedAtPtr := utils.ParseTimePtr(dto.ArchivedAt)

	desc := ""
	if dto.Description != nil {
		desc = *dto.Description
	}

	base := domain.NewMoney(dto.BasePriceNum, dto.BasePriceDen)

	var existingDiscount *domain.Discount
	if dto.DiscountPct != nil && dto.DiscountStart != nil && dto.DiscountEnd != nil {
		pct := new(big.Rat)
		if _, ok := pct.SetString(*dto.DiscountPct); ok {
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
		desc,
		dto.Category,
		base,
		existingDiscount,
		domain.ProductStatus(dto.Status),
		utils.TimeOrZero(createdAtPtr),
		utils.TimeOrZero(updatedAtPtr),
		archivedAtPtr,
	)

	if err := product.RemoveDiscount(now); err != nil {
		return err
	}

	plan := commitplan.NewPlan()
	plan.Add(it.ProductRepo.UpdateMut(product))

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

	return it.Committer.Apply(ctx, plan)
}
