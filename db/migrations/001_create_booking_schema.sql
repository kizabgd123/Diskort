CREATE TABLE courts (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  sport TEXT NOT NULL,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE booking_slots (
  id TEXT PRIMARY KEY,
  court_id TEXT NOT NULL REFERENCES courts(id),
  starts_at TIMESTAMPTZ NOT NULL,
  ends_at TIMESTAMPTZ NOT NULL,
  price_minor INTEGER NOT NULL,
  currency CHAR(3) NOT NULL DEFAULT 'RSD',
  status TEXT NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'locked', 'booked', 'cancelled')),
  version INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (court_id, starts_at)
);

CREATE TABLE bookings (
  id TEXT PRIMARY KEY,
  google_booking_id TEXT UNIQUE,
  slot_id TEXT NOT NULL REFERENCES booking_slots(id),
  customer_name TEXT NOT NULL,
  customer_phone TEXT NOT NULL,
  customer_email TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending_payment', 'confirmed', 'cancelled', 'failed')),
  payment_intent_id TEXT UNIQUE,
  payment_authorization_id TEXT,
  payment_capture_id TEXT,
  amount_minor INTEGER NOT NULL,
  currency CHAR(3) NOT NULL DEFAULT 'RSD',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE payment_events (
  id BIGSERIAL PRIMARY KEY,
  booking_id TEXT REFERENCES bookings(id),
  payment_intent_id TEXT,
  event_type TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE notification_outbox (
  id BIGSERIAL PRIMARY KEY,
  booking_id TEXT NOT NULL REFERENCES bookings(id),
  channel TEXT NOT NULL CHECK (channel IN ('email', 'sms', 'webhook')),
  payload JSONB NOT NULL,
  status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'sent', 'failed')),
  attempts INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  sent_at TIMESTAMPTZ
);

CREATE INDEX idx_booking_slots_availability ON booking_slots (starts_at, status, court_id);
CREATE INDEX idx_bookings_slot_status ON bookings (slot_id, status);
