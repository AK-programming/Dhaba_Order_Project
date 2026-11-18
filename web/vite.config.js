import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Dev only: forward /api/* to the services running on localhost.
// In Docker and Kubernetes, Nginx does this (see nginx.conf).
const proxy = (target, name) => ({
  target,
  changeOrigin: true,
  rewrite: p => p.replace(new RegExp(`^/api/${name}`), `/${name}`),
})

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api/menu':   proxy('http://localhost:8000', 'menu'),
      '/api/orders': proxy('http://localhost:8080', 'orders'),
      '/api/info':   proxy('http://localhost:8080', 'info'),
      '/api/stats':  proxy('http://localhost:9000', 'stats'),
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.js',
  },
})
