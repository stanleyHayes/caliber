import '@testing-library/jest-dom/vitest';

import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

// With globals disabled, RTL's automatic afterEach(cleanup) isn't registered, so
// unmount + reset the DOM between tests ourselves to keep them isolated.
afterEach(() => {
  cleanup();
});
