// Package query is the CQRS read side: queries never mutate state.
package query

import (
	"context"
	"time"

	"azan-backend/internal/cqrs"
	"azan-backend/internal/domain"
	"azan-backend/internal/storage"
)

type GetPrayerTimes struct {
	Lat, Lng      float64
	Date          time.Time
	TZOffsetHours float64
	Method        string
	AsrMethod     string
}

func (GetPrayerTimes) QueryName() string { return "GetPrayerTimes" }

type GetQibla struct {
	Lat, Lng float64
}

func (GetQibla) QueryName() string { return "GetQibla" }

type QiblaResult struct {
	Bearing         float64 `json:"bearing"`
	DistanceToKaaba float64 `json:"distance_to_kaaba_m"`
}

type GetNotificationMode struct {
	DeviceID string
}

func (GetNotificationMode) QueryName() string { return "GetNotificationMode" }

type GetTasbih struct {
	DeviceID string
}

func (GetTasbih) QueryName() string { return "GetTasbih" }

type GetPreferences struct {
	DeviceID string
}

func (GetPreferences) QueryName() string { return "GetPreferences" }

type GetHomeLocation struct {
	DeviceID string
}

func (GetHomeLocation) QueryName() string { return "GetHomeLocation" }

func RegisterHandlers(bus *cqrs.QueryBus, store *storage.Store) {
	bus.Register("GetPrayerTimes", func(ctx context.Context, q cqrs.Query) (any, error) {
		qq := q.(GetPrayerTimes)
		return domain.CalculatePrayerTimes(qq.Date, qq.Lat, qq.Lng, qq.TZOffsetHours, qq.Method, qq.AsrMethod)
	})

	bus.Register("GetQibla", func(ctx context.Context, q cqrs.Query) (any, error) {
		qq := q.(GetQibla)
		return QiblaResult{
			Bearing:         domain.QiblaBearing(qq.Lat, qq.Lng),
			DistanceToKaaba: domain.HaversineMeters(qq.Lat, qq.Lng, domain.KaabaLat, domain.KaabaLng),
		}, nil
	})

	bus.Register("GetNotificationMode", func(ctx context.Context, q cqrs.Query) (any, error) {
		qq := q.(GetNotificationMode)
		state, err := store.GetDeviceState(ctx, qq.DeviceID)
		if err == storage.ErrNotFound {
			// No location reported yet: default to the full azan experience.
			return domain.DeviceState{DeviceID: qq.DeviceID, NotificationMode: domain.ModeAzan}, nil
		}
		return state, err
	})

	bus.Register("GetTasbih", func(ctx context.Context, q cqrs.Query) (any, error) {
		qq := q.(GetTasbih)
		return store.ListTasbih(ctx, qq.DeviceID)
	})

	bus.Register("GetPreferences", func(ctx context.Context, q cqrs.Query) (any, error) {
		qq := q.(GetPreferences)
		return store.GetPreferences(ctx, qq.DeviceID)
	})

	bus.Register("GetHomeLocation", func(ctx context.Context, q cqrs.Query) (any, error) {
		qq := q.(GetHomeLocation)
		return store.GetHomeLocation(ctx, qq.DeviceID)
	})
}
