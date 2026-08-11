import { defineConfig } from 'vitest/config';

export default defineConfig({
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
