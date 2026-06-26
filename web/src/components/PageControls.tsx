import { Pagination, Stack } from '@mui/material';

// Reusable 1-based pagination control. Server pages map directly to UI pages;
// hidden when there is a single page (no unbounded or noisy controls).
export function PageControls({
  page,
  pageCount,
  onChange,
}: {
  page: number;
  pageCount: number;
  onChange: (page: number) => void;
}) {
  if (pageCount <= 1) {
    return null;
  }
  return (
    <Stack sx={{ pt: 1, alignItems: 'center' }}>
      <Pagination
        page={page}
        count={pageCount}
        onChange={(_, p) => onChange(p)}
        color="primary"
        shape="rounded"
      />
    </Stack>
  );
}
