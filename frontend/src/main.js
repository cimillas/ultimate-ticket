const output = document.getElementById('output');
let apiBase = import.meta.env.VITE_API_BASE_URL;
if (!apiBase) {
  console.warn('VITE_API_BASE_URL not set, using http://localhost:8080');
  apiBase = 'http://localhost:8080';
}

function setOutput(status, body) {
  output.textContent = JSON.stringify({ status, body }, null, 2);
}

async function request(path, options = {}) {
  const res = await fetch(`${apiBase}${path}`, options);
  const text = await res.text();
  let body = text;
  try {
    body = text ? JSON.parse(text) : null;
  } catch {
    body = text;
  }
  setOutput(res.status, body);
  return body;
}

document.getElementById('create-event').addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const name = form.name.value.trim();
  const startsAt = form.starts_at.value.trim();

  const payload = { name };
  if (startsAt) {
    payload.starts_at = startsAt;
  }

  await request('/admin/events', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
});

document.getElementById('list-events').addEventListener('click', async () => {
  await request('/admin/events');
});

document.getElementById('create-zone').addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const eventID = form.event_id.value.trim();
  const name = form.name.value.trim();
  const capacity = Number(form.capacity.value);

  await request(`/admin/events/${eventID}/zones`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, capacity }),
  });
});

document.getElementById('list-zones').addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const eventID = form.event_id.value.trim();
  await request(`/admin/events/${eventID}/zones`);
});

document.getElementById('create-hold').addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const eventID = form.event_id.value.trim();
  const zoneID = form.zone_id.value.trim();
  const quantity = Number(form.quantity.value);
  const idempotencyKey = form.idempotency_key.value.trim();

  await request('/holds', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      event_id: eventID,
      zone_id: zoneID,
      quantity,
      idempotency_key: idempotencyKey,
    }),
  });
});

document.getElementById('confirm-hold').addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const holdID = form.hold_id.value.trim();
  const idempotencyKey = form.idempotency_key.value.trim();

  await request(`/holds/${holdID}/confirm`, {
    method: 'POST',
    headers: { 'Idempotency-Key': idempotencyKey },
  });
});
