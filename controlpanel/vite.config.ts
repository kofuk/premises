import react from '@vitejs/plugin-react';
import checker from 'vite-plugin-checker';
import tsconfigPaths from 'vite-tsconfig-paths';
import {defineConfig} from 'vitest/config';

export default defineConfig({
  server: {
    open: false,
    watch: {
      ignored: ['!frontend/**']
    }
  },
  build: {
    outDir: 'gen'
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./test-setup.ts']
  },
  plugins: [react(), tsconfigPaths(), checker({typescript: true})]
});
