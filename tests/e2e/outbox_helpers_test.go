package e2e

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/iterator"
)

type outboxEvent struct {
	EventID     string
	EventType   string
	AggregateID string
	Status      string
	CreatedAt   time.Time
}

func mustFetchOutboxEvents(ctx context.Context, t *testing.T, client *spanner.Client, aggregateID string) []outboxEvent {
	t.Helper()
	items, err := fetchOutboxEvents(ctx, client, aggregateID)
	require.NoError(t, err)
	return items
}

func fetchOutboxEvents(ctx context.Context, client *spanner.Client, aggregateID string) ([]outboxEvent, error) {
	stmt := spanner.Statement{
		SQL: `SELECT event_id, event_type, aggregate_id, status, created_at
        FROM outbox_events
        WHERE aggregate_id = @id
        ORDER BY created_at ASC, event_id ASC`,
		Params: map[string]any{"id": aggregateID},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	out := make([]outboxEvent, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		var e outboxEvent
		if err := row.Columns(&e.EventID, &e.EventType, &e.AggregateID, &e.Status, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
}
