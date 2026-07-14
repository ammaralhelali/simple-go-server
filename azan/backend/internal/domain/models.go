package domain

import "time"

type HomeLocation struct {
	DeviceID string  `json:"device_id"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	RadiusM  float64 `json:"radius_m"`
}

type DeviceState struct {
	DeviceID         string           `json:"device_id"`
	NotificationMode NotificationMode `json:"notification_mode"`
	DistanceM        *float64         `json:"distance_m"`
	LastLat          *float64         `json:"last_lat"`
	LastLng          *float64         `json:"last_lng"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

type TasbihCounter struct {
	DeviceID string `json:"device_id"`
	Dhikr    string `json:"dhikr"`
	Count    int    `json:"count"`
	Target   int    `json:"target"`
	Total    int    `json:"total"`
}

type Preferences struct {
	DeviceID  string `json:"device_id"`
	Method    string `json:"method"`
	AsrMethod string `json:"asr_method"`
}
