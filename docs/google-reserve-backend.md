# Google Reserve Backend with Payments

This backend is a Go implementation for FK Sava 45 reservation orchestration. It is designed to sit behind a production API gateway and expose Google Reserve-style adapter endpoints while coordinating PostgreSQL, Redis, Stripe, 3DS/SCA, queues, notifications, logging, and monitoring.

## Components

- **HTTP adapter:** exposes `ListAvailability`, `Checkout`, `CreateBooking`, and `GetBookingStatus` under `/v1/google-reserve/*`.
- **Booking service:** validates slot availability, authorizes payment before locking, acquires an atomic slot lock, creates the booking, captures payment, and rolls back payment authorization when a later step fails.
- **Payment orchestrator:** coordinates Stripe PSP calls and 3DS/SCA challenge generation.
- **Locking layer:** provides an in-memory implementation for tests/local demos and an interface intended for Redis `SET key value NX EX` semantics in production.
- **Queue and notification layer:** publishes asynchronous booking confirmation jobs; production deployments can replace the in-memory publisher with BullMQ, SQS, or another queue.
- **Persistence:** `db/migrations/001_create_booking_schema.sql` defines PostgreSQL tables for courts, slots, bookings, payment events, and the notification outbox.
- **Monitoring:** structured JSON logs emit booking lifecycle events and enqueue failures.

## Endpoint workflow

### `ListAvailability`

1. Reads available slots for the requested time window.
2. Returns only slots with `available` status.
3. Can be fronted by Redis caching for high-volume Google polling.

### `Checkout`

1. Validates that the requested slot is still available.
2. Creates a Stripe payment intent through the payment orchestrator.
3. Requests a 3DS/SCA challenge URL when required.
4. Returns the checkout session without locking the slot yet, keeping slot locks short-lived.

### `CreateBooking`

1. Authorizes the previously created payment intent.
2. Acquires the slot lock using the atomic locking layer.
3. Re-checks slot availability under the lock.
4. Marks the slot as booked.
5. Captures payment only after the slot is booked.
6. Creates the booking record and enqueues booking confirmation notifications.
7. Cancels/rolls back the payment intent if lock, availability, booking, or capture steps fail.

### `GetBookingStatus`

Returns the booking status, payment identifiers, customer data, and timestamps for Google Reserve reconciliation.

## Production deployment checklist

- Replace `payments.FakeStripe` with a real Stripe client using idempotency keys, PaymentIntent authorization, capture, cancel, and webhook reconciliation.
- Replace `locking.InMemoryLocker` with a Redis implementation using atomic `SET NX EX` plus token-checked release.
- Replace `queue.InMemoryQueue` with BullMQ, SQS, or a cloud-native queue and bind workers to the notification outbox.
- Wire `booking.Store` to PostgreSQL using the schema in `db/migrations/001_create_booking_schema.sql`.
- Add Google Reserve authentication, request signature validation, rate limiting, and API gateway observability.
- Export structured logs and metrics to the production monitoring stack.
