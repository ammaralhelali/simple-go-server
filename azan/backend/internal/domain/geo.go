package domain

import "math"

// Coordinates of the Kaaba in Makkah, the direction of the Qibla.
const (
	KaabaLat = 21.4224779
	KaabaLng = 39.8251832

	earthRadiusM = 6371000.0
)

func deg2rad(d float64) float64 { return d * math.Pi / 180 }
func rad2deg(r float64) float64 { return r * 180 / math.Pi }

// HaversineMeters returns the great-circle distance between two points in meters.
func HaversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := deg2rad(lat2 - lat1)
	dLng := deg2rad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(deg2rad(lat1))*math.Cos(deg2rad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// QiblaBearing returns the initial great-circle bearing from the given point
// towards the Kaaba, in compass degrees (0 = North, clockwise).
func QiblaBearing(lat, lng float64) float64 {
	phi := deg2rad(lat)
	phiK := deg2rad(KaabaLat)
	dLng := deg2rad(KaabaLng - lng)

	y := math.Sin(dLng)
	x := math.Cos(phi)*math.Tan(phiK) - math.Sin(phi)*math.Cos(dLng)
	bearing := rad2deg(math.Atan2(y, x))
	return math.Mod(bearing+360, 360)
}

// NotificationMode is the read-model value the mobile client consumes to pick
// how prayer notifications are delivered.
type NotificationMode string

const (
	// ModeAzan plays the full azan sound — used while the user is at home.
	ModeAzan NotificationMode = "azan"
	// ModeNormal is a regular notification — used once the user has moved
	// beyond the home geofence radius.
	ModeNormal NotificationMode = "normal"
)

// ResolveNotificationMode applies the geofence rule: azan sound inside the
// home radius, a normal notification outside it.
func ResolveNotificationMode(distanceM, radiusM float64) NotificationMode {
	if distanceM <= radiusM {
		return ModeAzan
	}
	return ModeNormal
}
