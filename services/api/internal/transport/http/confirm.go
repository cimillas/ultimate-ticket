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
			writeError(w, http.StatusMethodNotAllowed, codeMethodNotAllowed, "method not allowed")
			return
		}

		holdID, ok := parseConfirmHoldPath(r.URL.Path)
		if !ok {
			writeError(w, http.StatusNotFound, codeNotFound, "not found")
			return
		}

		key := r.Header.Get(idempotencyHeader)
		if key == "" {
			writeError(w, http.StatusBadRequest, codeIdempotencyRequired, domain.ErrIdempotencyKeyRequired.Error())
			return
		}

		res, err := svc.ConfirmHold(r.Context(), app.ConfirmHoldInput{
			HoldID:         holdID,
			IdempotencyKey: key,
		})
		if err != nil {
			switch err {
			case domain.ErrHoldNotFound:
				writeError(w, http.StatusNotFound, codeHoldNotFound, err.Error())
				return
			case domain.ErrInvalidID:
				writeError(w, http.StatusNotFound, codeInvalidID, err.Error())
				return
			case domain.ErrHoldExpired, domain.ErrHoldAlreadyConfirmed:
				code := codeHoldAlreadyConfirmed
				if err == domain.ErrHoldExpired {
					code = codeHoldExpired
				}
				writeError(w, http.StatusConflict, code, err.Error())
				return
			case domain.ErrIdempotencyKeyRequired:
				writeError(w, http.StatusBadRequest, codeIdempotencyRequired, err.Error())
				return
			default:
				writeError(w, http.StatusInternalServerError, codeInternalError, "internal error")
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
