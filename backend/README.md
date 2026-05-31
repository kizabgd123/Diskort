# FK Sava 45 Google Reserve Backend

This backend replaces the earlier client-only prototype with a production-oriented Go service for Google Reserve booking flows, Stripe PSP orchestration, 3DS/SCA authentication, Redis-style atomic locking, asynchronous notifications, and PostgreSQL persistence design.

## Services

- `GET /healthz` — health check for load balancers and uptime probes.
- `GET /google-reserve/availability?from=<RFC3339>&to=<RFC3339>` — ListAvailability adapter response.
- `POST /google-reserve/checkout` — validates the slot and creates a Stripe checkout/payment-intent session with 3DS/SCA metadata.
- `POST /google-reserve/bookings` — CreateBooking flow: authorize payment, acquire slot lock, create booking, mark slot booked, capture payment, enqueue notification.
- `GET /google-reserve/bookings/{booking_id}` — GetBookingStatus adapter response.

## Architecture

```text
Google Reserve
  -> HTTP adapter
  -> Booking Service
     -> Payment Orchestrator
        -> Stripe PSP
        -> 3DS/SCA challenge service
     -> Redis lock manager (in-memory implementation in this repo; Redis SET NX PX in production)
     -> PostgreSQL repository (schema in migrations/001_init.sql)
     -> BullMQ/SQS-compatible queue publisher for async notifications
```

## Production configuration

The checked-in runtime uses standard-library Go and in-memory adapters so tests run without Docker or external accounts. Production deployment should wire concrete adapters with these environment variables:

- `DATABASE_URL` for PostgreSQL.
- `REDIS_URL` for atomic slot locks and availability cache.
- `STRIPE_SECRET_KEY` and `STRIPE_WEBHOOK_SECRET`.
- `SCA_PROVIDER_URL` and credentials for the 3DS/SCA provider.
- `QUEUE_URL` for SQS or BullMQ-compatible Redis queue workers.
- `HTTP_ADDR` for the public HTTP listener.

## Transaction flow

1. `Checkout` validates slot availability and creates a Stripe payment intent.
2. 3DS/SCA challenge metadata is returned to the Google checkout client.
3. `CreateBooking` authorizes payment before locking, as required by the requested flow.
4. The booking service acquires an atomic slot lock with a short TTL.
5. The booking row is created and the slot is marked booked.
6. Payment is captured only after the booking write succeeds.
7. On any failure after authorization, the service voids the payment and releases/rolls back the slot state.
8. A confirmation notification job is queued asynchronously.

## Local run

```bash
go test ./...
go run ./cmd/reserve
```
