import React, { useEffect, useMemo, useState } from 'https://esm.sh/react@19.1.1';
import { createRoot } from 'https://esm.sh/react-dom@19.1.1/client';
import BookingForm from './components/BookingForm.js';
import CourtCalendar from './components/CourtCalendar.js';
import { isFirebaseConfigured, mapsDirectionsUrl, subscribeToAvailability } from './services/reservations.js';
import { trackBookingStep } from './services/analytics.js';
import './styles.css';

const h = React.createElement;

function todayBelgrade() {
  return new Intl.DateTimeFormat('en-CA', { timeZone: 'Europe/Belgrade' }).format(new Date());
}

function App() {
  const [date, setDate] = useState(todayBelgrade());
  const [slots, setSlots] = useState([]);
  const [selectedSlot, setSelectedSlot] = useState(null);
  const [latencyMs, setLatencyMs] = useState(0);

  useEffect(() => {
    const started = performance.now();
    const unsubscribe = subscribeToAvailability(date, (nextSlots) => {
      setSlots(nextSlots);
      setLatencyMs(Math.round(performance.now() - started));
    });
    return unsubscribe;
  }, [date]);

  const availableCount = useMemo(() => slots.filter((slot) => slot.available).length, [slots]);

  function selectSlot(slot) {
    setSelectedSlot(slot);
    trackBookingStep('slot_selected', { slotId: slot.id, priceRsd: slot.priceRsd });
  }

  return h('main', null,
    h('section', { className: 'hero' },
      h('div', null,
        h('p', { className: 'eyebrow' }, 'FK Sava 45 · Blok 45, New Belgrade'),
        h('h1', null, 'Independent court reservation system for fast, duplicate-safe bookings.'),
        h('p', null, 'Real-time Firebase availability, Google navigation and calendar handoff, Stripe/Square payment routing, GTM and Meta Pixel conversion events, and BI-ready booking telemetry.'),
        h('div', { className: 'hero-actions' },
          h('a', { className: 'primary', href: '#booking' }, 'Book Online'),
          h('a', { href: mapsDirectionsUrl(), target: '_blank', rel: 'noreferrer' }, 'Open Google Maps')
        )
      ),
      h('dl', { className: 'metrics' },
        metric('Available today', availableCount),
        metric('Query latency', `${latencyMs} ms`),
        metric('Cloud status', isFirebaseConfigured ? 'Firebase live' : 'Local demo')
      )
    ),
    h('section', { className: 'ops-grid' },
      card('Business model fit', 'FK Sava 45 serves dense residential demand in Block 45 with football-first hourly inventory. The system optimizes high-intent evening slots, walk-up mobile traffic, club-managed overrides, and ad-attributed bookings.'),
      card('Reserve with Google readiness', 'Slot IDs are deterministic, Firestore create semantics block duplicates, and payment metadata is tokenized for a compliant transfer layer before funds are split between venue and operating accounts.'),
      card('BI pipeline', "Every booking step writes GTM dataLayer events and Meta Pixel leads/purchases, ready to stream into BigQuery, Looker Studio, or a warehouse used by the venue's finance dashboard.")
    ),
    h('section', { className: 'booking-layout', id: 'booking' },
      h('div', null,
        h('div', { className: 'calendar-toolbar' },
          h('div', null, h('p', { className: 'eyebrow' }, 'Real-time calendar'), h('h2', null, 'Choose date and time')),
          h('input', { type: 'date', value: date, onChange: (event) => { setDate(event.target.value); setSelectedSlot(null); } })
        ),
        h(CourtCalendar, { slots, selectedSlot, onSelect: selectSlot })
      ),
      h(BookingForm, { selectedSlot, onBooked: () => setSelectedSlot(null) })
    )
  );
}

function metric(term, detail) {
  return h('div', null, h('dt', null, term), h('dd', null, detail));
}

function card(title, copy) {
  return h('article', null, h('h2', null, title), h('p', null, copy));
}

createRoot(document.getElementById('root')).render(h(App));
