import { refreshAuthStatus, renderAuthStatus, request, toggleLoginLink } from './common.js';

const output = document.getElementById('output');
const authStatus = document.getElementById('auth-status');
const loginForm = document.getElementById('login-form');
const registerForm = document.getElementById('register-form');
const logoutButton = document.getElementById('logout-button');
const adminLink = document.getElementById('admin-link');
const registerLink = document.getElementById('register-link');
const allowRegisterKey = 'ut_allow_register';
const isRegisterPage = Boolean(registerForm && !loginForm);
let allowRegister = true;

const requestWithOutput = (path, options) => request(output, path, options);

const updateAdminLink = (user) => {
  if (!adminLink) {
    return;
  }
  adminLink.hidden = user?.role !== 'admin';
};
const updateNavLinks = (user) => {
  updateAdminLink(user);
  toggleLoginLink(user);
};

if (registerLink) {
  registerLink.addEventListener('click', () => {
    try {
      window.sessionStorage.setItem(allowRegisterKey, '1');
    } catch {
      return;
    }
  });
}

if (isRegisterPage) {
  let allowed = false;
  try {
    allowed = window.sessionStorage.getItem(allowRegisterKey) === '1';
  } catch {
    allowed = false;
  }
  if (!allowed) {
    window.location.href = '/login/';
    allowRegister = false;
  } else {
    try {
      window.sessionStorage.removeItem(allowRegisterKey);
    } catch {
      allowRegister = true;
    }
  }
}

if (loginForm) {
  loginForm.addEventListener('submit', async (event) => {
    event.preventDefault();
    const form = event.currentTarget;
    const identifier = form.identifier.value.trim();
    const password = form.password.value;

    const res = await requestWithOutput('/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ identifier, password }),
    });
    if (res?.status >= 200 && res.status < 300) {
      const user = res?.body?.user;
      if (user) {
        renderAuthStatus(authStatus, user);
        updateNavLinks(user);
      } else {
        const auth = await refreshAuthStatus(authStatus);
        updateNavLinks(auth.user);
      }
    }
  });
}

if (registerForm && allowRegister) {
  registerForm.addEventListener('submit', async (event) => {
    event.preventDefault();
    const form = event.currentTarget;
    const username = form.username.value.trim();
    const email = form.email.value.trim();
    const password = form.password.value;

    const res = await requestWithOutput('/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, email, password }),
    });
    if (res?.status >= 200 && res.status < 300) {
      const user = res?.body?.user;
      if (user) {
        renderAuthStatus(authStatus, user);
        updateNavLinks(user);
      } else {
        const auth = await refreshAuthStatus(authStatus);
        updateNavLinks(auth.user);
      }
    }
  });
}

if (logoutButton) {
  logoutButton.addEventListener('click', async () => {
    const res = await requestWithOutput('/auth/logout', { method: 'POST' });
    if (res?.status >= 200 && res.status < 300) {
      const auth = await refreshAuthStatus(authStatus);
      updateNavLinks(auth.user);
    }
  });
}

refreshAuthStatus(authStatus).then(({ user }) => {
  updateNavLinks(user);
});
