# Azan App

A prayer-times app with location-aware azan notifications:

- **Azan sound at home** — the app watches your location and plays the full azan
  sound for prayer notifications while you are within your home radius.
- **Normal notification away from home** — once you move ≥ 5 m (configurable)
  from home, prayer notifications switch to a regular notification sound.
- **Qibla compass** — live magnetometer compass pointing at the Kaaba.
- **Tasbih counter** — synced dhikr counters (33/66/99 targets, lifetime totals).
- **Prayer times** — five daily prayers + sunrise, multiple calculation methods
  (MWL, ISNA, Egypt, Makkah/Umm Al-Qura, Karachi) and Standard/Hanafi Asr,
  with a next-prayer countdown.

## Architecture

```
azan/
├── backend/    Go 1.24, CQRS, PostgreSQL (pgx)
│   ├── cmd/api/                 entrypoint
│   └── internal/
│       ├── cqrs/                command & query buses
│       ├── command/             write side (RegisterDevice, SetHomeLocation,
│       │                        ReportLocation, IncrementTasbih, ...)
│       ├── query/               read side (GetPrayerTimes, GetQibla,
│       │                        GetNotificationMode, GetTasbih, ...)
│       ├── domain/              prayer-time astronomy, qibla bearing,
│       │                        haversine geofence rule
│       ├── storage/             PostgreSQL repos + schema (device_state is the
│       │                        projection; location_events is the event log)
│       └── httpapi/             JSON REST API over the buses
└── mobile/     React Native (Expo) + TanStack React Query
    └── src/
        ├── api/                 fetch client + React Query hooks
        ├── services/            notification channels & background geofence task
        ├── hooks/               useCurrentLocation
        └── screens/             Prayers, Qibla, Tasbih, Settings
```

### How the home/away azan switching works

1. The user taps **“Set current location as Home”** on the Settings screen
   (`PUT /devices/{id}/home`, default radius 5 m).
2. An Expo background location task reports every fix to
   `POST /devices/{id}/location`.
3. The **ReportLocation command handler** computes the haversine distance to
   home and resolves the mode: `azan` inside the radius, `normal` outside.
   It appends `location_reported` / `mode_changed` events and updates the
   `device_state` read model.
4. The response (and the `GET /devices/{id}/notification-mode` query) tells the
   app which mode is active; on a change it reschedules today's remaining
   prayer notifications on the matching Android channel (`azan` channel with
   the full azan sound vs. default-sound `prayer-normal` channel).

> ⚠️ A 5 m radius is at the limit of consumer GPS accuracy (typically 3–10 m).
> The radius is stored per device and adjustable in Settings if the mode
> flips while you're still inside the house.

## Running the backend

```bash
cd azan
docker compose up --build
# API on http://localhost:8081, PostgreSQL on :5432
```

Or locally against your own PostgreSQL:

```bash
cd azan/backend
DATABASE_URL=postgres://azan:azan@localhost:5432/azan?sslmode=disable go run ./cmd/api
```

Tests: `cd azan/backend && go test ./...`

### API

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/api/v1/devices` | Register device → `{device_id}` |
| PUT | `/api/v1/devices/{id}/home` | Set home `{lat,lng,radius_m}` |
| POST | `/api/v1/devices/{id}/location` | Report location → mode + distance |
| GET | `/api/v1/devices/{id}/notification-mode` | Current mode (read model) |
| GET | `/api/v1/prayer-times?lat&lng&tz&method&asr&date` | Prayer times |
| GET | `/api/v1/qibla?lat&lng` | Qibla bearing + distance to Makkah |
| POST | `/api/v1/devices/{id}/tasbih/{dhikr}/increment` | Count dhikr |
| POST | `/api/v1/devices/{id}/tasbih/{dhikr}/reset` | Reset round |
| GET | `/api/v1/devices/{id}/tasbih` | All counters |
| GET/PUT | `/api/v1/devices/{id}/preferences` | Calculation method prefs |

## Running the mobile app

```bash
cd azan/mobile
npm install
# 1. Drop your azan audio at assets/sounds/azan.wav
# 2. Point API_BASE_URL in src/api/client.ts at your backend
#    (Android emulator: http://10.0.2.2:8081, real device: your LAN IP)
npx expo prebuild   # custom notification sound + background location need a dev build
npx expo run:android   # or run:ios
```

Background location and custom notification sounds do **not** work in Expo Go —
use a development build (`expo prebuild` + `run:android`/`run:ios`) as shown.
