package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/namikom-sandbox/orders-service/internal/orders"
)

type fakeRepo struct {
	mu     sync.Mutex
	nextID int64
	byID   map[int64]orders.Order
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byID: map[int64]orders.Order{}}
}

func (f *fakeRepo) Create(_ context.Context, n orders.NewOrder) (orders.Order, error) {
	if err := n.Validate(); err != nil {
		return orders.Order{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	o := orders.Order{
		ID:          f.nextID,
		CustomerID:  n.CustomerID,
		Currency:    n.Currency,
		AmountCents: n.AmountCents,
		Status:      "placed",
		CreatedAt:   time.Unix(f.nextID, 0).UTC(),
	}
	f.byID[o.ID] = o
	return o, nil
}

func (f *fakeRepo) Get(_ context.Context, id int64) (orders.Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.byID[id]
	if !ok {
		return orders.Order{}, orders.ErrNotFound
	}
	return o, nil
}

func (f *fakeRepo) List(_ context.Context, customerID string) ([]orders.Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []orders.Order{}
	for id := f.nextID; id > 0; id-- {
		if o, ok := f.byID[id]; ok && o.CustomerID == customerID {
			out = append(out, o)
		}
	}
	return out, nil
}

func newTestHandler() http.Handler {
	return New(newFakeRepo(), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func do(t *testing.T, h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCreateThenGet(t *testing.T) {
	h := newTestHandler()
	rec := do(t, h, "POST", "/orders", `{"customer_id":"c-1","currency":"EUR","amount_cents":1250}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: status %d, body %s", rec.Code, rec.Body)
	}
	var created orders.Order
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	rec = do(t, h, "GET", "/orders/1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get: status %d, body %s", rec.Code, rec.Body)
	}
	var got orders.Order
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got != created {
		t.Fatalf("got %+v, want %+v", got, created)
	}
}

func TestCreateRejectsInvalidOrders(t *testing.T) {
	h := newTestHandler()
	for name, body := range map[string]string{
		"not json":        `{`,
		"no customer":     `{"currency":"EUR","amount_cents":100}`,
		"lowercase code":  `{"customer_id":"c-1","currency":"eur","amount_cents":100}`,
		"zero amount":     `{"customer_id":"c-1","currency":"EUR","amount_cents":0}`,
		"negative amount": `{"customer_id":"c-1","currency":"EUR","amount_cents":-5}`,
	} {
		t.Run(name, func(t *testing.T) {
			if rec := do(t, h, "POST", "/orders", body); rec.Code != http.StatusBadRequest {
				t.Fatalf("status %d, want 400", rec.Code)
			}
		})
	}
}

func TestGetMissingOrderIs404(t *testing.T) {
	if rec := do(t, newTestHandler(), "GET", "/orders/42", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
}

func TestGetRejectsBadID(t *testing.T) {
	for _, id := range []string{"abc", "0", "-1"} {
		if rec := do(t, newTestHandler(), "GET", "/orders/"+id, ""); rec.Code != http.StatusBadRequest {
			t.Fatalf("id %q: status %d, want 400", id, rec.Code)
		}
	}
}

func TestListNeedsCustomer(t *testing.T) {
	if rec := do(t, newTestHandler(), "GET", "/orders", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", rec.Code)
	}
}

func TestListIsEmptyArrayWhenNone(t *testing.T) {
	rec := do(t, newTestHandler(), "GET", "/orders?customer_id=nobody", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("status %d, body %q; want 200 and []", rec.Code, rec.Body)
	}
}
