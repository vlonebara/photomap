import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig, loadEnv } from 'vite'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, '.', 'PHOTOMAP_')

  return {
    plugins: [vue(), tailwindcss()],
    server: {
      proxy: {
        '/api': {
          target: env.PHOTOMAP_API_TARGET || 'http://localhost:8080',
          // keep the browser host so it matches the origin checked by Go
          changeOrigin: false,
        },
      },
    },
  }
})
