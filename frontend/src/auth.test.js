import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const flushPromises = () => new Promise((resolve) => setTimeout(resolve, 0));

const setupLoginDOM = () => {
  document.body.innerHTML = `
    <div class="auth-status"><strong id="auth-status"></strong></div>
    <button id="logout-button">Logout</button>
    <form id="login-form">
      <input name="identifier" />
      <input name="password" />
      <button type="submit">Login</button>
    </form>
    <a href="/admin/" id="admin-link">Admin</a>
    <a id="register-link" href="/register/">Create an account</a>
    <pre id="output"></pre>
  `;
};

const setupRegisterDOM = () => {
  document.body.innerHTML = `
    <div class="auth-status"><strong id="auth-status"></strong></div>
    <button id="logout-button">Logout</button>
    <form id="register-form">
      <input name="username" />
      <input name="email" />
      <input name="password" />
      <button type="submit">Register</button>
    </form>
    <a href="/admin/" id="admin-link">Admin</a>
    <pre id="output"></pre>
  `;
};

const loadAuth = async () => {
  await import('./auth.js');
};

beforeEach(() => {
  vi.resetModules();
  sessionStorage.clear();
});

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.innerHTML = '';
});

describe('auth.js', () => {
  it('posts login with identifier and password', async () => {
    setupLoginDOM();
    const fetchSpy = vi.fn(async (url) => {
      if (url.endsWith('/me')) {
        return { status: 401, text: async () => JSON.stringify({ error: 'unauthorized' }) };
      }
      return { status: 200, text: async () => JSON.stringify({ user: { username: 'ana' } }) };
    });
    vi.stubGlobal('fetch', fetchSpy);

    await loadAuth();

    const form = document.getElementById('login-form');
    form.identifier.value = 'ana';
    form.password.value = 'secret';

    form.dispatchEvent(new Event('submit'));
    await flushPromises();

    const loginCall = fetchSpy.mock.calls.find((call) => call[0].endsWith('/auth/login'));
    const [url, options] = loginCall;
    expect(url).toBe('http://localhost:8080/auth/login');
    expect(options.method).toBe('POST');
    expect(JSON.parse(options.body)).toEqual({ identifier: 'ana', password: 'secret' });
  });

  it('posts registration with the expected payload', async () => {
    setupRegisterDOM();
    sessionStorage.setItem('ut_allow_register', '1');
    const fetchSpy = vi.fn(async (url) => {
      if (url.endsWith('/me')) {
        return { status: 401, text: async () => JSON.stringify({ error: 'unauthorized' }) };
      }
      return { status: 200, text: async () => JSON.stringify({ user: { username: 'new-user' } }) };
    });
    vi.stubGlobal('fetch', fetchSpy);

    await loadAuth();

    const form = document.getElementById('register-form');
    form.username.value = 'new-user';
    form.email.value = 'new@example.com';
    form.password.value = 'secret';

    form.dispatchEvent(new Event('submit'));
    await flushPromises();

    const registerCall = fetchSpy.mock.calls.find((call) => call[0].endsWith('/auth/register'));
    const [url, options] = registerCall;
    expect(url).toBe('http://localhost:8080/auth/register');
    expect(options.method).toBe('POST');
    expect(JSON.parse(options.body)).toEqual({
      username: 'new-user',
      email: 'new@example.com',
      password: 'secret',
    });
  });

  it('posts logout', async () => {
    setupRegisterDOM();
    sessionStorage.setItem('ut_allow_register', '1');
    const fetchSpy = vi.fn(async (url) => {
      if (url.endsWith('/me')) {
        return { status: 200, text: async () => JSON.stringify({ user: { username: 'ana' } }) };
      }
      return { status: 200, text: async () => JSON.stringify({ ok: true }) };
    });
    vi.stubGlobal('fetch', fetchSpy);

    await loadAuth();

    document.getElementById('logout-button').click();
    await flushPromises();

    const logoutCall = fetchSpy.mock.calls.find((call) => call[0].endsWith('/auth/logout'));
    const [url, options] = logoutCall;
    expect(url).toBe('http://localhost:8080/auth/logout');
    expect(options.method).toBe('POST');
  });

  it('redirects register when not allowed', async () => {
    setupRegisterDOM();
    const originalLocation = window.location;
    Object.defineProperty(window, 'location', {
      value: { href: '/register/' },
      writable: true,
    });

    const fetchSpy = vi.fn(async () => ({
      status: 401,
      text: async () => JSON.stringify({ error: 'unauthorized' }),
    }));
    vi.stubGlobal('fetch', fetchSpy);

    await loadAuth();

    expect(window.location.href).toBe('/login/');

    Object.defineProperty(window, 'location', { value: originalLocation });
  });
});
