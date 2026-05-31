const DEFAULT_COURTS = [
  { id: 'pitch-5a', name: 'Mini pitch A', sport: 'Football 5v5', priceRsd: 4800 },
  { id: 'pitch-5b', name: 'Mini pitch B', sport: 'Football 5v5', priceRsd: 4800 },
  { id: 'pitch-7', name: 'Main pitch', sport: 'Football 7v7', priceRsd: 7200 }
];

const env = (key) => window.FK_SAVA45_CONFIG?.[key] || '';
const memoryReservations = new Map();

export const courts = DEFAULT_COURTS;
export const reservationCollection = 'fkSava45Reservations';
export const isFirebaseConfigured = Boolean(env('FIREBASE_PROJECT_ID') && env('FIREBASE_API_KEY'));

export function slotId({ courtId, date, startTime }) {
  return `${courtId}_${date}_${startTime}`.replace(/[^a-zA-Z0-9_-]/g, '-');
}

export function addMinutes(time, minutes) {
  const [hours, mins] = time.split(':').map(Number);
  const date = new Date(Date.UTC(2026, 0, 1, hours, mins + minutes));
  return date.toISOString().slice(11, 16);
}

export function buildSlots(date, startHour = 8, endHour = 23) {
  return courts.flatMap((court) => {
    const slots = [];
    for (let hour = startHour; hour < endHour; hour += 1) {
      slots.push({
        id: slotId({ courtId: court.id, date, startTime: `${String(hour).padStart(2, '0')}:00` }),
        courtId: court.id,
        courtName: court.name,
        sport: court.sport,
        priceRsd: court.priceRsd,
        date,
        startTime: `${String(hour).padStart(2, '0')}:00`,
        endTime: `${String(hour + 1).padStart(2, '0')}:00`
      });
    }
    return slots;
  });
}

export function subscribeToAvailability(date, callback) {
  let cancelled = false;
  const baseline = buildSlots(date);

  async function refresh() {
    const reservations = isFirebaseConfigured ? await fetchReservations(date) : Array.from(memoryReservations.values());
    if (!cancelled) callback(mergeAvailability(baseline, reservations));
  }

  refresh();
  const interval = window.setInterval(refresh, 5000);
  return () => {
    cancelled = true;
    window.clearInterval(interval);
  };
}

async function fetchReservations(date) {
  const projectId = env('FIREBASE_PROJECT_ID');
  const apiKey = env('FIREBASE_API_KEY');
  const url = `https://firestore.googleapis.com/v1/projects/${projectId}/databases/(default)/documents:runQuery?key=${apiKey}`;
  const body = {
    structuredQuery: {
      from: [{ collectionId: reservationCollection }],
      where: { fieldFilter: { field: { fieldPath: 'date' }, op: 'EQUAL', value: { stringValue: date } } }
    }
  };
  const response = await fetch(url, { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify(body) });
  if (!response.ok) return [];
  const rows = await response.json();
  return rows.filter((row) => row.document).map((row) => decodeFirestore(row.document.fields));
}

function decodeFirestore(fields) {
  const decoded = {};
  Object.entries(fields || {}).forEach(([key, value]) => {
    if ('stringValue' in value) decoded[key] = value.stringValue;
    else if ('integerValue' in value) decoded[key] = Number(value.integerValue);
    else if ('booleanValue' in value) decoded[key] = value.booleanValue;
    else if ('timestampValue' in value) decoded[key] = value.timestampValue;
    else decoded[key] = null;
  });
  return decoded;
}

function encodeFirestore(data) {
  const fields = {};
  Object.entries(data).forEach(([key, value]) => {
    if (typeof value === 'number') fields[key] = { integerValue: value };
    else if (typeof value === 'boolean') fields[key] = { booleanValue: value };
    else if (value && typeof value === 'object') fields[key] = { stringValue: JSON.stringify(value) };
    else fields[key] = { stringValue: String(value ?? '') };
  });
  return { fields };
}

export function mergeAvailability(slots, reservations) {
  const reserved = new Map(reservations.filter((item) => item.status !== 'cancelled').map((item) => [item.slotId, item]));
  return slots.map((slot) => ({ ...slot, reservation: reserved.get(slot.id) || null, available: !reserved.has(slot.id) }));
}

export async function reserveSlot(slot, customer, paymentProvider = 'stripe') {
  const reservation = {
    slotId: slot.id,
    courtId: slot.courtId,
    courtName: slot.courtName,
    date: slot.date,
    startTime: slot.startTime,
    endTime: slot.endTime,
    priceRsd: slot.priceRsd,
    customer,
    paymentProvider,
    status: 'pending_payment',
    calendarSyncStatus: 'queued',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString()
  };

  if (!isFirebaseConfigured) {
    if (memoryReservations.has(slot.id)) {
      throw new Error('This slot has just been reserved. Please choose another time.');
    }
    memoryReservations.set(slot.id, reservation);
    return reservation;
  }

  const projectId = env('FIREBASE_PROJECT_ID');
  const apiKey = env('FIREBASE_API_KEY');
  const url = `https://firestore.googleapis.com/v1/projects/${projectId}/databases/(default)/documents/${reservationCollection}?documentId=${slot.id}&key=${apiKey}`;
  const response = await fetch(url, { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify(encodeFirestore(reservation)) });
  if (!response.ok) {
    throw new Error('This slot has just been reserved. Please choose another time.');
  }
  return reservation;
}

export async function startPayment(reservation) {
  const provider = reservation.paymentProvider || 'stripe';
  if (provider === 'square') {
    return { provider, redirectUrl: env('SQUARE_CHECKOUT_URL') || '#square-checkout' };
  }
  return { provider: 'stripe', publishableKey: env('STRIPE_PUBLISHABLE_KEY'), redirectUrl: '#create-checkout-session-in-cloud-function' };
}

export function googleCalendarUrl(reservation) {
  const start = `${reservation.date.replaceAll('-', '')}T${reservation.startTime.replace(':', '')}00`;
  const end = `${reservation.date.replaceAll('-', '')}T${reservation.endTime.replace(':', '')}00`;
  const params = new URLSearchParams({
    action: 'TEMPLATE',
    text: `FK Sava 45 - ${reservation.courtName}`,
    details: 'Online reservation for FK Sava 45. Payment status and court allocation are synchronized from Firebase.',
    location: 'FK Sava 45, Blok 45, Novi Beograd, Serbia',
    dates: `${start}/${end}`
  });
  return `https://calendar.google.com/calendar/render?${params.toString()}`;
}

export function mapsDirectionsUrl() {
  const params = new URLSearchParams({
    api: '1',
    destination: 'FK Sava 45, Vojvođanska bb, Blok 45, Novi Beograd, Serbia'
  });
  return `https://www.google.com/maps/dir/?${params.toString()}`;
}
