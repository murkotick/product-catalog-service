package committer

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
)

type Adapter struct {
	client *spanner.Client
}

func NewAdapter(client *spanner.Client) *Adapter {
	return &Adapter{client: client}
}

func (a *Adapter) Apply(ctx context.Context, plan *Plan) error {
	if plan == nil || plan.IsEmpty() {
		return nil
	}

	if a.client == nil {
		return fmt.Errorf("committer: spanner client is nil")
	}

	_, err := a.client.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		return tx.BufferWrite(plan.Mutations())
	})
	return err
}
