import { afterEach, describe, expect, it, vi } from 'vitest';

import {
  fetchEvents,
  fetchJSON,
  fetchZones,
  refreshAuthStatus,
  renderAuthStatus,
  request,
  setOutput,
  setupEventPicker,
  setupHoldPicker,
  setupZonePicker,
} from './common.js';

const flushPromises = () => new Promise((resolve) => setTimeout(resolve, 0));

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.innerHTML = '';
});

describe('setupZonePicker', () => {
  const zones = [
    { id: 'zone-1', name: 'Zone A', available: 4, capacity: 10 },
    { id: 'zone-2', name: 'Zone B', available: 2, capacity: 10 },
  ];

  function setupDOM() {
    document.body.innerHTML = `
      <form id="form">
        <input name="event_id" />
        <div class="zone-picker" data-zone-picker>
          <input name="zone_id" />
          <div class="zone-dropdown" data-zone-dropdown hidden></div>
        </div>
      </form>
      <pre id="output"></pre>
    `;
    const form = document.getElementById('form');
    const output = document.getElementById('output');
    setupZonePicker(output, form);
    return {
      form,
      output,
      eventInput: form.querySelector('input[name="event_id"]'),
      zoneInput: form.querySelector('input[name="zone_id"]'),
      dropdown: form.querySelector('[data-zone-dropdown]'),
    };
  }

  function mockFetch(body = zones, status = 200) {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        status,
        text: async () => (typeof body === 'string' ? body : JSON.stringify(body)),
      })),
    );
  }

  it('closes when clicking outside', async () => {
    mockFetch();
    const { eventInput, zoneInput, dropdown } = setupDOM();

    eventInput.value = 'event-1';
    eventInput.dispatchEvent(new Event('change'));
    await flushPromises();

    zoneInput.dispatchEvent(new Event('focus'));
    expect(dropdown.hidden).toBe(false);

    document.body.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
    expect(dropdown.hidden).toBe(true);
  });

  it('selects a zone and closes', async () => {
    mockFetch();
    const { eventInput, zoneInput, dropdown } = setupDOM();

    eventInput.value = 'event-1';
    eventInput.dispatchEvent(new Event('change'));
    await flushPromises();

    zoneInput.dispatchEvent(new Event('focus'));
    const firstOption = dropdown.querySelector('.zone-option');
    expect(firstOption).not.toBeNull();

    firstOption.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
    expect(zoneInput.value).toBe('zone-1');
    expect(dropdown.hidden).toBe(true);
  });

  it('clears zone_id when event changes', async () => {
    mockFetch();
    const { eventInput, zoneInput } = setupDOM();

    zoneInput.value = 'zone-1';
    eventInput.value = 'event-1';
    eventInput.dispatchEvent(new Event('change'));
    await flushPromises();

    expect(zoneInput.value).toBe('');
  });

  it('filters zones by name', async () => {
    mockFetch();
    const { eventInput, zoneInput, dropdown } = setupDOM();

    eventInput.value = 'event-1';
    eventInput.dispatchEvent(new Event('change'));
    await flushPromises();

    zoneInput.value = 'Zone B';
    zoneInput.dispatchEvent(new Event('input'));

    const options = dropdown.querySelectorAll('.zone-option');
    expect(options.length).toBe(1);
    expect(options[0].textContent).toContain('Zone B');
  });

  it('renders an empty state when no zones match', async () => {
    mockFetch();
    const { eventInput, zoneInput, dropdown } = setupDOM();

    eventInput.value = 'event-1';
    eventInput.dispatchEvent(new Event('change'));
    await flushPromises();

    zoneInput.value = 'missing';
    zoneInput.dispatchEvent(new Event('input'));

    expect(dropdown.querySelector('.zone-empty')?.textContent).toBe('No zones found');
  });
});

describe('setupEventPicker', () => {
  const events = [
    { id: 'event-1', name: 'Concert', status: 'active' },
    { id: 'event-2', name: 'Closed Event', status: 'closed' },
    { id: 'event-3', name: 'Cancelled', status: 'cancelled' },
  ];

  function setupDOM() {
    document.body.innerHTML = `
      <form id="form">
        <div class="event-picker" data-event-picker>
          <input name="event_id" />
          <div class="event-dropdown" data-event-dropdown hidden></div>
        </div>
      </form>
      <pre id="output"></pre>
    `;
    const form = document.getElementById('form');
    const output = document.getElementById('output');
    setupEventPicker(output, form);
    return {
      form,
      output,
      eventInput: form.querySelector('input[name="event_id"]'),
      dropdown: form.querySelector('[data-event-dropdown]'),
    };
  }

  function mockFetch(body = events, status = 200) {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        status,
        text: async () => (typeof body === 'string' ? body : JSON.stringify(body)),
      })),
    );
  }

  it('shows active events only', async () => {
    mockFetch();
    const { eventInput, dropdown } = setupDOM();

    eventInput.dispatchEvent(new Event('focus'));
    await flushPromises();

    const options = dropdown.querySelectorAll('.event-option');
    expect(options.length).toBe(1);
    expect(options[0].textContent).toContain('Concert');
  });

  it('selects an event and closes', async () => {
    mockFetch();
    const { eventInput, dropdown } = setupDOM();

    eventInput.dispatchEvent(new Event('focus'));
    await flushPromises();

    const option = dropdown.querySelector('.event-option');
    option.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));

    expect(eventInput.value).toBe('event-1');
    expect(dropdown.hidden).toBe(true);
  });

  it('filters events by name', async () => {
    mockFetch();
    const { eventInput, dropdown } = setupDOM();

    eventInput.dispatchEvent(new Event('focus'));
    await flushPromises();

    eventInput.value = 'conc';
    eventInput.dispatchEvent(new Event('input'));

    const options = dropdown.querySelectorAll('.event-option');
    expect(options.length).toBe(1);
    expect(options[0].textContent).toContain('Concert');
  });

  it('closes when clicking outside', async () => {
    mockFetch();
    const { eventInput, dropdown } = setupDOM();

    eventInput.dispatchEvent(new Event('focus'));
    await flushPromises();

    document.body.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
    expect(dropdown.hidden).toBe(true);
  });
});

