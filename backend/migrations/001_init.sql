CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE slot_status AS ENUM ('AVAILABLE', 'HELD', 'BOOKED');
CREATE TYPE booking_status AS ENUM ('PENDING', 'CONFIRMED', 'FAILED', 'CANCELLED');
CREATE TYPE payment_status AS ENUM ('REQUIRES_ACTION', 'AUTHORIZED', 'CAPTURED', 'VOIDED');

CREATE TABLE courts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    sport TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE slots (
    id TEXT PRIMARY KEY,
    court_id TEXT NOT NULL REFERENCES courts(id),
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    price_rsd INTEGER NOT NULL CHECK (price_rsd > 0),
    status slot_status NOT NULL DEFAULT 'AVAILABLE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (court_id, start_at)
);

CREATE TABLE bookings (
    id TEXT PRIMARY KEY,
    google_booking_id TEXT UNIQUE,
    slot_id TEXT NOT NULL REFERENCES slots(id),
    customer_name TEXT NOT NULL,
    customer_email TEXT NOT NULL,
    customer_phone TEXT NOT NULL,
    status booking_status NOT NULL DEFAULT 'PENDING',
    payment_intent_id TEXT NOT NULL UNIQUE,
    payment_status payment_status NOT NULL,
    confirmation_token TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX one_confirmed_booking_per_slot
    ON bookings(slot_id)
    WHERE status = 'CONFIRMED';

CREATE TABLE payment_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    booking_id TEXT REFERENCES bookings(id),
    payment_intent_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE outbox_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    attempts INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ
);

INSERT INTO courts (id, name, sport) VALUES
    ('pitch-5a', 'Mini pitch A', 'Football 5v5'),
    ('pitch-5b', 'Mini pitch B', 'Football 5v5'),
    ('pitch-7', 'Main pitch', 'Football 7v7')
ON CONFLICT (id) DO NOTHING;
