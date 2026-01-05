let apiBase = import.meta.env.VITE_API_BASE_URL;
if (!apiBase) {
  console.warn('VITE_API_BASE_URL not set, using http://localhost:8080');
  apiBase = 'http://localhost:8080';
}

export function setOutput(output, status, body) {
  if (!output) {
    return;
  }
  output.textContent = JSON.stringify({ status, body }, null, 2);
}

export async function fetchJSON(path, options = {}) {
  const res = await fetch(`${apiBase}${path}`, options);
  const text = await res.text();
  let body = text;
  try {
    body = text ? JSON.parse(text) : null;
  } catch {
    body = text;
  }
  return { status: res.status, body };
}

export async function fetchZones(output, eventID) {
  if (!eventID) {
    return null;
  }
  const res = await fetchJSON(`/admin/events/${eventID}/zones`);
  if (res.status >= 200 && res.status < 300 && Array.isArray(res.body)) {
    return res.body;
  }
  setOutput(output, res.status, res.body);
  return null;
}

export async function fetchEvents(output) {
  const res = await fetchJSON('/admin/events');
  if (res.status >= 200 && res.status < 300 && Array.isArray(res.body)) {
    return res.body.filter((event) => !event?.status || event.status === 'active');
  }
  setOutput(output, res.status, res.body);
  return null;
}

function zoneOptionLabel(zone) {
  if (!zone || !zone.name || zone.available === undefined || zone.capacity === undefined) {
    return zone?.id || 'Unknown zone';
  }
  return `${zone.name} (${zone.available}/${zone.capacity})`;
}

function eventOptionLabel(event) {
  if (!event || !event.name) {
    return event?.id || 'Unknown event';
  }
  if (event.id) {
    return `${event.name} (${event.id})`;
  }
  return event.name;
}

export function setupEventPicker(output, form) {
  const eventInput = form?.querySelector('input[name="event_id"]');
  const picker = form?.querySelector('[data-event-picker]');
  const dropdown = form?.querySelector('[data-event-dropdown]');
  if (!eventInput || !picker || !dropdown) {
    return;
  }

  let eventsCache = [];

  const closeDropdown = () => {
    dropdown.hidden = true;
  };

  const openDropdown = () => {
    if (dropdown.childElementCount === 0) {
      return;
    }
    dropdown.hidden = false;
  };

  const renderOptions = (events, filter = '') => {
    dropdown.innerHTML = '';
    const query = filter.trim().toLowerCase();

    const filtered = query
      ? events.filter((event) => {
          const name = event.name ? event.name.toLowerCase() : '';
          const id = event.id ? event.id.toLowerCase() : '';
          return name.includes(query) || id.includes(query);
        })
      : events;

    if (filtered.length === 0) {
      const empty = document.createElement('div');
      empty.className = 'event-empty';
      empty.textContent = 'No active events';
      dropdown.appendChild(empty);
      return;
    }

    for (const event of filtered) {
      const button = document.createElement('button');
      button.type = 'button';
      button.className = 'event-option';
      button.dataset.eventId = event.id;
      button.textContent = eventOptionLabel(event);
      dropdown.appendChild(button);
    }
  };

  const loadEvents = async (open = false) => {
    dropdown.innerHTML = '';
    eventsCache = [];
    const events = await fetchEvents(output);
    eventsCache = Array.isArray(events) ? events : [];
    renderOptions(eventsCache, eventInput.value);
    if (open) {
      openDropdown();
    }
  };

  eventInput.addEventListener('focus', () => {
    loadEvents(true);
  });

  eventInput.addEventListener('input', () => {
    if (eventsCache.length === 0) {
      loadEvents(true);
      return;
    }
    renderOptions(eventsCache, eventInput.value);
    openDropdown();
  });

  eventInput.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') {
      closeDropdown();
    }
  });

  dropdown.addEventListener('mousedown', (event) => {
    const option = event.target.closest('.event-option');
    if (!option) {
      return;
    }
    event.preventDefault();
    eventInput.value = option.dataset.eventId;
    eventInput.dispatchEvent(new Event('change', { bubbles: true }));
    closeDropdown();
  });

  document.addEventListener('mousedown', (event) => {
    if (!picker.contains(event.target)) {
      closeDropdown();
    }
  });
}

export function setupZonePicker(output, form) {
  const eventInput = form?.querySelector('input[name="event_id"]');
  const zoneInput = form?.querySelector('input[name="zone_id"]');
  const picker = form?.querySelector('[data-zone-picker]');
  const dropdown = form?.querySelector('[data-zone-dropdown]');
  if (!eventInput || !zoneInput || !picker || !dropdown) {
    return;
  }

  let loadedFor = '';
  let zonesCache = [];

  const closeDropdown = () => {
    dropdown.hidden = true;
  };

  const openDropdown = () => {
    if (dropdown.childElementCount === 0) {
      return;
    }
    dropdown.hidden = false;
  };

  const renderOptions = (zones, filter = '') => {
    dropdown.innerHTML = '';
    const query = filter.trim().toLowerCase();

    const filtered = query
      ? zones.filter((zone) => {
          const name = zone.name ? zone.name.toLowerCase() : '';
          const id = zone.id ? zone.id.toLowerCase() : '';
          return name.includes(query) || id.includes(query);
        })
      : zones;

    if (filtered.length === 0) {
      const empty = document.createElement('div');
      empty.className = 'zone-empty';
      empty.textContent = 'No zones found';
      dropdown.appendChild(empty);
      return;
    }

    for (const zone of filtered) {
      const button = document.createElement('button');
      button.type = 'button';
      button.className = 'zone-option';
      button.dataset.zoneId = zone.id;
      button.textContent = zoneOptionLabel(zone);
      dropdown.appendChild(button);
    }
  };

  const loadZones = async (open = false) => {
    const eventID = eventInput.value.trim();
    if (!eventID || eventID === loadedFor) {
      return;
    }
    loadedFor = eventID;
    zonesCache = [];
    dropdown.innerHTML = '';
    const zones = await fetchZones(output, eventID);
    zonesCache = Array.isArray(zones) ? zones : [];
    renderOptions(zonesCache, zoneInput.value);
    if (open) {
      openDropdown();
    }
  };

  const resetZones = () => {
    loadedFor = '';
    zonesCache = [];
    dropdown.innerHTML = '';
    closeDropdown();
  };

  eventInput.addEventListener('change', () => {
    zoneInput.value = '';
    resetZones();
    loadZones(false);
  });

  zoneInput.addEventListener('focus', () => {
    if (!loadedFor) {
      loadZones(true);
      return;
    }
    renderOptions(zonesCache, zoneInput.value);
    openDropdown();
  });

  zoneInput.addEventListener('input', () => {
    if (!loadedFor) {
      return;
    }
    renderOptions(zonesCache, zoneInput.value);
    openDropdown();
  });

  zoneInput.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') {
      closeDropdown();
    }
  });

  dropdown.addEventListener('mousedown', (event) => {
    const option = event.target.closest('.zone-option');
    if (!option) {
      return;
    }
    event.preventDefault();
    zoneInput.value = option.dataset.zoneId;
    closeDropdown();
  });

  document.addEventListener('mousedown', (event) => {
    if (!picker.contains(event.target)) {
      closeDropdown();
    }
  });
}

export async function request(output, path, options = {}) {
  const res = await fetchJSON(path, options);
  const { status, body } = res;
  setOutput(output, status, body);
  return { status, body };
}
