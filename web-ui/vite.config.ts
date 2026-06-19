import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { readFileSync } from 'node:fs';

// Single source of truth for the displayed app version: package.json.
const pkg = JSON.parse(
  readFileSync(new URL('./package.json', import.meta.url), 'utf-8'),
) as { version: string };

export default defineConfig({
  plugins: [react()],
  define: {
    __APP_VERSION__: JSON.stringify(pkg.version),
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: process.env.API_BASE_URL || 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
    rollupOptions: {
      output: {
        // Split heavy vendors into their own chunks so the initial bundle does
        // not ship echarts (~1MB, only used by the node-detail page) to every
        // Login/Dashboard session.
        manualChunks: {
          react: ['react', 'react-dom', 'react-router-dom'],
          antd: ['antd', '@ant-design/icons'],
          echarts: ['echarts', 'echarts-for-react'],
          query: ['@tanstack/react-query'],
        },
      },
    },
  },
});
