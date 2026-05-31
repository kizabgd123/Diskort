# FK Sava 45 Independent Reservation System

## Business and location model

FK Sava 45 is treated as a football-first community venue in Block 45, New Belgrade, with bookable small-sided pitches and strong evening/weekend demand from nearby residents. Public directory information places the FK Sava 45 football stadium in Blok 45/New Belgrade, with PlanPlus listing the venue as FK Sava 45 Football Stadium and the address area around Vojvođanska bb. Local Block 45 guides describe the neighborhood as a dense residential area on the Sava riverbank with extensive public transport access, which supports a mobile-first booking funnel for recurring recreational teams and short-notice players.

The commercial model implemented in this repository is:

- **Hourly court inventory:** deterministic slot IDs by court/date/start time.
- **Online reservation deposit or full prepayment:** Stripe Billing is the default processor and Square can be selected as an alternate checkout provider.
- **Venue operations override:** Firebase is the real-time source of truth, allowing staff to block, cancel, or confirm slots from an admin console or future back-office screen.
- **Advertising attribution:** Google Tag Manager and Meta Pixel events track slot selection, lead submission, and payment intent.
- **BI analytics:** reservation documents and front-end events are structured for export into BigQuery/Looker Studio or another warehouse.

## Cloud architecture

```text
Static React UI
  ├─ Firebase Firestore realtime listener: fkSava45Reservations
  ├─ Firestore document ID create semantics to prevent duplicate booking
  ├─ Stripe/Square checkout handoff for tokenized payments
  ├─ Google Calendar event URL and Calendar API sync queue flag
  ├─ Google Maps directions URL for venue navigation
  └─ GTM dataLayer + Meta Pixel conversion events
```

## Data synchronization and duplicate prevention

Each reservation uses the slot ID as the Firestore document ID. The client creates the Firestore reservation document with the slot ID as `documentId`; Firestore rejects the create if the document already exists, and staff-only updates can cancel or confirm existing reservations. This gives clients a fast local UI while keeping the cloud write path authoritative.

Recommended Firestore rule shape:

```js
match /fkSava45Reservations/{slotId} {
  allow read: if true;
  allow create: if request.resource.data.slotId == slotId
    && request.resource.data.status in ['pending_payment', 'confirmed'];
  allow update: if request.auth.token.admin == true;
}
```

## Production configuration checklist

Copy `public/config.example.js` to `public/config.js` or inject `window.FK_SAVA45_CONFIG` with these keys before deployment:

- `FIREBASE_API_KEY`
- `FIREBASE_PROJECT_ID`
- `STRIPE_PUBLISHABLE_KEY`
- `SQUARE_CHECKOUT_URL`

Server-side Cloud Functions should add:

- Stripe or Square secret keys.
- Payment intent/session creation.
- Payment webhook verification.
- Google Calendar API service-account sync.
- Split-payment transfer logic for venue/operator distribution.
- Reserve with Google compatible tokenized transaction payloads.

## Testing plan

- **Data transfer speed:** measure time from date change to first Firestore snapshot and keep p95 under 1 second on Serbian mobile networks.
- **Stability under load:** run concurrent transaction attempts for the same slot and confirm only one active reservation document is created.
- **Payment resilience:** replay webhooks and verify idempotent status changes.
- **Analytics QA:** verify GTM preview and Meta Events Manager receive `booking_step` and purchase-intent events with slot IDs.
