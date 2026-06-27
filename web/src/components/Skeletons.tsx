import { Card, CardContent, Skeleton, Stack } from '@mui/material';

export function CardSkeleton({ lines = 3 }: { lines?: number }) {
  return (
    <Card variant="outlined" role="status" aria-label="Loading content">
      <CardContent>
        <Stack spacing={1.2}>
          <Skeleton variant="text" width="45%" height={28} />
          {Array.from({ length: lines }).map((_, i) => (
            <Skeleton key={i} variant="text" width={`${85 - i * 12}%`} />
          ))}
        </Stack>
      </CardContent>
    </Card>
  );
}

export function CardListSkeleton({ count = 3 }: { count?: number }) {
  return (
    <Stack spacing={2}>
      {Array.from({ length: count }).map((_, i) => (
        <CardSkeleton key={i} lines={4} />
      ))}
    </Stack>
  );
}
