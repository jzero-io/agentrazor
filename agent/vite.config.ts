import { defineConfig, loadEnv } from 'vite';
import vue from '@vitejs/plugin-vue';

export default defineConfig(({ mode, command }) => {
  const env = loadEnv(mode, process.cwd(), '');

  return {
    // Assets stay relative so the same build works both at / and behind the
    // Admin reverse proxy. Vite's dev client needs an absolute mount path.
    base: command === 'serve' ? '/agent-app/' : './',
    plugins: [vue()],
    server: {
      host: '0.0.0.0',
      port: 5174,
      proxy: {
        '/api': {
          target: env.VITE_API_PROXY_TARGET || 'http://localhost:8001',
          changeOrigin: true
        }
      }
    }
  };
});
