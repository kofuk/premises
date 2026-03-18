import react from '@vitejs/plugin-react';
import type {UserConfig} from 'vite';
import {checker} from 'vite-plugin-checker';
import type {ViteUserConfig as VitestUserConfig} from 'vitest/config';

export default {
  server: {
    open: false,
    watch: {
      ignored: ['!src/**']
    },
    hmr: {
      clientPort: 8000
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
  plugins: [react(), checker({typescript: true})],
  resolve: {
    tsconfigPaths: true
  }
} satisfies UserConfig | Pick<VitestUserConfig, 'test'>;
