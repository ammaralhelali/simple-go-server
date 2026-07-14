-- Azan app schema. Write models + read models (device_state is the projection
-- the query side reads; location_events is the append-only event log).

CREATE TABLE IF NOT EXISTS devices (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform    TEXT NOT NULL DEFAULT 'unknown',
    push_token  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS home_locations (
    device_id   UUID PRIMARY KEY REFERENCES devices(id) ON DELETE CASCADE,
    lat         DOUBLE PRECISION NOT NULL,
    lng         DOUBLE PRECISION NOT NULL,
    radius_m    DOUBLE PRECISION NOT NULL DEFAULT 5,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Read-model projection: current notification mode per device.
CREATE TABLE IF NOT EXISTS device_state (
    device_id          UUID PRIMARY KEY REFERENCES devices(id) ON DELETE CASCADE,
    notification_mode  TEXT NOT NULL DEFAULT 'azan',
    distance_m         DOUBLE PRECISION,
    last_lat           DOUBLE PRECISION,
    last_lng           DOUBLE PRECISION,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Append-only event log of location reports and mode transitions.
CREATE TABLE IF NOT EXISTS location_events (
    id          BIGSERIAL PRIMARY KEY,
    device_id   UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    event_type  TEXT NOT NULL, -- 'location_reported' | 'mode_changed'
    lat         DOUBLE PRECISION,
    lng         DOUBLE PRECISION,
    distance_m  DOUBLE PRECISION,
    mode        TEXT,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_location_events_device ON location_events (device_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS tasbih_counters (
    device_id   UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    dhikr       TEXT NOT NULL,
    count       INTEGER NOT NULL DEFAULT 0,
    target      INTEGER NOT NULL DEFAULT 33,
    total       INTEGER NOT NULL DEFAULT 0, -- lifetime total, survives resets
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (device_id, dhikr)
);

CREATE TABLE IF NOT EXISTS preferences (
    device_id   UUID PRIMARY KEY REFERENCES devices(id) ON DELETE CASCADE,
    method      TEXT NOT NULL DEFAULT 'MWL',
    asr_method  TEXT NOT NULL DEFAULT 'Standard',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
