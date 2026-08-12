import { defineConfig } from 'vitest/config';
import vue from '@vitejs/plugin-vue';

export default defineConfig({
  plugins: [vue()],
  test: {
    environment: 'jsdom',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
      include: ['src/utils/**/*.ts', 'src/stores/**/*.ts'],
      thresholds: { statements: 75, branches: 70, functions: 75, lines: 75 },
    },
  },
});
