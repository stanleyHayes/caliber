// Brandable design tokens (single source of truth). Colors map to the MUI theme
// palette in theme.ts; the mono family is used for statuses/codes.
export const brand = {
  primaryBlue: '#0066CC',
  ink: '#111418',
  slate: '#6B7280',
} as const;

export const fonts = {
  title: '"Fraunces Variable", Georgia, "Times New Roman", serif',
  body: '"Outfit Variable", system-ui, -apple-system, Segoe UI, Roboto, sans-serif',
  mono: '"JetBrains Mono Variable", ui-monospace, SFMono-Regular, Menlo, monospace',
} as const;
