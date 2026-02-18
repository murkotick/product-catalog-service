package m_outbox

const (
	TableName = "outbox_events"

	ColEventID     = "event_id"
	ColEventType   = "event_type"
	ColAggregateID = "aggregate_id"
	ColPayload     = "payload"
	ColStatus      = "status"
	ColCreatedAt   = "created_at"
	ColProcessedAt = "processed_at"
)
