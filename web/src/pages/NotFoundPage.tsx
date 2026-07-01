import { Button, Stack, Typography } from '@mui/material';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';

export function NotFoundPage() {
  const { t } = useTranslation();
  return (
    <Stack spacing={2} sx={{ py: 6, alignItems: 'flex-start' }}>
      <Typography variant="h3" component="h1">{t('notFound.heading')}</Typography>
      <Typography color="text.secondary">{t('notFound.message')}</Typography>
      <Button component={Link} to="/" variant="contained">
        {t('notFound.backHome')}
      </Button>
    </Stack>
  );
}
