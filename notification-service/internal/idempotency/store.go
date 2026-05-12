package idempotency

import "context"

type Store interface {
	Seen(ctx context.Context, id string) (bool, error)
	Mark(ctx context.Context, id string) error
}
