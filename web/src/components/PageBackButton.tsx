import ArrowBackRounded from '@mui/icons-material/ArrowBackRounded';
import { Button } from '@mui/material';
import { useInRouterContext, useLocation, useNavigate } from 'react-router-dom';

import { fonts } from '../theme/tokens';

type PageBackButtonProps = {
  fallbackTo?: string;
  label?: string;
  alwaysFallback?: boolean;
};

function fromStatePath(state: unknown, currentPath: string): string | null {
  if (!state || typeof state !== 'object' || !('from' in state)) {
    return null;
  }

  const from = (state as { from?: unknown }).from;
  if (typeof from !== 'string' || from.length === 0 || from === currentPath) {
    return null;
  }

  return from;
}

function BackButtonView({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <Button
      type="button"
      variant="text"
      size="small"
      startIcon={<ArrowBackRounded fontSize="small" />}
      onClick={onClick}
      sx={{
        alignSelf: 'flex-start',
        borderRadius: '8px',
        fontFamily: fonts.body,
        fontWeight: 800,
        minHeight: 38,
        px: 0,
      }}
    >
      {label}
    </Button>
  );
}

function RouterBackButton({ fallbackTo, label, alwaysFallback }: Required<PageBackButtonProps>) {
  const navigate = useNavigate();
  const location = useLocation();
  const currentPath = `${location.pathname}${location.search}${location.hash}`;

  const onClick = () => {
    if (alwaysFallback) {
      navigate(fallbackTo, { replace: true });
      return;
    }

    const from = fromStatePath(location.state, currentPath);
    if (from) {
      navigate(from);
      return;
    }

    if (location.key && location.key !== 'default') {
      navigate(-1);
      return;
    }

    navigate(fallbackTo, { replace: true });
  };

  return <BackButtonView label={label} onClick={onClick} />;
}

export function PageBackButton({ fallbackTo = '/app', label = 'Back', alwaysFallback = false }: PageBackButtonProps) {
  const inRouter = useInRouterContext();

  if (!inRouter) {
    return (
      <BackButtonView
        label={label}
        onClick={() => {
          if (!alwaysFallback && window.history.length > 1) {
            window.history.back();
          } else {
            window.location.assign(fallbackTo);
          }
        }}
      />
    );
  }

  return <RouterBackButton fallbackTo={fallbackTo} label={label} alwaysFallback={alwaysFallback} />;
}
