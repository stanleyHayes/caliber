import { Box, Button, type ButtonProps } from '@mui/material';

function Dots() {
  return (
    <Box
      component="span"
      aria-label="loading"
      sx={{
        display: 'inline-flex',
        gap: 0.6,
        '@keyframes caliberBlink': { '0%,80%,100%': { opacity: 0.25 }, '40%': { opacity: 1 } },
      }}
    >
      {[0, 1, 2].map((i) => (
        <Box
          key={i}
          component="span"
          sx={{
            width: 6,
            height: 6,
            borderRadius: '50%',
            bgcolor: 'currentColor',
            animation: 'caliberBlink 1.2s infinite ease-in-out',
            animationDelay: `${i * 0.18}s`,
            '@media (prefers-reduced-motion: reduce)': { animation: 'none', opacity: 0.5 },
          }}
        />
      ))}
    </Box>
  );
}

// DotsButton shows width-stable animated dots while loading instead of a spinner.
export function DotsButton({ loading, children, disabled, ...props }: ButtonProps & { loading?: boolean }) {
  return (
    <Button {...props} disabled={disabled ?? loading}>
      {loading ? <Dots /> : children}
    </Button>
  );
}
