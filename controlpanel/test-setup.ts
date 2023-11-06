import '@testing-library/jest-dom';

import fetch from 'node-fetch';

beforeEach(() => {
  global.fetch = fetch as unknown as typeof global.fetch;
});
