import path = require('path');
import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';
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
