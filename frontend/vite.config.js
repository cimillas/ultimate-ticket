import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }) => {
  const rootDir = fileURLToPath(new URL('.', import.meta.url));
  const env = loadEnv(mode, '..', '');
  const port = Number(env.FRONTEND_PORT || 5173);

  return {
    envDir: '..',
    server: {
      port,
    },
    build: {
      rollupOptions: {
        input: {
          main: resolve(rootDir, 'index.html'),
          admin: resolve(rootDir, 'admin/index.html'),
          login: resolve(rootDir, 'login/index.html'),
          register: resolve(rootDir, 'register/index.html'),
        },
      },
    },
  };
});