describe('setupHoldPicker', () => {
  const holds = [{ id: 'hold-1' }, { id: 'hold-2' }];

  function setupDOM() {
    document.body.innerHTML = `
      <form id="form">
        <div class="hold-picker" data-hold-picker>
          <input name="hold_id" />
          <div class="hold-dropdown" data-hold-dropdown hidden></div>
        </div>
      </form>
      <pre id="output"></pre>
    `;
    const form = document.getElementById('form');
    const output = document.getElementById('output');
    setupHoldPicker(output, form);
    return {
      holdInput: form.querySelector('input[name="hold_id"]'),
      dropdown: form.querySelector('[data-hold-dropdown]'),
    };
  }

  function mockFetch(body = holds, status = 200) {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        status,
        text: async () => (typeof body === 'string' ? body : JSON.stringify(body)),
      })),
    );
  }

  it('selects a hold and closes', async () => {
    mockFetch();
    const { holdInput, dropdown } = setupDOM();

    holdInput.dispatchEvent(new Event('focus'));
    await flushPromises();

    const option = dropdown.querySelector('.hold-option');
    option.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));

    expect(holdInput.value).toBe('hold-1');
    expect(dropdown.hidden).toBe(true);
  });

  it('filters holds by id', async () => {
    mockFetch();
    const { holdInput, dropdown } = setupDOM();

    holdInput.dispatchEvent(new Event('focus'));
    await flushPromises();

    holdInput.value = 'hold-2';
    holdInput.dispatchEvent(new Event('input'));

    const options = dropdown.querySelectorAll('.hold-option');
    expect(options.length).toBe(1);
    expect(options[0].textContent).toBe('hold-2');
  });

  it('renders an empty state when no holds match', async () => {
    mockFetch();
    const { holdInput, dropdown } = setupDOM();

    holdInput.dispatchEvent(new Event('focus'));
    await flushPromises();

    holdInput.value = 'missing';
    holdInput.dispatchEvent(new Event('input'));

    expect(dropdown.querySelector('.hold-empty')?.textContent).toBe('No active holds');
  });

  it('closes when clicking outside', async () => {
    mockFetch();
    const { holdInput, dropdown } = setupDOM();

    holdInput.dispatchEvent(new Event('focus'));
    await flushPromises();

    document.body.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
    expect(dropdown.hidden).toBe(true);
  });
});

describe('fetchJSON', () => {
  it('parses JSON responses', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        status: 200,
        text: async () => JSON.stringify({ ok: true }),
      })),
    );

    const res = await fetchJSON('/ping');
    expect(res.status).toBe(200);
    expect(res.body).toEqual({ ok: true });
  });

  it('returns raw text for non-JSON', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        status: 500,
        text: async () => 'boom',
      })),
    );

    const res = await fetchJSON('/oops');
    expect(res.status).toBe(500);
    expect(res.body).toBe('boom');
  });

  it('returns null for empty responses', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        status: 204,
        text: async () => '',
      })),
    );

    const res = await fetchJSON('/empty');
    expect(res.status).toBe(204);
    expect(res.body).toBeNull();
  });
});

describe('fetchZones', () => {
  it('sets output when the response is not OK', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        status: 404,
        text: async () => JSON.stringify({ error: 'missing' }),
      })),
    );

    const output = document.createElement('pre');
    const res = await fetchZones(output, 'event-404');
    expect(res).toBeNull();
    const parsed = JSON.parse(output.textContent);
    expect(parsed.status).toBe(404);
    expect(parsed.body).toEqual({ error: 'missing' });
  });
});

describe('auth status', () => {
  it('renders signed-in and signed-out states', () => {
    const container = document.createElement('div');
    container.className = 'auth-status';
    const el = document.createElement('strong');
    container.appendChild(el);
    renderAuthStatus(el, { username: 'ana', role: 'admin' });
    expect(el.textContent).toBe('ana (admin)');
    expect(container.hidden).toBe(false);

    renderAuthStatus(el, null);
    expect(el.textContent).toBe('');
    expect(container.hidden).toBe(true);
  });

  it('refreshes auth status from /me', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        status: 200,
        text: async () => JSON.stringify({ user: { username: 'ana', role: 'user' } }),
      })),
    );

    const el = document.createElement('strong');
    await refreshAuthStatus(el);
    expect(el.textContent).toBe('ana (user)');
  });
});

describe('fetchEvents', () => {
  it('filters to active events', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        status: 200,
        text: async () =>
          JSON.stringify([
            { id: 'event-1', status: 'active' },
            { id: 'event-2', status: 'closed' },
          ]),
      })),
    );

    const output = document.createElement('pre');
    const res = await fetchEvents(output);
    expect(res?.length).toBe(1);
    expect(res?.[0].id).toBe('event-1');
  });
});

describe('request', () => {
  it('reports network errors in the output', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => {
      throw new Error('network down');
    }));

    const output = document.createElement('pre');
    const res = await request(output, '/events');
    expect(res.status).toBe(0);
    const parsed = JSON.parse(output.textContent);
    expect(parsed.body.error).toBe('network_error');
  });
});
