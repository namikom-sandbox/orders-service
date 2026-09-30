// Package orders holds the order model and its Postgres store.
package orders

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

// ErrNotFound is returned when an order doesn't exist.
var ErrNotFound = errors.New("order not found")

// ErrInvalid is returned when a new order fails validation.
var ErrInvalid = errors.New("invalid order")

// Order is a placed order.
type Order struct {
	ID          int64     `json:"id"`
	CustomerID  string    `json:"customer_id"`
	Currency    string    `json:"currency"`
	AmountCents int64     `json:"amount_cents"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewOrder is what a client sends to place an order.
type NewOrder struct {
	CustomerID  string `json:"customer_id"`
	Currency    string `json:"currency"`
	AmountCents int64  `json:"amount_cents"`
}

var currencyCode = regexp.MustCompile(`^[A-Z]{3}$`)

// Validate checks a new order before it is stored.
func (n NewOrder) Validate() error {
	switch {
	case n.CustomerID == "":
		return fmt.Errorf("%w: customer_id is required", ErrInvalid)
	case !currencyCode.MatchString(n.Currency):
		return fmt.Errorf("%w: currency must be an ISO 4217 code", ErrInvalid)
	case n.AmountCents <= 0:
		return fmt.Errorf("%w: amount_cents must be positive", ErrInvalid)
	}
	return nil
}
