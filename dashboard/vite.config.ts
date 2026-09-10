import path from 'node:path';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';
import { tanstackRouter } from '@tanstack/router-plugin/vite';

export default defineConfig({
  plugins: [tanstackRouter({ target: 'react', autoCodeSplitting: true }), react()],
  server: {
    host: '127.0.0.1',
    proxy: Object.fromEntries(['/api', '/auth', '/config'].map(prefix => [prefix, {
      target: process.env.WARPTAIL_API_URL || 'http://localhost:8001',
      changeOrigin: false,
    }])),
  },
  resolve: { alias: { '@': path.resolve(import.meta.dirname, './src') } },
});
