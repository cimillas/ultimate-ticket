package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/app"
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

const idempotencyHeader = "Idempotency-Key"

// HoldConfirmer is the minimal interface needed to confirm a hold.
type HoldConfirmer interface {
	ConfirmHold(ctx context.Context, in app.ConfirmHoldInput) (app.ConfirmHoldResult, error)
}

// HandleConfirmHold returns an HTTP handler for confirming holds.
func HandleConfirmHold(svc HoldConfirmer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		holdID, ok := parseConfirmHoldPath(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}

		key := r.Header.Get(idempotencyHeader)
		if key == "" {
			http.Error(w, domain.ErrIdempotencyKeyRequired.Error(), http.StatusBadRequest)
			return
		}

		res, err := svc.ConfirmHold(r.Context(), app.ConfirmHoldInput{
			HoldID:         holdID,
			IdempotencyKey: key,
		})
		if err != nil {
			switch err {
			case domain.ErrHoldNotFound:
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			case domain.ErrInvalidID:
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			case domain.ErrHoldExpired, domain.ErrHoldAlreadyConfirmed:
				http.Error(w, err.Error(), http.StatusConflict)
				return
			case domain.ErrIdempotencyKeyRequired:
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			default:
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		}

		resp := confirmHoldResponse{
			ID:        res.Order.ID,
			HoldID:    res.Order.HoldID,
			Status:    "confirmed",
			CreatedAt: res.Order.CreatedAt,
		}

		w.Header().Set("Content-Type", "application/json")
		if res.Created {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func parseConfirmHoldPath(path string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 {
		return "", false
	}
	if parts[0] != "holds" || parts[2] != "confirm" {
		return "", false
	}
	if parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

type confirmHoldResponse struct {
	ID        string    `json:"id"`
	HoldID    string    `json:"hold_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
