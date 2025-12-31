import { defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, '..', '');
  const port = Number(env.FRONTEND_PORT || 5173);

  return {
    envDir: '..',
    server: {
      port,
    },
  };
});
