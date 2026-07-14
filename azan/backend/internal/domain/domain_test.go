package domain

import (
	"math"
	"testing"
	"time"
)

func TestQiblaBearing(t *testing.T) {
	cases := []struct {
		name     string
		lat, lng float64
		want     float64
	}{
		{"Cairo", 30.0444, 31.2357, 136.1},
		{"London", 51.5074, -0.1278, 118.99},
		{"Jakarta", -6.2088, 106.8456, 295.15},
	}
	for _, c := range cases {
		got := QiblaBearing(c.lat, c.lng)
		if math.Abs(got-c.want) > 1.5 {
			t.Errorf("%s: qibla = %.2f, want ~%.2f", c.name, got, c.want)
		}
	}
}

func TestHaversine(t *testing.T) {
	// Two points ~5 m apart (1e-4 deg lat ≈ 11.1 m, so 4.5e-5 ≈ 5 m).
	d := HaversineMeters(30.0, 31.0, 30.000045, 31.0)
	if d < 4 || d > 6 {
		t.Errorf("expected ~5m, got %.2f", d)
	}
	if HaversineMeters(30, 31, 30, 31) != 0 {
		t.Error("identical points must be 0 m apart")
	}
}

func TestResolveNotificationMode(t *testing.T) {
	if ResolveNotificationMode(3, 5) != ModeAzan {
		t.Error("inside radius must be azan mode")
	}
	if ResolveNotificationMode(5.1, 5) != ModeNormal {
		t.Error("outside radius must be normal mode")
	}
}

func TestCalculatePrayerTimes(t *testing.T) {
	date := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	// Cairo, UTC+3 (EEST)
	pt, err := CalculatePrayerTimes(date, 30.0444, 31.2357, 3, "MWL", "Standard")
	if err != nil {
		t.Fatal(err)
	}

	ordered := []time.Time{pt.Fajr, pt.Sunrise, pt.Dhuhr, pt.Asr, pt.Maghrib, pt.Isha}
	for i := 1; i < len(ordered); i++ {
		if !ordered[i].After(ordered[i-1]) {
			t.Fatalf("prayer times out of order: %v", ordered)
		}
	}

	// Solar noon in Cairo mid-July is close to 12:00 + ~55m (lng/tz offset + eqt).
	if pt.Dhuhr.Hour() < 11 || pt.Dhuhr.Hour() > 13 {
		t.Errorf("dhuhr looks wrong: %v", pt.Dhuhr)
	}
	// Fajr with an 18° angle in summer Cairo should land between 03:00 and 04:30.
	if pt.Fajr.Hour() < 3 || pt.Fajr.Hour() > 4 {
		t.Errorf("fajr looks wrong: %v", pt.Fajr)
	}

	if _, err := CalculatePrayerTimes(date, 30, 31, 3, "nope", "Standard"); err == nil {
		t.Error("expected error for unknown method")
	}
}

func TestIshaMinutesMethod(t *testing.T) {
	date := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	pt, err := CalculatePrayerTimes(date, 21.4225, 39.8262, 3, "Makkah", "Standard")
	if err != nil {
		t.Fatal(err)
	}
	gap := pt.Isha.Sub(pt.Maghrib)
	if gap < 89*time.Minute || gap > 91*time.Minute {
		t.Errorf("Makkah method: isha should be 90m after maghrib, got %v", gap)
	}
}
