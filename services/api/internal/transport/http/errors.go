package http

import (
	"encoding/json"
	"net/http"
)

const (
	codeMethodNotAllowed     = "method_not_allowed"
	codeNotFound             = "not_found"
	codeInvalidRequestBody   = "invalid_request_body"
	codeMissingRequiredField = "missing_required_field"
	codeInvalidStartsAt      = "invalid_starts_at"
	codeInvalidID            = "invalid_id"
	codeEventNameRequired    = "event_name_required"
	codeZoneNameRequired     = "zone_name_required"
	codeInvalidQuantity      = "invalid_quantity"
	codeInvalidCapacity      = "invalid_capacity"
	codeIdempotencyRequired  = "idempotency_key_required"
	codeIdempotencyConflict  = "idempotency_conflict"
	codeInsufficientCapacity = "insufficient_capacity"
	codeZoneNotFound         = "zone_not_found"
	codeEventNotFound        = "event_not_found"
	codeZoneAlreadyExists    = "zone_already_exists"
	codeHoldNotFound         = "hold_not_found"
	codeHoldExpired          = "hold_expired"
	codeHoldAlreadyConfirmed = "hold_already_confirmed"
	codeForbidden            = "forbidden"
	codeInternalError        = "internal_error"
)

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	payload, err := json.Marshal(errorResponse{
		Error: msg,
		Code:  code,
	})
	if err != nil {
		_, _ = w.Write([]byte(`{"error":"internal error","code":"internal_error"}`))
		return
	}
	_, _ = w.Write(payload)
}
