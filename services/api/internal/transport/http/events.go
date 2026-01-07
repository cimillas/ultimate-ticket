package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

type PublicEventService interface {
	ListEvents(ctx context.Context) ([]domain.Event, error)
}

type PublicZoneService interface {
	ListZones(ctx context.Context, eventID string) ([]domain.Zone, error)
}

func HandleEvents(svc PublicEventService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, codeMethodNotAllowed, "method not allowed")
			return
		}

		events, err := svc.ListEvents(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, codeInternalError, "internal error")
			return
		}

		resp := make([]eventResponse, 0, len(events))
		for _, event := range events {
			resp = append(resp, eventResponse{
				ID:          event.ID,
				Name:        event.Name,
				StartsAt:    event.StartsAt,
				Status:      string(event.Status),
				CancelledAt: event.CancelledAt,
				IsComplete:  event.IsComplete,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func HandleEventZones(svc PublicZoneService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventID, ok := parseEventZonesPath(r.URL.Path)
		if !ok {
			writeError(w, http.StatusNotFound, codeNotFound, "not found")
			return
		}
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, codeMethodNotAllowed, "method not allowed")
			return
		}

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
				ID:        zone.ID,
				EventID:   zone.EventID,
				Name:      zone.Name,
				Capacity:  zone.Capacity,
				Available: zone.Available,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func parseEventZonesPath(path string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 {
		return "", false
	}
	if parts[0] != "events" || parts[2] != "zones" {
		return "", false
	}
	if parts[1] == "" {
		return "", false
	}
	return parts[1], true
}
