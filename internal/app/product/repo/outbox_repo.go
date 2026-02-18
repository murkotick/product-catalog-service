package repo

import (
	"cloud.google.com/go/spanner"

	contracts "github.com/murkotick/product-catalog-service/internal/app/product/contracts"
	"github.com/murkotick/product-catalog-service/internal/models/m_outbox"
)

// OutboxRepo is the Spanner implementation of the transactional outbox repository.
// It returns *spanner.Mutation but never applies it.
type OutboxRepo struct{}

func NewOutboxRepo() *OutboxRepo {
	return &OutboxRepo{}
}

func (r *OutboxRepo) InsertMut(e *contracts.OutboxEvent) *spanner.Mutation {
	if e == nil {
		return nil
	}

	values := m_outbox.BuildInsertMap(
		e.EventID,
		e.EventType,
		e.AggregateID,
		e.PayloadJSON,
		e.Status,
		e.CreatedAtUTC,
	)
	return m_outbox.InsertMutation(values)
}
