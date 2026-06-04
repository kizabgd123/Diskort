import React from 'https://esm.sh/react@19.1.1';

const h = React.createElement;

export default function CourtCalendar({ slots, selectedSlot, onSelect }) {
  return h('section', { className: 'calendar', 'aria-label': 'Court availability calendar' },
    slots.map((slot) => h('button', {
      className: `slot ${slot.available ? 'available' : 'reserved'} ${selectedSlot?.id === slot.id ? 'selected' : ''}`,
      disabled: !slot.available,
      key: slot.id,
      onClick: () => onSelect(slot)
    },
      h('span', null, slot.startTime),
      h('strong', null, slot.courtName),
      h('small', null, slot.available ? `${slot.sport} · RSD ${slot.priceRsd.toLocaleString('sr-RS')}` : 'Reserved')
    ))
  );
}
