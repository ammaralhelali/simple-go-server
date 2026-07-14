// Package storage is the PostgreSQL persistence layer shared by the command
// (write) and query (read) sides.
package storage

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"azan-backend/internal/domain"
)

//go:embed schema.sql
var schemaSQL string

var ErrNotFound = errors.New("not found")

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	var pool *pgxpool.Pool
	var err error
	// The database container may still be starting; retry briefly.
	for attempt := 0; attempt < 10; attempt++ {
		pool, err = pgxpool.New(ctx, databaseURL)
		if err == nil {
			if err = pool.Ping(ctx); err == nil {
				break
			}
			pool.Close()
		}
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, schemaSQL)
	return err
}

func (s *Store) Close() { s.pool.Close() }

// --- devices ---

func (s *Store) CreateDevice(ctx context.Context, platform, pushToken string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO devices (platform, push_token) VALUES ($1, NULLIF($2, '')) RETURNING id`,
		platform, pushToken).Scan(&id)
	return id, err
}

// --- home location ---

func (s *Store) UpsertHomeLocation(ctx context.Context, deviceID string, lat, lng, radiusM float64) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO home_locations (device_id, lat, lng, radius_m, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (device_id) DO UPDATE
		SET lat = EXCLUDED.lat, lng = EXCLUDED.lng, radius_m = EXCLUDED.radius_m, updated_at = now()`,
		deviceID, lat, lng, radiusM)
	return err
}

func (s *Store) GetHomeLocation(ctx context.Context, deviceID string) (domain.HomeLocation, error) {
	var h domain.HomeLocation
	err := s.pool.QueryRow(ctx,
		`SELECT device_id, lat, lng, radius_m FROM home_locations WHERE device_id = $1`,
		deviceID).Scan(&h.DeviceID, &h.Lat, &h.Lng, &h.RadiusM)
	if errors.Is(err, pgx.ErrNoRows) {
		return h, ErrNotFound
	}
	return h, err
}

// --- device state (read model) + location events (event log) ---

// ApplyLocationReport atomically appends the location event, updates the
// projection, and records a mode_changed event when the geofence transition
// happens. Returns the previous mode ("" if none).
func (s *Store) ApplyLocationReport(ctx context.Context, deviceID string, lat, lng float64, distanceM *float64, mode domain.NotificationMode) (domain.NotificationMode, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var previous domain.NotificationMode
	err = tx.QueryRow(ctx,
		`SELECT notification_mode FROM device_state WHERE device_id = $1 FOR UPDATE`,
		deviceID).Scan(&previous)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO device_state (device_id, notification_mode, distance_m, last_lat, last_lng, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (device_id) DO UPDATE
		SET notification_mode = EXCLUDED.notification_mode, distance_m = EXCLUDED.distance_m,
		    last_lat = EXCLUDED.last_lat, last_lng = EXCLUDED.last_lng, updated_at = now()`,
		deviceID, string(mode), distanceM, lat, lng)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO location_events (device_id, event_type, lat, lng, distance_m, mode)
		VALUES ($1, 'location_reported', $2, $3, $4, $5)`,
		deviceID, lat, lng, distanceM, string(mode))
	if err != nil {
		return "", err
	}

	if previous != "" && previous != mode {
		_, err = tx.Exec(ctx, `
			INSERT INTO location_events (device_id, event_type, distance_m, mode)
			VALUES ($1, 'mode_changed', $2, $3)`,
			deviceID, distanceM, string(mode))
		if err != nil {
			return "", err
		}
	}

	return previous, tx.Commit(ctx)
}

func (s *Store) GetDeviceState(ctx context.Context, deviceID string) (domain.DeviceState, error) {
	var st domain.DeviceState
	err := s.pool.QueryRow(ctx, `
		SELECT device_id, notification_mode, distance_m, last_lat, last_lng, updated_at
		FROM device_state WHERE device_id = $1`,
		deviceID).Scan(&st.DeviceID, &st.NotificationMode, &st.DistanceM, &st.LastLat, &st.LastLng, &st.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return st, ErrNotFound
	}
	return st, err
}

// --- tasbih ---

func (s *Store) IncrementTasbih(ctx context.Context, deviceID, dhikr string, target int) (domain.TasbihCounter, error) {
	var t domain.TasbihCounter
	err := s.pool.QueryRow(ctx, `
		INSERT INTO tasbih_counters (device_id, dhikr, count, target, total, updated_at)
		VALUES ($1, $2, 1, $3, 1, now())
		ON CONFLICT (device_id, dhikr) DO UPDATE
		SET count = CASE WHEN tasbih_counters.count + 1 > tasbih_counters.target THEN 1
		                 ELSE tasbih_counters.count + 1 END,
		    total = tasbih_counters.total + 1,
		    updated_at = now()
		RETURNING device_id, dhikr, count, target, total`,
		deviceID, dhikr, target).Scan(&t.DeviceID, &t.Dhikr, &t.Count, &t.Target, &t.Total)
	return t, err
}

func (s *Store) ResetTasbih(ctx context.Context, deviceID, dhikr string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE tasbih_counters SET count = 0, updated_at = now()
		WHERE device_id = $1 AND dhikr = $2`,
		deviceID, dhikr)
	return err
}

func (s *Store) ListTasbih(ctx context.Context, deviceID string) ([]domain.TasbihCounter, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT device_id, dhikr, count, target, total FROM tasbih_counters
		WHERE device_id = $1 ORDER BY dhikr`,
		deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counters := []domain.TasbihCounter{}
	for rows.Next() {
		var t domain.TasbihCounter
		if err := rows.Scan(&t.DeviceID, &t.Dhikr, &t.Count, &t.Target, &t.Total); err != nil {
			return nil, err
		}
		counters = append(counters, t)
	}
	return counters, rows.Err()
}

// --- preferences ---

func (s *Store) UpsertPreferences(ctx context.Context, deviceID, method, asrMethod string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO preferences (device_id, method, asr_method, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (device_id) DO UPDATE
		SET method = EXCLUDED.method, asr_method = EXCLUDED.asr_method, updated_at = now()`,
		deviceID, method, asrMethod)
	return err
}

func (s *Store) GetPreferences(ctx context.Context, deviceID string) (domain.Preferences, error) {
	p := domain.Preferences{DeviceID: deviceID, Method: "MWL", AsrMethod: "Standard"}
	err := s.pool.QueryRow(ctx,
		`SELECT method, asr_method FROM preferences WHERE device_id = $1`,
		deviceID).Scan(&p.Method, &p.AsrMethod)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, nil // defaults
	}
	return p, err
}
