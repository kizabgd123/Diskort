import React, { useState } from 'https://esm.sh/react@19.1.1';
import { googleCalendarUrl, reserveSlot, startPayment } from '../services/reservations.js';
import { trackBookingStep, trackPurchase } from '../services/analytics.js';

const h = React.createElement;

export default function BookingForm({ selectedSlot, onBooked }) {
  const [customer, setCustomer] = useState({ name: '', phone: '', email: '' });
  const [paymentProvider, setPaymentProvider] = useState('stripe');
  const [status, setStatus] = useState('idle');
  const [error, setError] = useState('');
  const [reservation, setReservation] = useState(null);

  if (!selectedSlot) {
    return h('aside', { className: 'booking-card muted' }, 'Choose an available slot to start booking online.');
  }

  async function submit(event) {
    event.preventDefault();
    setStatus('saving');
    setError('');
    try {
      trackBookingStep('form_submit', { slotId: selectedSlot.id, paymentProvider });
      const saved = await reserveSlot(selectedSlot, customer, paymentProvider);
      const payment = await startPayment(saved);
      setReservation({ ...saved, payment });
      setStatus('reserved');
      trackPurchase(saved);
      onBooked(saved);
    } catch (err) {
      setError(err.message);
      setStatus('error');
    }
  }

  return h('aside', { className: 'booking-card' },
    h('p', { className: 'eyebrow' }, 'Book Online'),
    h('h2', null, selectedSlot.courtName),
    h('p', null, `${selectedSlot.date} · ${selectedSlot.startTime}-${selectedSlot.endTime} · RSD ${selectedSlot.priceRsd.toLocaleString('sr-RS')}`),
    h('form', { onSubmit: submit },
      field('Name', h('input', { required: true, value: customer.name, onChange: (event) => setCustomer({ ...customer, name: event.target.value }) })),
      field('Phone', h('input', { required: true, value: customer.phone, onChange: (event) => setCustomer({ ...customer, phone: event.target.value }) })),
      field('Email', h('input', { type: 'email', required: true, value: customer.email, onChange: (event) => setCustomer({ ...customer, email: event.target.value }) })),
      field('Payment processor', h('select', { value: paymentProvider, onChange: (event) => setPaymentProvider(event.target.value) },
        h('option', { value: 'stripe' }, 'Stripe Billing'),
        h('option', { value: 'square' }, 'Square Checkout')
      )),
      h('button', { disabled: status === 'saving', type: 'submit' }, status === 'saving' ? 'Reserving…' : 'Reserve with Google-ready payment')
    ),
    error ? h('p', { className: 'error' }, error) : null,
    reservation ? h('div', { className: 'confirmation' },
      h('strong', null, 'Reservation held for payment.'),
      h('a', { href: googleCalendarUrl(reservation), target: '_blank', rel: 'noreferrer' }, 'Add to Google Calendar'),
      h('a', { href: reservation.payment.redirectUrl }, `Continue to ${reservation.payment.provider}`)
    ) : null
  );
}

function field(label, input) {
  return h('label', null, label, input);
}
