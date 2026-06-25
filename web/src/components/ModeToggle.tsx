import { DarkModeOutlined, LightModeOutlined } from '@mui/icons-material';
import { IconButton, Tooltip, useColorScheme } from '@mui/material';

export function ModeToggle() {
  const { mode, setMode } = useColorScheme();
  const resolved = mode === 'system' ? undefined : mode;
  const next = resolved === 'dark' ? 'light' : 'dark';
  return (
    <Tooltip title={`Switch to ${next} mode`}>
      <IconButton onClick={() => setMode(next)} color="inherit" aria-label={`switch to ${next} mode`}>
        {resolved === 'dark' ? <LightModeOutlined /> : <DarkModeOutlined />}
      </IconButton>
    </Tooltip>
  );
}
