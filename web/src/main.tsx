import { CssBaseline, ThemeProvider } from '@mui/material';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

import { App } from './App';
import './theme/fonts';
import { theme } from './theme/theme';

const container = document.getElementById('root');
if (!container) {
  throw new Error('root container missing');
}

createRoot(container).render(
  <StrictMode>
    <ThemeProvider theme={theme} defaultMode="system">
      <CssBaseline />
      <App />
    </ThemeProvider>
  </StrictMode>,
);
