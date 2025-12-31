package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/app"
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

// HoldCreator is the minimal interface needed to create a hold.
type HoldCreator interface {
	CreateHold(rctx context.Context, in app.CreateHoldInput) (domain.Hold, error)
}

// HandleCreateHold returns an HTTP handler for creating holds.
func HandleCreateHold(svc HoldCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req createHoldRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := req.validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		hold, err := svc.CreateHold(r.Context(), app.CreateHoldInput{
			EventID:        req.EventID,
			ZoneID:         req.ZoneID,
			Quantity:       req.Quantity,
			IdempotencyKey: req.IdempotencyKey,
		})
		if err != nil {
			switch err {
			case domain.ErrInvalidQuantity:
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			case domain.ErrInvalidID:
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			case domain.ErrIdempotencyKeyRequired:
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			case domain.ErrZoneNotFound:
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			case domain.ErrIdempotencyConflict:
				http.Error(w, err.Error(), http.StatusConflict)
				return
			case domain.ErrInsufficientCapacity:
				http.Error(w, err.Error(), http.StatusConflict)
				return
			default:
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		}

		resp := createHoldResponse{
			ID:        hold.ID,
			Status:    string(hold.Status),
			ExpiresAt: hold.ExpiresAt,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

type createHoldRequest struct {
	EventID        string `json:"event_id"`
	ZoneID         string `json:"zone_id"`
	Quantity       int    `json:"quantity"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (r createHoldRequest) validate() error {
	if r.EventID == "" || r.ZoneID == "" {
		return errors.New("event_id and zone_id are required")
	}
	if r.IdempotencyKey == "" {
		return domain.ErrIdempotencyKeyRequired
	}
	if r.Quantity <= 0 {
		return domain.ErrInvalidQuantity
	}
	return nil
}

type createHoldResponse struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
}
