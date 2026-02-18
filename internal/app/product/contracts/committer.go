package contracts

import (
	"context"

	commitplan "github.com/murkotick/product-catalog-service/internal/pkg/committer"
)

// Committer is a small abstraction the usecases call to apply a collection
// of mutations atomically. This keeps usecases independent of commitplan
// driver details.
//
// Note: Your existing pkg/committer/plan.go may already provide a richer
// wrapper for commitplan. If so, adapt its exported Apply method to satisfy
// this interface (or change this interface to match it). The goal here is
// to keep usecases simple.
type Committer interface {
	// Apply atomically applies the provided mutation plan.
	//
	// Note: in production this would typically be backed by github.com/Vektor-AI/commitplan
	// with the Spanner driver. For this test task, we keep the interface minimal and
	// allow swapping the implementation without touching usecases.
	Apply(ctx context.Context, plan *commitplan.Plan) error
}
