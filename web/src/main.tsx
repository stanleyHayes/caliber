import { CssBaseline, GlobalStyles, ThemeProvider } from '@mui/material';
import { MotionConfig } from 'motion/react';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

import { App } from './App';
import { I18nProvider } from './i18n/I18nProvider';
import './theme/fonts';
import { theme } from './theme/theme';

const container = document.getElementById('root');
if (!container) {
  throw new Error('root container missing');
}

// The circular-reveal toggle drives the clip-path; suppress the default
// view-transition cross-fade so the reveal is clean.
const viewTransitionStyles = (
  <GlobalStyles
    styles={{
      '::view-transition-old(root), ::view-transition-new(root)': {
        animation: 'none',
        mixBlendMode: 'normal',
      },
    }}
  />
);

createRoot(container).render(
  <StrictMode>
    <ThemeProvider theme={theme} defaultMode="system">
      <CssBaseline />
      {viewTransitionStyles}
      <I18nProvider>
        <MotionConfig reducedMotion="user">
          <App />
        </MotionConfig>
      </I18nProvider>
    </ThemeProvider>
  </StrictMode>,
);
