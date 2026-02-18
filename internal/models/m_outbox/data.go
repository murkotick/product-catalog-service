package m_outbox

import (
	"time"

	"cloud.google.com/go/spanner"
)

// BuildInsertMap constructs a map with fields for outbox insertion.
func BuildInsertMap(eventID, eventType, aggregateID string, payload string, status string, createdAt time.Time) map[string]interface{} {
	return map[string]interface{}{
		ColEventID:     eventID,
		ColEventType:   eventType,
		ColAggregateID: aggregateID,
		ColPayload:     payload,
		ColStatus:      status,
		ColCreatedAt:   createdAt,
		ColProcessedAt: nil,
	}
}

// InsertMutation constructs a mutation for the outbox table.
func InsertMutation(values map[string]interface{}) *spanner.Mutation {
	cols := make([]string, 0, len(values))
	vals := make([]interface{}, 0, len(values))
	for c, v := range values {
		cols = append(cols, c)
		vals = append(vals, v)
	}
	return spanner.Insert(TableName, cols, vals)
}
