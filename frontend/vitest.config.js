import { loadEnv } from 'vite';
import { defineConfig } from 'vitest/config';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const apiBase = env.VITE_API_BASE_URL || 'http://localhost:8080';

  return {
    test: {
      environment: 'happy-dom',
    },
    define: {
      'import.meta.env.VITE_API_BASE_URL': JSON.stringify(apiBase),
    },
  };
});
