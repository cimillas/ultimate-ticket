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

// AdminEventService is the minimal interface needed for admin event endpoints.
type AdminEventService interface {
	CreateEvent(ctx context.Context, in app.CreateEventInput) (domain.Event, error)
	ListEvents(ctx context.Context) ([]domain.Event, error)
}

// AdminZoneService is the minimal interface needed for admin zone endpoints.
type AdminZoneService interface {
	CreateZone(ctx context.Context, in app.CreateZoneInput) (domain.Zone, error)
	ListZones(ctx context.Context, eventID string) ([]domain.Zone, error)
}

// HandleAdminEvents returns an HTTP handler for admin event creation/listing.
func HandleAdminEvents(svc AdminEventService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			events, err := svc.ListEvents(r.Context())
			if err != nil {
				writeError(w, http.StatusInternalServerError, codeInternalError, "internal error")
				return
			}
			resp := make([]eventResponse, 0, len(events))
			for _, event := range events {
				resp = append(resp, eventResponse{
					ID:       event.ID,
					Name:     event.Name,
					StartsAt: event.StartsAt,
				})
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		case http.MethodPost:
			var req createEventRequest
			dec := json.NewDecoder(r.Body)
			dec.DisallowUnknownFields()
			if err := dec.Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, codeInvalidRequestBody, "invalid request body")
				return
			}
			if req.Name == "" {
				writeError(w, http.StatusBadRequest, codeEventNameRequired, domain.ErrEventNameRequired.Error())
				return
			}

			var startsAt *time.Time
			if req.StartsAt != "" {
				parsed, err := time.Parse(time.RFC3339, req.StartsAt)
				if err != nil {
					writeError(w, http.StatusBadRequest, codeInvalidStartsAt, "invalid starts_at format")
					return
				}
				startsAt = &parsed
			}

			event, err := svc.CreateEvent(r.Context(), app.CreateEventInput{
				Name:     req.Name,
				StartsAt: startsAt,
			})
			if err != nil {
				switch err {
				case domain.ErrEventNameRequired:
					writeError(w, http.StatusBadRequest, codeEventNameRequired, err.Error())
				default:
					writeError(w, http.StatusInternalServerError, codeInternalError, "internal error")
				}
				return
			}

			resp := eventResponse{
				ID:       event.ID,
				Name:     event.Name,
				StartsAt: event.StartsAt,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(resp)
			return
		default:
			writeError(w, http.StatusMethodNotAllowed, codeMethodNotAllowed, "method not allowed")
			return
		}
	}
}

// HandleAdminZones returns an HTTP handler for admin zone creation/listing.
func HandleAdminZones(svc AdminZoneService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventID, ok := parseAdminEventZonesPath(r.URL.Path)
		if !ok {
			writeError(w, http.StatusNotFound, codeNotFound, "not found")
			return
		}

		switch r.Method {
		case http.MethodGet:
			zones, err := svc.ListZones(r.Context(), eventID)
			if err != nil {
				switch err {
				case domain.ErrInvalidID:
					writeError(w, http.StatusNotFound, codeInvalidID, err.Error())
				case domain.ErrEventNotFound:
					writeError(w, http.StatusNotFound, codeEventNotFound, err.Error())
				default:
					writeError(w, http.StatusInternalServerError, codeInternalError, "internal error")
				}
				return
			}
			resp := make([]zoneResponse, 0, len(zones))
			for _, zone := range zones {
				resp = append(resp, zoneResponse{
					ID:       zone.ID,
					EventID:  zone.EventID,
					Name:     zone.Name,
					Capacity: zone.Capacity,
				})
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		case http.MethodPost:
			var req createZoneRequest
			dec := json.NewDecoder(r.Body)
			dec.DisallowUnknownFields()
			if err := dec.Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, codeInvalidRequestBody, "invalid request body")
				return
			}
			if req.Name == "" {
				writeError(w, http.StatusBadRequest, codeZoneNameRequired, domain.ErrZoneNameRequired.Error())
				return
			}
			if req.Capacity <= 0 {
				writeError(w, http.StatusBadRequest, codeInvalidCapacity, domain.ErrInvalidCapacity.Error())
				return
			}

			zone, err := svc.CreateZone(r.Context(), app.CreateZoneInput{
				EventID:  eventID,
				Name:     req.Name,
				Capacity: req.Capacity,
			})
			if err != nil {
				switch err {
				case domain.ErrInvalidID:
					writeError(w, http.StatusNotFound, codeInvalidID, err.Error())
				case domain.ErrZoneNameRequired, domain.ErrInvalidCapacity:
					code := codeInvalidCapacity
					if err == domain.ErrZoneNameRequired {
						code = codeZoneNameRequired
					}
					writeError(w, http.StatusBadRequest, code, err.Error())
				case domain.ErrEventNotFound:
					writeError(w, http.StatusNotFound, codeEventNotFound, err.Error())
				case domain.ErrZoneAlreadyExists:
					writeError(w, http.StatusConflict, codeZoneAlreadyExists, err.Error())
				default:
					writeError(w, http.StatusInternalServerError, codeInternalError, "internal error")
				}
				return
			}

			resp := zoneResponse{
				ID:       zone.ID,
				EventID:  zone.EventID,
				Name:     zone.Name,
				Capacity: zone.Capacity,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(resp)
			return
		default:
			writeError(w, http.StatusMethodNotAllowed, codeMethodNotAllowed, "method not allowed")
			return
		}
	}
}

type createEventRequest struct {
	Name     string `json:"name"`
	StartsAt string `json:"starts_at,omitempty"`
}

type eventResponse struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	StartsAt time.Time `json:"starts_at"`
}

type createZoneRequest struct {
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
}

type zoneResponse struct {
	ID       string `json:"id"`
	EventID  string `json:"event_id"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
}

func parseAdminEventZonesPath(path string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 {
		return "", false
	}
	if parts[0] != "admin" || parts[1] != "events" || parts[3] != "zones" {
		return "", false
	}
	if parts[2] == "" {
		return "", false
	}
	return parts[2], true
}
