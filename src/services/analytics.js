export function pushDataLayer(event, payload = {}) {
  window.dataLayer = window.dataLayer || [];
  window.dataLayer.push({ event, ...payload, timestamp: new Date().toISOString() });
}

export function trackMeta(event, payload = {}) {
  if (typeof window.fbq === 'function') {
    window.fbq('track', event, payload);
  }
}

export function trackBookingStep(step, payload = {}) {
  pushDataLayer('booking_step', { step, ...payload });
  trackMeta('Lead', { content_name: step, ...payload });
}

export function trackPurchase(reservation) {
  const payload = {
    currency: 'RSD',
    value: reservation.priceRsd,
    content_ids: [reservation.slotId],
    content_name: `${reservation.courtName} ${reservation.date} ${reservation.startTime}`
  };
  pushDataLayer('purchase_intent', payload);
  trackMeta('Purchase', payload);
}
