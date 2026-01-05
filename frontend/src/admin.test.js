import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const flushPromises = () => new Promise((resolve) => setTimeout(resolve, 0));

const setupDOM = () => {
  document.body.innerHTML = `
    <button id="list-events">List events</button>
    <form id="cancel-event">
      <div data-event-picker>
        <input name="event_id" />
        <div data-event-dropdown hidden></div>
      </div>
      <button type="submit">Cancel event</button>
    </form>
    <form id="create-event">
      <input name="name" />
      <input name="starts_at" />
      <button type="submit">Create event</button>
    </form>
    <form id="create-zone">
      <div data-event-picker>
        <input name="event_id" />
        <div data-event-dropdown hidden></div>
      </div>
      <input name="name" />
      <input name="capacity" />
      <button type="submit">Create zone</button>
    </form>
    <form id="list-zones">
      <div data-event-picker>
        <input name="event_id" />
        <div data-event-dropdown hidden></div>
      </div>
      <button type="submit">List zones</button>
    </form>
    <form id="list-active-holds">
      <div data-event-picker>
        <input name="event_id" />
        <div data-event-dropdown hidden></div>
      </div>
      <div data-zone-picker>
        <input name="zone_id" />
        <div data-zone-dropdown hidden></div>
      </div>
      <button type="submit">List holds</button>
    </form>
    <form id="list-confirmed-orders">
      <div data-event-picker>
        <input name="event_id" />
        <div data-event-dropdown hidden></div>
      </div>
      <div data-zone-picker>
        <input name="zone_id" />
        <div data-zone-dropdown hidden></div>
      </div>
      <button type="submit">List orders</button>
    </form>
    <pre id="output"></pre>
  `;
};

const loadAdmin = async () => {
  await import('./admin.js');
};

beforeEach(() => {
  vi.resetModules();
  setupDOM();
});

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.innerHTML = '';
});

describe('admin.js', () => {
  it('validates starts_at before sending a request', async () => {
    const fetchSpy = vi.fn(async () => ({
      status: 200,
      text: async () => '',
    }));
    vi.stubGlobal('fetch', fetchSpy);

    await loadAdmin();

    const form = document.getElementById('create-event');
    form.querySelector('input[name="name"]').value = 'Concert';
    form.querySelector('input[name="starts_at"]').value = 'invalid-date';

    form.dispatchEvent(new Event('submit'));
    await flushPromises();

    expect(fetchSpy).not.toHaveBeenCalled();
    const output = document.getElementById('output');
    const parsed = JSON.parse(output.textContent);
    expect(parsed.body).toEqual({ error: 'invalid starts_at', code: 'invalid_starts_at' });
  });

  it('posts events with a valid starts_at payload', async () => {
    const fetchSpy = vi.fn(async () => ({
      status: 201,
      text: async () => JSON.stringify({ ok: true }),
    }));
    vi.stubGlobal('fetch', fetchSpy);

    await loadAdmin();

    const form = document.getElementById('create-event');
    form.querySelector('input[name="name"]').value = 'Concert';
    form.querySelector('input[name="starts_at"]').value = '2025-02-01T10:00:00Z';

    form.dispatchEvent(new Event('submit'));
    await flushPromises();

    const [url, options] = fetchSpy.mock.calls[0];
    expect(url).toBe('http://localhost:8080/admin/events');
    const payload = JSON.parse(options.body);
    expect(payload.name).toBe('Concert');
    expect(payload.starts_at).toBe('2025-02-01T10:00:00.000Z');
  });

  it('posts zones with the expected payload', async () => {
    const fetchSpy = vi.fn(async () => ({
      status: 201,
      text: async () => JSON.stringify({ ok: true }),
    }));
    vi.stubGlobal('fetch', fetchSpy);

    await loadAdmin();

    const form = document.getElementById('create-zone');
    form.querySelector('input[name="event_id"]').value = 'event-1';
    form.querySelector('input[name="name"]').value = 'Zone A';
    form.querySelector('input[name="capacity"]').value = '10';

    form.dispatchEvent(new Event('submit'));
    await flushPromises();

    const [url, options] = fetchSpy.mock.calls[0];
    expect(url).toBe('http://localhost:8080/admin/events/event-1/zones');
    const payload = JSON.parse(options.body);
    expect(payload).toEqual({ name: 'Zone A', capacity: 10 });
  });

  it('requests list endpoints with the correct paths', async () => {
    const fetchSpy = vi.fn(async () => ({
      status: 200,
      text: async () => JSON.stringify([]),
    }));
    vi.stubGlobal('fetch', fetchSpy);

    await loadAdmin();

    document.getElementById('list-events').click();
    await flushPromises();
    expect(fetchSpy.mock.calls[0][0]).toBe('http://localhost:8080/admin/events');

    const listZones = document.getElementById('list-zones');
    listZones.querySelector('input[name="event_id"]').value = 'event-2';
    listZones.dispatchEvent(new Event('submit'));
    await flushPromises();
    expect(fetchSpy.mock.calls[1][0]).toBe('http://localhost:8080/admin/events/event-2/zones');

    const listHolds = document.getElementById('list-active-holds');
    listHolds.querySelector('input[name="event_id"]').value = 'event-3';
    listHolds.querySelector('input[name="zone_id"]').value = 'zone-3';
    listHolds.dispatchEvent(new Event('submit'));
    await flushPromises();
    expect(fetchSpy.mock.calls[2][0]).toBe('http://localhost:8080/admin/events/event-3/zones/zone-3/holds');

    const listOrders = document.getElementById('list-confirmed-orders');
    listOrders.querySelector('input[name="event_id"]').value = 'event-4';
    listOrders.querySelector('input[name="zone_id"]').value = 'zone-4';
    listOrders.dispatchEvent(new Event('submit'));
    await flushPromises();
    expect(fetchSpy.mock.calls[3][0]).toBe('http://localhost:8080/admin/events/event-4/zones/zone-4/orders');
  });

  it('posts event cancellation', async () => {
    const fetchSpy = vi.fn(async () => ({
      status: 200,
      text: async () => JSON.stringify({ ok: true }),
    }));
    vi.stubGlobal('fetch', fetchSpy);

    await loadAdmin();

    const form = document.getElementById('cancel-event');
    form.querySelector('input[name="event_id"]').value = 'event-9';
    form.dispatchEvent(new Event('submit'));
    await flushPromises();

    const [url, options] = fetchSpy.mock.calls[0];
    expect(url).toBe('http://localhost:8080/admin/events/event-9/cancel');
    expect(options.method).toBe('POST');
  });
});
