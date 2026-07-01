import createEmotionCache from '@emotion/cache';
import { CacheProvider } from '@emotion/react';
import { QueryClientProvider } from '@tanstack/react-query';
import { CssBaseline, ThemeProvider } from '@mui/material';
import { MotionConfig } from 'motion/react';
import { renderToString } from 'react-dom/server';
import { StaticRouter } from 'react-router-dom';

import { AppRoutes } from './App';
import { queryClient } from './query/client';
import { theme } from './theme/theme';

/**
 * SSR entrypoint used by the build-time prerender script.
 *
 * It mirrors the global providers from main.tsx but replaces BrowserRouter with
 * StaticRouter so public routes can be rendered to a string for a given URL.
 */
export function render(url: string) {
  // A dedicated Emotion cache for each prerendered route. MUI components will
  // consume it via Emotion's React context, and the generated CSS class names
  // will align with the client-side hydration pass.
  const cache = createEmotionCache({ key: 'css', prepend: true });

  const html = renderToString(
    <CacheProvider value={cache}>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider theme={theme} defaultMode="light">
          <CssBaseline />
          <MotionConfig reducedMotion="user">
            <StaticRouter location={url}>
              <AppRoutes />
            </StaticRouter>
          </MotionConfig>
        </ThemeProvider>
      </QueryClientProvider>
    </CacheProvider>,
  );

  return { html };
}
