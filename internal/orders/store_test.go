package orders

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// newTestStore starts Postgres in a container. Needs Docker.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	if testing.Short() {
		t.Skip("needs Docker; skipped with -short")
	}
	ctx := context.Background()
	ctr, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("orders"),
		postgres.WithUsername("orders"),
		postgres.WithPassword("orders"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = ctr.Terminate(context.Background()) })

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	s := NewStore(pool)
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestStoreCreateThenGet(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	created, err := s.Create(ctx, NewOrder{CustomerID: "c-1", Currency: "EUR", AmountCents: 1250})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 || created.Status != "placed" || created.CreatedAt.IsZero() {
		t.Fatalf("unexpected order: %+v", created)
	}
	got, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("created_at: got %v, want %v", got.CreatedAt, created.CreatedAt)
	}
	got.CreatedAt, created.CreatedAt = time.Time{}, time.Time{}
	if got != created {
		t.Fatalf("got %+v, want %+v", got, created)
	}
}

func TestStoreListNewestFirst(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	a, err := s.Create(ctx, NewOrder{CustomerID: "c-1", Currency: "EUR", AmountCents: 100})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Create(ctx, NewOrder{CustomerID: "c-1", Currency: "EUR", AmountCents: 200})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, NewOrder{CustomerID: "c-2", Currency: "SEK", AmountCents: 300}); err != nil {
		t.Fatal(err)
	}
	list, err := s.List(ctx, "c-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != b.ID || list[1].ID != a.ID {
		t.Fatalf("got %+v, want orders %d then %d", list, b.ID, a.ID)
	}
}

func TestStoreGetMissingOrder(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Get(context.Background(), 424242)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestStoreMigrateTwice(t *testing.T) {
	s := newTestStore(t)
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestStoreRejectsInvalidOrder(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create(context.Background(), NewOrder{CustomerID: "c-1", Currency: "EUR"}); err == nil {
		t.Fatal("want an error for a zero amount")
	}
}
