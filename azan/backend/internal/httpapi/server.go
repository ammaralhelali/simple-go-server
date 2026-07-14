// Package httpapi exposes the command and query buses over a JSON REST API.
package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"azan-backend/internal/command"
	"azan-backend/internal/cqrs"
	"azan-backend/internal/query"
	"azan-backend/internal/storage"
)

type Server struct {
	commands *cqrs.CommandBus
	queries  *cqrs.QueryBus
}

func New(commands *cqrs.CommandBus, queries *cqrs.QueryBus) *Server {
	return &Server{commands: commands, queries: queries}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Commands (write side)
	mux.HandleFunc("POST /api/v1/devices", s.registerDevice)
	mux.HandleFunc("PUT /api/v1/devices/{id}/home", s.setHomeLocation)
	mux.HandleFunc("POST /api/v1/devices/{id}/location", s.reportLocation)
	mux.HandleFunc("POST /api/v1/devices/{id}/tasbih/{dhikr}/increment", s.incrementTasbih)
	mux.HandleFunc("POST /api/v1/devices/{id}/tasbih/{dhikr}/reset", s.resetTasbih)
	mux.HandleFunc("PUT /api/v1/devices/{id}/preferences", s.setPreferences)

	// Queries (read side)
	mux.HandleFunc("GET /api/v1/prayer-times", s.getPrayerTimes)
	mux.HandleFunc("GET /api/v1/qibla", s.getQibla)
	mux.HandleFunc("GET /api/v1/devices/{id}/notification-mode", s.getNotificationMode)
	mux.HandleFunc("GET /api/v1/devices/{id}/tasbih", s.getTasbih)
	mux.HandleFunc("GET /api/v1/devices/{id}/preferences", s.getPreferences)
	mux.HandleFunc("GET /api/v1/devices/{id}/home", s.getHomeLocation)

	return logRequests(mux)
}

// --- command endpoints ---

func (s *Server) registerDevice(w http.ResponseWriter, r *http.Request) {
	var cmd command.RegisterDevice
	if !decode(w, r, &cmd) {
		return
	}
	s.dispatchCommand(w, r, cmd, http.StatusCreated)
}

func (s *Server) setHomeLocation(w http.ResponseWriter, r *http.Request) {
	var cmd command.SetHomeLocation
	if !decode(w, r, &cmd) {
		return
	}
	cmd.DeviceID = r.PathValue("id")
	s.dispatchCommand(w, r, cmd, http.StatusOK)
}

func (s *Server) reportLocation(w http.ResponseWriter, r *http.Request) {
	var cmd command.ReportLocation
	if !decode(w, r, &cmd) {
		return
	}
	cmd.DeviceID = r.PathValue("id")
	s.dispatchCommand(w, r, cmd, http.StatusOK)
}

func (s *Server) incrementTasbih(w http.ResponseWriter, r *http.Request) {
	cmd := command.IncrementTasbih{DeviceID: r.PathValue("id"), Dhikr: r.PathValue("dhikr")}
	if r.ContentLength > 0 && !decode(w, r, &cmd) {
		return
	}
	s.dispatchCommand(w, r, cmd, http.StatusOK)
}

func (s *Server) resetTasbih(w http.ResponseWriter, r *http.Request) {
	cmd := command.ResetTasbih{DeviceID: r.PathValue("id"), Dhikr: r.PathValue("dhikr")}
	s.dispatchCommand(w, r, cmd, http.StatusOK)
}

func (s *Server) setPreferences(w http.ResponseWriter, r *http.Request) {
	var cmd command.SetPreferences
	if !decode(w, r, &cmd) {
		return
	}
	cmd.DeviceID = r.PathValue("id")
	s.dispatchCommand(w, r, cmd, http.StatusOK)
}

// --- query endpoints ---

func (s *Server) getPrayerTimes(w http.ResponseWriter, r *http.Request) {
	lat, lng, ok := coords(w, r)
	if !ok {
		return
	}
	q := query.GetPrayerTimes{
		Lat:       lat,
		Lng:       lng,
		Date:      time.Now().UTC(),
		Method:    param(r, "method", "MWL"),
		AsrMethod: param(r, "asr", "Standard"),
	}
	if d := r.URL.Query().Get("date"); d != "" {
		parsed, err := time.Parse("2006-01-02", d)
		if err != nil {
			writeError(w, http.StatusBadRequest, "date must be YYYY-MM-DD")
			return
		}
		q.Date = parsed
	}
	if tz := r.URL.Query().Get("tz"); tz != "" {
		offset, err := strconv.ParseFloat(tz, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "tz must be a UTC offset in hours, e.g. 3 or -4.5")
			return
		}
		q.TZOffsetHours = offset
	}
	s.dispatchQuery(w, r, q)
}

func (s *Server) getQibla(w http.ResponseWriter, r *http.Request) {
	lat, lng, ok := coords(w, r)
	if !ok {
		return
	}
	s.dispatchQuery(w, r, query.GetQibla{Lat: lat, Lng: lng})
}

func (s *Server) getNotificationMode(w http.ResponseWriter, r *http.Request) {
	s.dispatchQuery(w, r, query.GetNotificationMode{DeviceID: r.PathValue("id")})
}

func (s *Server) getTasbih(w http.ResponseWriter, r *http.Request) {
	s.dispatchQuery(w, r, query.GetTasbih{DeviceID: r.PathValue("id")})
}

func (s *Server) getPreferences(w http.ResponseWriter, r *http.Request) {
	s.dispatchQuery(w, r, query.GetPreferences{DeviceID: r.PathValue("id")})
}

func (s *Server) getHomeLocation(w http.ResponseWriter, r *http.Request) {
	s.dispatchQuery(w, r, query.GetHomeLocation{DeviceID: r.PathValue("id")})
}

// --- helpers ---

func (s *Server) dispatchCommand(w http.ResponseWriter, r *http.Request, cmd cqrs.Command, okStatus int) {
	result, err := s.commands.Dispatch(r.Context(), cmd)
	if err != nil {
		writeDispatchError(w, err)
		return
	}
	writeJSON(w, okStatus, result)
}

func (s *Server) dispatchQuery(w http.ResponseWriter, r *http.Request, q cqrs.Query) {
	result, err := s.queries.Dispatch(r.Context(), q)
	if err != nil {
		writeDispatchError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeDispatchError(w http.ResponseWriter, err error) {
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	log.Printf("dispatch error: %v", err)
	writeError(w, http.StatusBadRequest, err.Error())
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func coords(w http.ResponseWriter, r *http.Request) (lat, lng float64, ok bool) {
	var err1, err2 error
	lat, err1 = strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, err2 = strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	if err1 != nil || err2 != nil || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		writeError(w, http.StatusBadRequest, "lat and lng query params are required")
		return 0, 0, false
	}
	return lat, lng, true
}

func param(r *http.Request, key, fallback string) string {
	if v := r.URL.Query().Get(key); v != "" {
		return v
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}
