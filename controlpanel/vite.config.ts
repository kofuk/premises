import react from '@vitejs/plugin-react';
import {defineConfig} from 'vite';
import tsconfigPaths from 'vite-tsconfig-paths';

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
  plugins: [react(), tsconfigPaths()]
});
