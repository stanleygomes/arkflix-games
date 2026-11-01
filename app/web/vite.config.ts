/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// Em desenvolvimento, a API vem do Go rodando em --web 127.0.0.1:8484.
export default defineConfig({
  plugins: [svelte()],
  build: { outDir: 'dist', emptyOutDir: true, target: 'es2022', assetsInlineLimit: 0 },
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:8484',
      '/media': 'http://127.0.0.1:8484',
    },
  },
  test: { include: ['tests/unit/**/*.test.ts'], environment: 'node' },
})
