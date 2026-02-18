package shared

import (
	"encoding/json"
	"fmt"

	"github.com/murkotick/product-catalog-service/internal/app/product/domain"
)

// MarshalDomainEventPayload converts a domain event into a JSON payload suitable for the outbox.
//
// The domain layer intentionally avoids serialization concerns; this adapter extracts primitives
// (e.g., Money as numerator/denominator) to keep payloads useful.
func MarshalDomainEventPayload(ev domain.DomainEvent) (string, error) {
	if ev == nil {
		return "{}", nil
	}

	switch e := ev.(type) {
	case *domain.ProductCreatedEvent:
		payload := map[string]interface{}{
			"product_id": e.ProductID,
			"name":       e.Name,
			"category":   e.Category,
			"base_price": map[string]interface{}{
				"numerator":   e.BasePrice.Numerator(),
				"denominator": e.BasePrice.Denominator(),
			},
			"created_at": e.CreatedAt,
		}
		b, err := json.Marshal(payload)
		return string(b), err

	case *domain.ProductUpdatedEvent:
		payload := map[string]interface{}{
			"product_id":  e.ProductID,
			"changes":     e.Changes,
			"updated_at":  e.UpdatedAt,
			"occurred_at": e.OccurredAt(),
		}
		b, err := json.Marshal(payload)
		return string(b), err

	case *domain.ProductActivatedEvent:
		payload := map[string]interface{}{
			"product_id":   e.ProductID,
			"activated_at": e.ActivatedAt,
			"occurred_at":  e.OccurredAt(),
			"event_type":   e.EventType(),
			"aggregate_id": e.AggregateID(),
		}
		b, err := json.Marshal(payload)
		return string(b), err

	case *domain.ProductDeactivatedEvent:
		payload := map[string]interface{}{
			"product_id":     e.ProductID,
			"deactivated_at": e.DeactivatedAt,
			"occurred_at":    e.OccurredAt(),
		}
		b, err := json.Marshal(payload)
		return string(b), err

	case *domain.ProductArchivedEvent:
		payload := map[string]interface{}{
			"product_id":  e.ProductID,
			"archived_at": e.ArchivedAt,
			"occurred_at": e.OccurredAt(),
		}
		b, err := json.Marshal(payload)
		return string(b), err

	case *domain.DiscountAppliedEvent:
		payload := map[string]interface{}{
			"product_id":          e.ProductID,
			"discount_percent":    e.DiscountPercent,
			"discount_start_date": e.DiscountStartDate,
			"discount_end_date":   e.DiscountEndDate,
			"applied_at":          e.AppliedAt,
			"occurred_at":         e.OccurredAt(),
		}
		b, err := json.Marshal(payload)
		return string(b), err

	case *domain.DiscountRemovedEvent:
		payload := map[string]interface{}{
			"product_id":  e.ProductID,
			"removed_at":  e.RemovedAt,
			"occurred_at": e.OccurredAt(),
		}
		b, err := json.Marshal(payload)
		return string(b), err

	case *domain.PriceChangedEvent:
		payload := map[string]interface{}{
			"product_id": e.ProductID,
			"old_price": map[string]interface{}{
				"numerator":   e.OldPrice.Numerator(),
				"denominator": e.OldPrice.Denominator(),
			},
			"new_price": map[string]interface{}{
				"numerator":   e.NewPrice.Numerator(),
				"denominator": e.NewPrice.Denominator(),
			},
			"changed_at":  e.ChangedAt,
			"occurred_at": e.OccurredAt(),
		}
		b, err := json.Marshal(payload)
		return string(b), err
	}

	// Fallback: try to marshal the event directly.
	b, err := json.Marshal(ev)
	if err != nil {
		return "", fmt.Errorf("marshal outbox payload for %T: %w", ev, err)
	}
	return string(b), nil
}
