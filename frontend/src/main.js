import { refreshAuthStatus, request, setupEventPicker, setupZonePicker, toggleLoginLink } from './common.js';

const output = document.getElementById('output');
const requestWithOutput = (path, options) => request(output, path, options);

const authStatus = document.getElementById('auth-status');
const adminLink = document.getElementById('admin-link');
const authOnlySections = document.querySelectorAll('[data-auth-only]');
const logoutButton = document.getElementById('logout-button');

function generateIdempotencyKey() {
  if (window.crypto?.randomUUID) {
    return window.crypto.randomUUID();
  }
  const bytes = new Uint8Array(16);
  if (window.crypto?.getRandomValues) {
    window.crypto.getRandomValues(bytes);
  } else {
    for (let i = 0; i < bytes.length; i += 1) {
      bytes[i] = Math.floor(Math.random() * 256);
    }
  }
  const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

function safeGetItem(key) {
  try {
    return window.localStorage.getItem(key);
  } catch {
    return null;
  }
}

function safeSetItem(key, value) {
  try {
    window.localStorage.setItem(key, value);
  } catch {
    return;
  }
}

function safeRemoveItem(key) {
  try {
    window.localStorage.removeItem(key);
  } catch {
    return;
  }
}

const listZonesForm = document.getElementById('list-zones');
const holdForm = document.getElementById('create-hold');
const holdKeyInput = holdForm.querySelector('input[name="idempotency_key"]');
const holdKeyStorageKey = 'ut_hold_idempotency_key';

function ensureHoldKey() {
  let key = holdKeyInput.value.trim();
  if (!key) {
    key = safeGetItem(holdKeyStorageKey) || generateIdempotencyKey();
    holdKeyInput.value = key;
  }
  if (key) {
    safeSetItem(holdKeyStorageKey, key);
  } else {
    safeRemoveItem(holdKeyStorageKey);
  }
  return key;
}

function rotateHoldKey() {
  const key = generateIdempotencyKey();
  holdKeyInput.value = key;
  safeSetItem(holdKeyStorageKey, key);
  return key;
}

const holdRegenButton = document.getElementById('regen-hold-key');
holdRegenButton.addEventListener('click', () => {
  rotateHoldKey();
});

holdKeyInput.addEventListener('input', () => {
  const value = holdKeyInput.value.trim();
  if (value) {
    safeSetItem(holdKeyStorageKey, value);
  } else {
    safeRemoveItem(holdKeyStorageKey);
  }
});

ensureHoldKey();

const confirmForm = document.getElementById('confirm-hold');
const confirmHoldIdInput = confirmForm.querySelector('input[name="hold_id"]');
const confirmKeyInput = confirmForm.querySelector('input[name="idempotency_key"]');
const confirmKeysStorageKey = 'ut_confirm_idempotency_keys';

function loadConfirmKeys() {
  const raw = safeGetItem(confirmKeysStorageKey);
  if (!raw) {
    return {};
  }
  try {
    return JSON.parse(raw);
  } catch {
    return {};
  }
}

function saveConfirmKeys(keys) {
  safeSetItem(confirmKeysStorageKey, JSON.stringify(keys));
}

function ensureConfirmKey() {
  const holdId = confirmHoldIdInput.value.trim();
  if (!holdId) {
    return;
  }
  if (confirmKeyInput.dataset.holdId === holdId && confirmKeyInput.value.trim()) {
    return;
  }
  const keys = loadConfirmKeys();
  let key = keys[holdId];
  if (!key) {
    key = generateIdempotencyKey();
    keys[holdId] = key;
    saveConfirmKeys(keys);
  }
  confirmKeyInput.value = key;
  confirmKeyInput.dataset.holdId = holdId;
}

function updateConfirmKeyStore() {
  const holdId = confirmHoldIdInput.value.trim();
  if (!holdId) {
    return;
  }
  const value = confirmKeyInput.value.trim();
  const keys = loadConfirmKeys();
  if (!value) {
    delete keys[holdId];
  } else {
    keys[holdId] = value;
  }
  saveConfirmKeys(keys);
  confirmKeyInput.dataset.holdId = holdId;
}

const confirmRegenButton = document.getElementById('regen-confirm-key');
confirmRegenButton.addEventListener('click', () => {
  const holdId = confirmHoldIdInput.value.trim();
  const key = generateIdempotencyKey();
  confirmKeyInput.value = key;
  confirmKeyInput.dataset.holdId = holdId;
  if (holdId) {
    const keys = loadConfirmKeys();
    keys[holdId] = key;
    saveConfirmKeys(keys);
  }
});

confirmHoldIdInput.addEventListener('change', () => {
  ensureConfirmKey();
});

confirmKeyInput.addEventListener('input', () => {
  updateConfirmKeyStore();
});

setupEventPicker(output, listZonesForm);
setupEventPicker(output, holdForm);
setupZonePicker(output, holdForm);

const updateAuthUI = (user) => {
  const isAdmin = user?.role === 'admin';
  if (adminLink) {
    adminLink.hidden = !isAdmin;
  }
  toggleLoginLink(user);
  authOnlySections.forEach((section) => {
    section.hidden = !user;
  });
};

authOnlySections.forEach((section) => {
  section.hidden = true;
});

refreshAuthStatus(authStatus).then(({ user }) => {
  updateAuthUI(user);
});

if (logoutButton) {
  logoutButton.addEventListener('click', async () => {
    const res = await requestWithOutput('/auth/logout', { method: 'POST' });
    if (res?.status >= 200 && res.status < 300) {
      const auth = await refreshAuthStatus(authStatus);
      updateAuthUI(auth.user);
    }
  });
}

document.getElementById('list-events').addEventListener('click', async () => {
  await requestWithOutput('/events');
});

listZonesForm.addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const eventID = form.event_id.value.trim();
  await requestWithOutput(`/events/${eventID}/zones`);
});

holdForm.addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const eventID = form.event_id.value.trim();
  const zoneID = form.zone_id.value.trim();
  const quantity = Number(form.quantity.value);
  const idempotencyKey = ensureHoldKey();

  const res = await requestWithOutput('/holds', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      event_id: eventID,
      zone_id: zoneID,
      quantity,
      idempotency_key: idempotencyKey,
    }),
  });
  if (res?.status >= 200 && res.status < 300) {
    rotateHoldKey();
  }
});

confirmForm.addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = event.currentTarget;
  const holdID = form.hold_id.value.trim();
  ensureConfirmKey();
  const idempotencyKey = form.idempotency_key.value.trim();

  await requestWithOutput(`/holds/${holdID}/confirm`, {
    method: 'POST',
    headers: { 'Idempotency-Key': idempotencyKey },
  });
});
