package domain

import (
	"fmt"
	"math"
	"time"
)

// CalculationMethod holds the sun angles a convention uses for Fajr and Isha.
// IshaMinutes > 0 means Isha is a fixed offset after Maghrib instead of an angle.
type CalculationMethod struct {
	Name        string
	FajrAngle   float64
	IshaAngle   float64
	IshaMinutes float64
}

var Methods = map[string]CalculationMethod{
	"MWL":     {Name: "Muslim World League", FajrAngle: 18, IshaAngle: 17},
	"ISNA":    {Name: "Islamic Society of North America", FajrAngle: 15, IshaAngle: 15},
	"Egypt":   {Name: "Egyptian General Authority of Survey", FajrAngle: 19.5, IshaAngle: 17.5},
	"Makkah":  {Name: "Umm Al-Qura University, Makkah", FajrAngle: 18.5, IshaMinutes: 90},
	"Karachi": {Name: "University of Islamic Sciences, Karachi", FajrAngle: 18, IshaAngle: 18},
}

// AsrFactors: shadow-length factor. Standard (Shafi'i, Maliki, Hanbali) = 1, Hanafi = 2.
var AsrFactors = map[string]float64{
	"Standard": 1,
	"Hanafi":   2,
}

// PrayerTimes are clock times (in the requested UTC offset) for one calendar day.
type PrayerTimes struct {
	Date    string    `json:"date"`
	Fajr    time.Time `json:"fajr"`
	Sunrise time.Time `json:"sunrise"`
	Dhuhr   time.Time `json:"dhuhr"`
	Asr     time.Time `json:"asr"`
	Maghrib time.Time `json:"maghrib"`
	Isha    time.Time `json:"isha"`
	Method  string    `json:"method"`
	Qibla   float64   `json:"qibla"`
}

// calculator implements the widely used PrayTimes.org astronomical algorithm.
// All trigonometry below operates in degrees.
type calculator struct {
	lat, lng float64
	jd       float64 // julian date shifted by longitude so times come out in local solar days
}

func sind(d float64) float64  { return math.Sin(deg2rad(d)) }
func cosd(d float64) float64  { return math.Cos(deg2rad(d)) }
func tand(d float64) float64  { return math.Tan(deg2rad(d)) }
func asind(x float64) float64 { return rad2deg(math.Asin(x)) }
func acosd(x float64) float64 { return rad2deg(math.Acos(x)) }
func atan2d(y, x float64) float64 {
	return rad2deg(math.Atan2(y, x))
}

func fixAngle(a float64) float64 {
	a = math.Mod(a, 360)
	if a < 0 {
		a += 360
	}
	return a
}

func fixHour(h float64) float64 {
	h = math.Mod(h, 24)
	if h < 0 {
		h += 24
	}
	return h
}

func julianDate(year, month, day int) float64 {
	if month <= 2 {
		year--
		month += 12
	}
	a := math.Floor(float64(year) / 100)
	b := 2 - a + math.Floor(a/4)
	return math.Floor(365.25*float64(year+4716)) +
		math.Floor(30.6001*float64(month+1)) +
		float64(day) + b - 1524.5
}

// sunPosition returns the sun's declination and the equation of time (hours)
// for a given julian date. Source: US Naval Observatory approximation, as used
// by PrayTimes.org.
func sunPosition(jd float64) (decl, eqt float64) {
	d := jd - 2451545.0
	g := fixAngle(357.529 + 0.98560028*d)
	q := fixAngle(280.459 + 0.98564736*d)
	l := fixAngle(q + 1.915*sind(g) + 0.020*sind(2*g))
	e := 23.439 - 0.00000036*d

	ra := atan2d(cosd(e)*sind(l), cosd(l)) / 15
	decl = asind(sind(e) * sind(l))
	eqt = q/15 - fixHour(ra)
	return decl, eqt
}

// midDay returns solar noon (hours) for the given day portion.
func (c calculator) midDay(portion float64) float64 {
	_, eqt := sunPosition(c.jd + portion)
	return fixHour(12 - eqt)
}

// sunAngleTime returns the hour at which the sun reaches `angle` degrees below
// the horizon. ccw=true means before noon (Fajr/Sunrise side).
func (c calculator) sunAngleTime(angle, portion float64, ccw bool) float64 {
	decl, _ := sunPosition(c.jd + portion)
	noon := c.midDay(portion)
	ratio := (-sind(angle) - sind(decl)*sind(c.lat)) / (cosd(decl) * cosd(c.lat))
	if ratio < -1 || ratio > 1 {
		return math.NaN() // sun never reaches this angle (extreme latitudes)
	}
	t := acosd(ratio) / 15
	if ccw {
		return noon - t
	}
	return noon + t
}

// asrTime returns Asr for the given shadow factor (1 = Standard, 2 = Hanafi).
func (c calculator) asrTime(factor, portion float64) float64 {
	decl, _ := sunPosition(c.jd + portion)
	angle := -rad2deg(math.Atan(1 / (factor + tand(math.Abs(c.lat-decl)))))
	return c.sunAngleTime(angle, portion, false)
}

// CalculatePrayerTimes computes the five daily prayers (plus sunrise) for the
// given date, coordinates and UTC offset (hours).
func CalculatePrayerTimes(date time.Time, lat, lng, tzOffsetHours float64, methodKey, asrKey string) (PrayerTimes, error) {
	method, ok := Methods[methodKey]
	if !ok {
		return PrayerTimes{}, fmt.Errorf("unknown calculation method %q", methodKey)
	}
	asrFactor, ok := AsrFactors[asrKey]
	if !ok {
		return PrayerTimes{}, fmt.Errorf("unknown asr method %q", asrKey)
	}

	c := calculator{
		lat: lat,
		lng: lng,
		jd:  julianDate(date.Year(), int(date.Month()), date.Day()) - lng/(15*24),
	}

	const riseSetAngle = 0.833 // atmospheric refraction + solar disc radius

	// Initial estimates (hours), refined iteratively.
	fajr, sunrise, dhuhr, asr, maghrib, isha := 5.0, 6.0, 12.0, 13.0, 18.0, 18.0
	for i := 0; i < 2; i++ {
		fajr = c.sunAngleTime(method.FajrAngle, fajr/24, true)
		sunrise = c.sunAngleTime(riseSetAngle, sunrise/24, true)
		dhuhr = c.midDay(dhuhr / 24)
		asr = c.asrTime(asrFactor, asr/24)
		maghrib = c.sunAngleTime(riseSetAngle, maghrib/24, false)
		if method.IshaMinutes > 0 {
			isha = maghrib + method.IshaMinutes/60
		} else {
			isha = c.sunAngleTime(method.IshaAngle, isha/24, false)
		}
	}

	// Convert from solar time to clock time at the requested UTC offset.
	adjust := tzOffsetHours - lng/15
	loc := time.FixedZone("client", int(tzOffsetHours*3600))
	toTime := func(hours float64) time.Time {
		if math.IsNaN(hours) {
			return time.Time{}
		}
		h := fixHour(hours + adjust)
		midnight := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
		return midnight.Add(time.Duration(h * float64(time.Hour))).Round(time.Minute)
	}

	return PrayerTimes{
		Date:    date.Format("2006-01-02"),
		Fajr:    toTime(fajr),
		Sunrise: toTime(sunrise),
		Dhuhr:   toTime(dhuhr),
		Asr:     toTime(asr),
		Maghrib: toTime(maghrib),
		Isha:    toTime(isha),
		Method:  methodKey,
		Qibla:   QiblaBearing(lat, lng),
	}, nil
}
