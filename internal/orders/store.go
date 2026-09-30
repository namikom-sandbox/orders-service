package orders

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schema string

const columns = "id, customer_id, currency, amount_cents, status, created_at"

// Store keeps orders in Postgres.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns a store that uses pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Migrate creates the tables. It runs on every start.
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// Create stores a new order and returns it.
func (s *Store) Create(ctx context.Context, n NewOrder) (Order, error) {
	if err := n.Validate(); err != nil {
		return Order{}, err
	}
	row := s.pool.QueryRow(ctx,
		`INSERT INTO orders (customer_id, currency, amount_cents)
		 VALUES ($1, $2, $3)
		 RETURNING `+columns,
		n.CustomerID, n.Currency, n.AmountCents)
	o, err := scanOrder(row)
	if err != nil {
		return Order{}, fmt.Errorf("create order: %w", err)
	}
	return o, nil
}

// Get returns one order.
func (s *Store) Get(ctx context.Context, id int64) (Order, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+columns+` FROM orders WHERE id = $1`, id)
	o, err := scanOrder(row)
	if err != nil {
		return Order{}, fmt.Errorf("get order %d: %w", id, err)
	}
	return o, nil
}

// List returns a customer's orders, newest first.
func (s *Store) List(ctx context.Context, customerID string) ([]Order, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+columns+` FROM orders WHERE customer_id = $1
		 ORDER BY created_at DESC, id DESC`, customerID)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()
	out := []Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("list orders: %w", err)
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	return out, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanOrder(r scanner) (Order, error) {
	var o Order
	err := r.Scan(&o.ID, &o.CustomerID, &o.Currency, &o.AmountCents, &o.Status, &o.CreatedAt)
	return o, err
}
