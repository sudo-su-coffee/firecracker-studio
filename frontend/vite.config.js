import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api/v1': {
        target: process.env.FIRECRACKER_STUDIO_API || 'http://127.0.0.1:7822',
        changeOrigin: true,
      },
    },
  },
})
