# FK Sava 45 Independent Reservation System

## Business and location model

FK Sava 45 is treated as a football-first community venue in Block 45, New Belgrade, with bookable small-sided pitches and strong evening/weekend demand from nearby residents. Public directory information places the FK Sava 45 football stadium in Blok 45/New Belgrade, with PlanPlus listing the venue as FK Sava 45 Football Stadium and the address area around Vojvođanska bb. Local Block 45 guides describe the neighborhood as a dense residential area on the Sava riverbank with extensive public transport access, which supports a mobile-first booking funnel for recurring recreational teams and short-notice players.

The commercial model implemented in this repository is:

- **Hourly court inventory:** deterministic slot IDs by court/date/start time.
- **Online reservation deposit or full prepayment:** Stripe Billing is the default processor and Square remains documented as a secondary PSP option for future adapters.
- **Venue operations override:** PostgreSQL is the durable source of truth, Redis provides short-lived atomic locks, and a queue/outbox supports confirmations and operational updates.
- **Advertising attribution:** Google Tag Manager and Meta Pixel events track slot selection, lead submission, and payment intent in the static front-end prototype.
- **BI analytics:** reservation rows, payment events, and queue/outbox data are structured for export into BigQuery/Looker Studio or another warehouse.

## Backend architecture

```text
Google Reserve
  ├─ ListAvailability
  ├─ Checkout
  ├─ CreateBooking
  └─ GetBookingStatus
      -> Go HTTP adapter
      -> Booking Service
         -> Payment Orchestrator
            -> Stripe PSP
            -> 3DS/SCA challenge service
         -> Redis atomic lock manager
         -> PostgreSQL booking schema
         -> BullMQ/SQS-style async notification queue
```

The Go backend lives in `backend/` and exposes Google Reserve-ready endpoints for availability, checkout, booking creation, and booking status. The checked-in adapters are in-memory so CI and local development do not require Docker, but the interfaces are designed for PostgreSQL, Redis, Stripe, a 3DS/SCA provider, and SQS/BullMQ-style workers.

## Data synchronization and duplicate prevention

Each bookable slot has a deterministic slot ID. The production database schema enforces one confirmed booking per slot with a partial unique index, while the booking service also acquires a short-lived lock before committing the booking. In production the lock manager should use Redis `SET key value NX PX <ttl>` semantics; the repository includes an in-memory equivalent for tests.

The `CreateBooking` flow intentionally authorizes payment before acquiring the slot lock, then captures payment only after the slot is successfully booked. If slot locking, booking persistence, or capture fails, rollback logic releases the lock, restores slot state where needed, and voids the payment authorization.

## PostgreSQL schema

The initial schema is in `backend/migrations/001_init.sql` and includes:

- `courts` for venue inventory.
- `slots` for bookable time windows.
- `bookings` for Google booking IDs, customer details, status, and Stripe payment intent IDs.
- `payment_events` for PSP webhook/audit history.
- `outbox_jobs` for asynchronous notification and BI export work.

## Production configuration checklist

Front-end demo configuration can be injected through `public/config.example.js`. The backend production deployment should provide:

- `DATABASE_URL` for PostgreSQL.
- `REDIS_URL` for atomic locks and availability cache.
- `STRIPE_SECRET_KEY` and `STRIPE_WEBHOOK_SECRET`.
- `SCA_PROVIDER_URL` and credentials for a 3DS/SCA provider.
- `QUEUE_URL` for SQS or BullMQ-compatible queue workers.
- `HTTP_ADDR` for the backend listener.

## Testing plan

- **End-to-end Google Reserve workflow:** run backend tests covering checkout, CreateBooking, payment capture, status lookup, and duplicate rejection.
- **Data transfer speed:** measure ListAvailability p95 and keep cache-backed reads under 1 second on Serbian mobile networks.
- **Stability under load:** run concurrent CreateBooking calls for the same slot and confirm only one booking is confirmed.
- **Payment resilience:** replay Stripe webhooks and verify idempotent payment status changes.
- **Analytics QA:** verify GTM preview and Meta Events Manager receive front-end `booking_step` and purchase-intent events with slot IDs.
