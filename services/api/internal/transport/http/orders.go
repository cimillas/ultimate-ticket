package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

// OrderLister is the minimal interface needed to list orders.
type OrderLister interface {
	ListOrdersByUser(ctx context.Context, userID string) ([]domain.OrderSummary, error)
}

// HandleOrders returns an HTTP handler for listing orders.
func HandleOrders(svc OrderLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, codeMethodNotAllowed, "method not allowed")
			return
		}
		session, ok := AuthFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, codeUnauthorized, domain.ErrUnauthorized.Error())
			return
		}

		orders, err := svc.ListOrdersByUser(r.Context(), session.User.ID)
		if err != nil {
			switch err {
			case domain.ErrUnauthorized:
				writeError(w, http.StatusUnauthorized, codeUnauthorized, err.Error())
			case domain.ErrInvalidID:
				writeError(w, http.StatusBadRequest, codeInvalidID, err.Error())
			default:
				writeError(w, http.StatusInternalServerError, codeInternalError, "internal error")
			}
			return
		}

		resp := make([]orderSummaryResponse, 0, len(orders))
		for _, order := range orders {
			resp = append(resp, orderSummaryResponse{
				ID:        order.ID,
				HoldID:    order.HoldID,
				EventID:   order.EventID,
				ZoneID:    order.ZoneID,
				Quantity:  order.Quantity,
				Status:    string(order.Status),
				CreatedAt: order.CreatedAt,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

type orderSummaryResponse struct {
	ID        string    `json:"id"`
	HoldID    string    `json:"hold_id"`
	EventID   string    `json:"event_id"`
	ZoneID    string    `json:"zone_id"`
	Quantity  int       `json:"quantity"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
