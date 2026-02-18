package contracts

import (
	"time"

	"cloud.google.com/go/spanner"
)

// OutboxRepo is the write-side repository interface for the transactional outbox.
// It returns Spanner mutations; it does not apply them.
type OutboxRepo interface {
	InsertMut(e *OutboxEvent) *spanner.Mutation
}

// OutboxEvent is the application-level representation of an event persisted to the outbox table.
// Usecases are responsible for enriching domain events into this structure.
type OutboxEvent struct {
	EventID      string
	EventType    string
	AggregateID  string
	PayloadJSON  string
	Status       string
	CreatedAtUTC time.Time
}
