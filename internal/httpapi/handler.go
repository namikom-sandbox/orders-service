// Package httpapi serves the orders HTTP API.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/namikom-sandbox/orders-service/internal/orders"
)

// Repository is what the API needs from storage.
type Repository interface {
	Create(ctx context.Context, n orders.NewOrder) (orders.Order, error)
	Get(ctx context.Context, id int64) (orders.Order, error)
	List(ctx context.Context, customerID string) ([]orders.Order, error)
}

// New returns the API's handler.
func New(repo Repository, log *slog.Logger) http.Handler {
	h := &handler{repo: repo, log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", h.create)
	mux.HandleFunc("GET /orders/{id}", h.get)
	mux.HandleFunc("GET /orders", h.list)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

type handler struct {
	repo Repository
	log  *slog.Logger
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	var n orders.NewOrder
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be JSON")
		return
	}
	if err := n.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	o, err := h.repo.Create(r.Context(), n)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id must be a positive integer")
		return
	}
	o, err := h.repo.Get(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("customer_id")
	if customerID == "" {
		writeError(w, http.StatusBadRequest, "customer_id is required")
		return
	}
	list, err := h.repo.List(r.Context(), customerID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, orders.ErrNotFound):
		writeError(w, http.StatusNotFound, "order not found")
	case errors.Is(err, orders.ErrInvalid):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		h.log.ErrorContext(r.Context(), "request failed",
			"method", r.Method, "path", r.URL.Path, "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
