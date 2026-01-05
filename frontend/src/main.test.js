import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const flushPromises = () => new Promise((resolve) => setTimeout(resolve, 0));

const setupDOM = () => {
  document.body.innerHTML = `
    <button id="list-events">List events</button>
    <form id="list-zones">
      <div data-event-picker>
        <input name="event_id" />
        <div data-event-dropdown hidden></div>
      </div>
      <button type="submit">List zones</button>
    </form>
    <form id="create-hold">
      <div data-event-picker>
        <input name="event_id" />
        <div data-event-dropdown hidden></div>
      </div>
      <div data-zone-picker>
        <input name="zone_id" />
        <div data-zone-dropdown hidden></div>
      </div>
      <input name="quantity" />
      <input name="idempotency_key" />
      <button id="regen-hold-key" type="button">Regenerate</button>
      <button type="submit">Create</button>
    </form>
    <form id="confirm-hold">
      <input name="hold_id" />
      <input name="idempotency_key" />
      <button id="regen-confirm-key" type="button">Regenerate</button>
      <button type="submit">Confirm</button>
    </form>
    <pre id="output"></pre>
  `;
};

const stubCrypto = () => {
  let seq = 0;
  vi.stubGlobal('crypto', {
    randomUUID: vi.fn(() => `uuid-${(seq += 1)}`),
    getRandomValues: (arr) => arr,
  });
};

const loadMain = async () => {
  await import('./main.js');
};

beforeEach(() => {
  vi.resetModules();
  localStorage.clear();
  setupDOM();
});

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.innerHTML = '';
});

describe('main.js', () => {
  it('auto-generates a hold idempotency key and stores it', async () => {
    stubCrypto();
    await loadMain();

    const holdKeyInput = document.querySelector('#create-hold input[name="idempotency_key"]');
    expect(holdKeyInput.value).toBe('uuid-1');
    expect(localStorage.getItem('ut_hold_idempotency_key')).toBe('uuid-1');
  });

  it('regenerates the hold idempotency key on click', async () => {
    stubCrypto();
    await loadMain();

    const holdKeyInput = document.querySelector('#create-hold input[name="idempotency_key"]');
    const regenButton = document.getElementById('regen-hold-key');

    regenButton.click();
    expect(holdKeyInput.value).toBe('uuid-2');
    expect(localStorage.getItem('ut_hold_idempotency_key')).toBe('uuid-2');
  });

  it('generates and stores confirm keys per hold id', async () => {
    stubCrypto();
    await loadMain();

    const holdIdInput = document.querySelector('#confirm-hold input[name="hold_id"]');
    const confirmKeyInput = document.querySelector('#confirm-hold input[name="idempotency_key"]');

    holdIdInput.value = 'hold-1';
    holdIdInput.dispatchEvent(new Event('change'));

    expect(confirmKeyInput.value).toBe('uuid-2');
    const stored = JSON.parse(localStorage.getItem('ut_confirm_idempotency_keys'));
    expect(stored['hold-1']).toBe('uuid-2');
  });

  it('posts a hold with the expected payload and rotates the key on success', async () => {
    stubCrypto();
    const fetchSpy = vi.fn(async () => ({
      status: 201,
      text: async () => JSON.stringify({ ok: true }),
    }));
    vi.stubGlobal('fetch', fetchSpy);

    await loadMain();

    const holdForm = document.getElementById('create-hold');
    holdForm.event_id.value = 'event-1';
    holdForm.zone_id.value = 'zone-1';
    holdForm.quantity.value = '2';

    holdForm.dispatchEvent(new Event('submit'));
    await flushPromises();

    const [url, options] = fetchSpy.mock.calls[0];
    expect(url).toBe('http://localhost:8080/holds');
    expect(options.method).toBe('POST');
    const payload = JSON.parse(options.body);
    expect(payload).toEqual({
      event_id: 'event-1',
      zone_id: 'zone-1',
      quantity: 2,
      idempotency_key: 'uuid-1',
    });

    const holdKeyInput = document.querySelector('#create-hold input[name="idempotency_key"]');
    expect(holdKeyInput.value).toBe('uuid-2');
  });

  it('posts a confirmation with the idempotency header', async () => {
    stubCrypto();
    const fetchSpy = vi.fn(async () => ({
      status: 200,
      text: async () => '',
    }));
    vi.stubGlobal('fetch', fetchSpy);

    await loadMain();

    const confirmForm = document.getElementById('confirm-hold');
    confirmForm.hold_id.value = 'hold-9';
    confirmForm.idempotency_key.value = 'confirm-key';
    confirmForm.querySelector('input[name="idempotency_key"]').dataset.holdId = 'hold-9';

    confirmForm.dispatchEvent(new Event('submit'));
    await flushPromises();

    const [url, options] = fetchSpy.mock.calls[0];
    expect(url).toBe('http://localhost:8080/holds/hold-9/confirm');
    expect(options.method).toBe('POST');
    expect(options.headers['Idempotency-Key']).toBe('confirm-key');
  });
});
