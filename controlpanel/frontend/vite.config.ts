import react from '@vitejs/plugin-react';
import type {UserConfig} from 'vite';
import checker from 'vite-plugin-checker';
import tsconfigPaths from 'vite-tsconfig-paths';
import type {ViteUserConfig as VitestUserConfig} from 'vitest/config';

export default {
  server: {
    open: false,
    watch: {
      ignored: ['!src/**']
    },
    hmr: {
      clientPort: 8888
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
} satisfies UserConfig | Pick<VitestUserConfig, 'test'>;
