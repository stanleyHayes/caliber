import { createTheme } from '@mui/material/styles';

import { brand, fonts } from './tokens';

// MUI v9 CSS-variable theme with light + dark color schemes. The class-based
// selector lets the circular-reveal toggle (CAL-169) flip modes via a root class.
export const theme = createTheme({
  cssVariables: { colorSchemeSelector: 'class' },
  colorSchemes: {
    light: {
      palette: {
        primary: { main: brand.primaryBlue },
        text: { primary: brand.ink, secondary: brand.slate },
        background: { default: '#F7F9FC', paper: '#FFFFFF' },
      },
    },
    dark: {
      palette: {
        primary: { main: '#4D9BFF' },
        text: { primary: '#F4F6F8', secondary: '#9AA4B2' },
        background: { default: '#0B0E11', paper: '#12161B' },
      },
    },
  },
  shape: { borderRadius: 10 },
  typography: {
    fontFamily: fonts.body,
    h1: { fontFamily: fonts.title, fontWeight: 600, letterSpacing: '-0.02em' },
    h2: { fontFamily: fonts.title, fontWeight: 600, letterSpacing: '-0.02em' },
    h3: { fontFamily: fonts.title, fontWeight: 600 },
    h4: { fontFamily: fonts.title, fontWeight: 600 },
    h5: { fontFamily: fonts.title },
    h6: { fontFamily: fonts.title },
    button: { textTransform: 'none', fontWeight: 600 },
  },
});
