// Package command is the CQRS write side: every state mutation goes through
// one of these commands and its handler.
package command

import (
	"context"
	"errors"
	"fmt"

	"azan-backend/internal/cqrs"
	"azan-backend/internal/domain"
	"azan-backend/internal/storage"
)

type RegisterDevice struct {
	Platform  string `json:"platform"`
	PushToken string `json:"push_token"`
}

func (RegisterDevice) CommandName() string { return "RegisterDevice" }

type SetHomeLocation struct {
	DeviceID string  `json:"-"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	RadiusM  float64 `json:"radius_m"`
}

func (SetHomeLocation) CommandName() string { return "SetHomeLocation" }

// ReportLocation is sent by the mobile app whenever the device moves. The
// handler resolves the geofence rule (azan sound at home, normal notification
// once >= radius_m away from home) and projects the result into device_state.
type ReportLocation struct {
	DeviceID string  `json:"-"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

func (ReportLocation) CommandName() string { return "ReportLocation" }

type ReportLocationResult struct {
	Mode        domain.NotificationMode `json:"notification_mode"`
	DistanceM   *float64                `json:"distance_m"`
	ModeChanged bool                    `json:"mode_changed"`
	HomeSet     bool                    `json:"home_set"`
}

type IncrementTasbih struct {
	DeviceID string `json:"-"`
	Dhikr    string `json:"-"`
	Target   int    `json:"target"`
}

func (IncrementTasbih) CommandName() string { return "IncrementTasbih" }

type ResetTasbih struct {
	DeviceID string `json:"-"`
	Dhikr    string `json:"-"`
}

func (ResetTasbih) CommandName() string { return "ResetTasbih" }

type SetPreferences struct {
	DeviceID  string `json:"-"`
	Method    string `json:"method"`
	AsrMethod string `json:"asr_method"`
}

func (SetPreferences) CommandName() string { return "SetPreferences" }

// Handlers wires every command handler onto the bus.
func RegisterHandlers(bus *cqrs.CommandBus, store *storage.Store) {
	bus.Register("RegisterDevice", func(ctx context.Context, c cqrs.Command) (any, error) {
		cmd := c.(RegisterDevice)
		if cmd.Platform == "" {
			cmd.Platform = "unknown"
		}
		id, err := store.CreateDevice(ctx, cmd.Platform, cmd.PushToken)
		if err != nil {
			return nil, err
		}
		return map[string]string{"device_id": id}, nil
	})

	bus.Register("SetHomeLocation", func(ctx context.Context, c cqrs.Command) (any, error) {
		cmd := c.(SetHomeLocation)
		if cmd.Lat < -90 || cmd.Lat > 90 || cmd.Lng < -180 || cmd.Lng > 180 {
			return nil, fmt.Errorf("invalid coordinates")
		}
		if cmd.RadiusM <= 0 {
			cmd.RadiusM = 5 // requested default: azan inside 5 m of home
		}
		if err := store.UpsertHomeLocation(ctx, cmd.DeviceID, cmd.Lat, cmd.Lng, cmd.RadiusM); err != nil {
			return nil, err
		}
		return domain.HomeLocation{DeviceID: cmd.DeviceID, Lat: cmd.Lat, Lng: cmd.Lng, RadiusM: cmd.RadiusM}, nil
	})

	bus.Register("ReportLocation", func(ctx context.Context, c cqrs.Command) (any, error) {
		cmd := c.(ReportLocation)

		home, err := store.GetHomeLocation(ctx, cmd.DeviceID)
		homeSet := true
		var distance *float64
		mode := domain.ModeAzan // until home is configured, keep the full azan experience
		switch {
		case errors.Is(err, storage.ErrNotFound):
			homeSet = false
		case err != nil:
			return nil, err
		default:
			d := domain.HaversineMeters(home.Lat, home.Lng, cmd.Lat, cmd.Lng)
			distance = &d
			mode = domain.ResolveNotificationMode(d, home.RadiusM)
		}

		previous, err := store.ApplyLocationReport(ctx, cmd.DeviceID, cmd.Lat, cmd.Lng, distance, mode)
		if err != nil {
			return nil, err
		}
		return ReportLocationResult{
			Mode:        mode,
			DistanceM:   distance,
			ModeChanged: previous != "" && previous != mode,
			HomeSet:     homeSet,
		}, nil
	})

	bus.Register("IncrementTasbih", func(ctx context.Context, c cqrs.Command) (any, error) {
		cmd := c.(IncrementTasbih)
		if cmd.Target <= 0 {
			cmd.Target = 33
		}
		return store.IncrementTasbih(ctx, cmd.DeviceID, cmd.Dhikr, cmd.Target)
	})

	bus.Register("ResetTasbih", func(ctx context.Context, c cqrs.Command) (any, error) {
		cmd := c.(ResetTasbih)
		if err := store.ResetTasbih(ctx, cmd.DeviceID, cmd.Dhikr); err != nil {
			return nil, err
		}
		return map[string]string{"status": "reset"}, nil
	})

	bus.Register("SetPreferences", func(ctx context.Context, c cqrs.Command) (any, error) {
		cmd := c.(SetPreferences)
		if _, ok := domain.Methods[cmd.Method]; !ok {
			return nil, fmt.Errorf("unknown calculation method %q", cmd.Method)
		}
		if _, ok := domain.AsrFactors[cmd.AsrMethod]; !ok {
			return nil, fmt.Errorf("unknown asr method %q", cmd.AsrMethod)
		}
		if err := store.UpsertPreferences(ctx, cmd.DeviceID, cmd.Method, cmd.AsrMethod); err != nil {
			return nil, err
		}
		return domain.Preferences{DeviceID: cmd.DeviceID, Method: cmd.Method, AsrMethod: cmd.AsrMethod}, nil
	})
}
