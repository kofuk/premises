import react from '@vitejs/plugin-react';
import checker from 'vite-plugin-checker';
import tsconfigPaths from 'vite-tsconfig-paths';
import {UserConfig} from 'vitest/config';

export default {
  server: {
    open: false,
    watch: {
      ignored: ['!src/**']
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
} satisfies UserConfig;
