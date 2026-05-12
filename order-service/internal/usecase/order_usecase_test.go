package usecase_test

import (
	"context"
	"database/sql"
	"errors"
	"order-service/internal/domain"
	"order-service/internal/usecase"
	"testing"
	"time"
)

// --- fakes ---

type fakeRepo struct {
	order  *domain.Order
	saved  []*domain.Order
	status map[string]string
}

func (r *fakeRepo) Save(_ context.Context, o *domain.Order) error {
	r.saved = append(r.saved, o)
	return nil
}
func (r *fakeRepo) FindByID(_ context.Context, id string) (*domain.Order, error) {
	if r.order != nil && r.order.ID == id {
		return r.order, nil
	}
	return nil, sql.ErrNoRows
}
func (r *fakeRepo) FindByIdempotencyKey(_ context.Context, _ string) (*domain.Order, error) {
	return nil, sql.ErrNoRows
}
func (r *fakeRepo) UpdateStatus(_ context.Context, id, status string) error {
	if r.status == nil {
		r.status = map[string]string{}
	}
	r.status[id] = status
	return nil
}
func (r *fakeRepo) FindByStatus(_ context.Context, _ string) ([]*domain.Order, error) {
	return nil, nil
}

type fakeCache struct {
	data    map[string]*domain.Order
	deleted []string
	setCalls int
}

func newFakeCache() *fakeCache { return &fakeCache{data: map[string]*domain.Order{}} }

func (c *fakeCache) Get(_ context.Context, id string) (*domain.Order, error) {
	return c.data[id], nil
}
func (c *fakeCache) Set(_ context.Context, o *domain.Order, _ time.Duration) error {
	c.setCalls++
	c.data[o.ID] = o
	return nil
}
func (c *fakeCache) Delete(_ context.Context, id string) error {
	c.deleted = append(c.deleted, id)
	delete(c.data, id)
	return nil
}

type fakePaymentClient struct{}

func (f *fakePaymentClient) Authorize(_ context.Context, _ string, _ int64) (string, error) {
	return "Authorized", nil
}
func (f *fakePaymentClient) GetPaymentStats(_ context.Context) (*usecase.PaymentStats, error) {
	return &usecase.PaymentStats{}, nil
}

// --- tests ---

// Test 6: GetOrder returns from cache without hitting the repository.
func TestGetOrder_CacheHit(t *testing.T) {
	cached := &domain.Order{ID: "ord-1", Status: "Paid"}
	cache := newFakeCache()
	cache.data["ord-1"] = cached

	repo := &fakeRepo{} // order not in repo — proves cache was used
	uc := usecase.NewOrderUseCase(repo, &fakePaymentClient{}, cache, 5*time.Minute)

	got, err := uc.GetOrder(context.Background(), "ord-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "ord-1" {
		t.Fatalf("expected ord-1, got %s", got.ID)
	}
	if cache.setCalls != 0 {
		t.Fatal("cache.Set should not be called on a cache hit")
	}
}

// Test 7: GetOrder on cache miss fetches from DB and populates cache.
func TestGetOrder_CacheMiss_PopulatesCache(t *testing.T) {
	order := &domain.Order{ID: "ord-2", Status: "Pending"}
	repo := &fakeRepo{order: order}
	cache := newFakeCache()
	uc := usecase.NewOrderUseCase(repo, &fakePaymentClient{}, cache, 5*time.Minute)

	got, err := uc.GetOrder(context.Background(), "ord-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "ord-2" {
		t.Fatalf("expected ord-2, got %s", got.ID)
	}
	if cache.setCalls != 1 {
		t.Fatalf("expected 1 cache.Set call, got %d", cache.setCalls)
	}
	if cache.data["ord-2"] == nil {
		t.Fatal("order was not stored in cache after DB read")
	}
}

// Test 8: CancelOrder invalidates the cache entry.
func TestCancelOrder_InvalidatesCache(t *testing.T) {
	order := &domain.Order{ID: "ord-3", Status: "Pending"}
	repo := &fakeRepo{order: order}
	cache := newFakeCache()
	cache.data["ord-3"] = order
	uc := usecase.NewOrderUseCase(repo, &fakePaymentClient{}, cache, 5*time.Minute)

	_, err := uc.CancelOrder(context.Background(), "ord-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cache.deleted) == 0 {
		t.Fatal("expected cache.Delete to be called after cancel")
	}
	if cache.deleted[0] != "ord-3" {
		t.Fatalf("expected delete for ord-3, got %s", cache.deleted[0])
	}
	if _, ok := cache.data["ord-3"]; ok {
		t.Fatal("order should be removed from cache after cancel")
	}
}

// ensure fakeRepo and fakeCache satisfy the interfaces at compile time
var _ usecase.OrderRepository = (*fakeRepo)(nil)
var _ usecase.OrderCache = (*fakeCache)(nil)
var _ usecase.PaymentClient = (*fakePaymentClient)(nil)

// silence unused import
var _ = errors.New
