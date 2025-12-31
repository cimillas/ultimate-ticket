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
			writeError(w, http.StatusMethodNotAllowed, codeMethodNotAllowed, "method not allowed")
			return
		}

		var req createHoldRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, codeInvalidRequestBody, "invalid request body")
			return
		}
		if err := req.validate(); err != nil {
			switch err {
			case errEventZoneRequired:
				writeError(w, http.StatusBadRequest, codeMissingRequiredField, err.Error())
			case domain.ErrIdempotencyKeyRequired:
				writeError(w, http.StatusBadRequest, codeIdempotencyRequired, err.Error())
			case domain.ErrInvalidQuantity:
				writeError(w, http.StatusBadRequest, codeInvalidQuantity, err.Error())
			default:
				writeError(w, http.StatusBadRequest, codeInvalidRequestBody, err.Error())
			}
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
				writeError(w, http.StatusBadRequest, codeInvalidQuantity, err.Error())
				return
			case domain.ErrInvalidID:
				writeError(w, http.StatusBadRequest, codeInvalidID, err.Error())
				return
			case domain.ErrIdempotencyKeyRequired:
				writeError(w, http.StatusBadRequest, codeIdempotencyRequired, err.Error())
				return
			case domain.ErrZoneNotFound:
				writeError(w, http.StatusNotFound, codeZoneNotFound, err.Error())
				return
			case domain.ErrIdempotencyConflict:
				writeError(w, http.StatusConflict, codeIdempotencyConflict, err.Error())
				return
			case domain.ErrInsufficientCapacity:
				writeError(w, http.StatusConflict, codeInsufficientCapacity, err.Error())
				return
			default:
				writeError(w, http.StatusInternalServerError, codeInternalError, "internal error")
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

var errEventZoneRequired = errors.New("event_id and zone_id are required")

func (r createHoldRequest) validate() error {
	if r.EventID == "" || r.ZoneID == "" {
		return errEventZoneRequired
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
