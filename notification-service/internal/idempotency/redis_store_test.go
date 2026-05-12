package idempotency_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"notification-service/internal/idempotency"
)

func newTestStore(t *testing.T) (*idempotency.RedisStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return idempotency.NewRedisStore(rdb), mr
}

// Test 1: Seen returns false for an event that was never marked.
func TestRedisStore_Seen_UnknownEvent(t *testing.T) {
	store, _ := newTestStore(t)
	seen, err := store.Seen(context.Background(), "evt-unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seen {
		t.Fatal("expected Seen=false for unknown event, got true")
	}
}

// Test 2: After Mark, Seen returns true.
func TestRedisStore_MarkThenSeen(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	id := "evt-abc"

	if err := store.Mark(ctx, id); err != nil {
		t.Fatalf("Mark: %v", err)
	}
	seen, err := store.Seen(ctx, id)
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}
	if !seen {
		t.Fatal("expected Seen=true after Mark, got false")
	}
}

// Test 3: Seen returns false after TTL expires.
func TestRedisStore_Seen_ExpiredKey(t *testing.T) {
	store, mr := newTestStore(t)
	ctx := context.Background()
	id := "evt-ttl"

	if err := store.Mark(ctx, id); err != nil {
		t.Fatalf("Mark: %v", err)
	}

	// Fast-forward past TTL
	mr.FastForward(25 * time.Hour)

	seen, err := store.Seen(ctx, id)
	if err != nil {
		t.Fatalf("Seen after expiry: %v", err)
	}
	if seen {
		t.Fatal("expected Seen=false after TTL expiry, got true")
	}
}

// Test 4: Marking the same event twice does not error.
func TestRedisStore_DoubleMarkIsIdempotent(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	id := "evt-dup"

	if err := store.Mark(ctx, id); err != nil {
		t.Fatalf("first Mark: %v", err)
	}
	if err := store.Mark(ctx, id); err != nil {
		t.Fatalf("second Mark: %v", err)
	}
	seen, _ := store.Seen(ctx, id)
	if !seen {
		t.Fatal("expected Seen=true after double Mark")
	}
}

// Test 5: Separate event IDs are tracked independently.
func TestRedisStore_IndependentEvents(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	_ = store.Mark(ctx, "evt-1")

	seen1, _ := store.Seen(ctx, "evt-1")
	seen2, _ := store.Seen(ctx, "evt-2")

	if !seen1 {
		t.Fatal("expected evt-1 to be seen")
	}
	if seen2 {
		t.Fatal("expected evt-2 to be unseen")
	}
}
