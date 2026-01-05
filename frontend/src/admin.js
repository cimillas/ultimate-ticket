import { request, setOutput, setupEventPicker, setupZonePicker } from './common.js';

const output = document.getElementById('output');
const requestWithOutput = (path, options) => request(output, path, options);

const cancelEventForm = document.getElementById('cancel-event');
const createZoneForm = document.getElementById('create-zone');
const listZonesForm = document.getElementById('list-zones');
const listActiveHoldsForm = document.getElementById('list-active-holds');
const listConfirmedOrdersForm = document.getElementById('list-confirmed-orders');

document.getElementById('create-event').addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const name = form.querySelector('input[name="name"]').value.trim();
  const startsAt = form.querySelector('input[name="starts_at"]').value.trim();

  const payload = { name };
  if (startsAt) {
    const parsed = new Date(startsAt);
    if (Number.isNaN(parsed.getTime())) {
      setOutput(output, 400, { error: 'invalid starts_at', code: 'invalid_starts_at' });
      return;
    }
    payload.starts_at = parsed.toISOString();
  }

  await requestWithOutput('/admin/events', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
});

createZoneForm.addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const eventID = form.querySelector('input[name="event_id"]').value.trim();
  const name = form.querySelector('input[name="name"]').value.trim();
  const capacity = Number(form.querySelector('input[name="capacity"]').value);

  await requestWithOutput(`/admin/events/${eventID}/zones`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, capacity }),
  });
});

document.getElementById('list-events').addEventListener('click', async () => {
  await requestWithOutput('/admin/events');
});

cancelEventForm.addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const eventID = form.querySelector('input[name="event_id"]').value.trim();

  await requestWithOutput(`/admin/events/${eventID}/cancel`, { method: 'POST' });
});

listZonesForm.addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const eventID = form.querySelector('input[name="event_id"]').value.trim();

  await requestWithOutput(`/admin/events/${eventID}/zones`);
});

listActiveHoldsForm.addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const eventID = form.querySelector('input[name="event_id"]').value.trim();
  const zoneID = form.querySelector('input[name="zone_id"]').value.trim();

  await requestWithOutput(`/admin/events/${eventID}/zones/${zoneID}/holds`);
});

listConfirmedOrdersForm.addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const eventID = form.querySelector('input[name="event_id"]').value.trim();
  const zoneID = form.querySelector('input[name="zone_id"]').value.trim();

  await requestWithOutput(`/admin/events/${eventID}/zones/${zoneID}/orders`);
});

setupEventPicker(output, cancelEventForm);
setupEventPicker(output, createZoneForm);
setupEventPicker(output, listZonesForm);
setupEventPicker(output, listActiveHoldsForm);
setupEventPicker(output, listConfirmedOrdersForm);
setupZonePicker(output, listActiveHoldsForm);
setupZonePicker(output, listConfirmedOrdersForm);
