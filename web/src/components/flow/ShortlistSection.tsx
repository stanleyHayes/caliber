import { ExpandMoreOutlined } from '@mui/icons-material';
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Alert,
  Box,
  Chip,
  Stack,
  Typography,
} from '@mui/material';
import { useState } from 'react';

import { ApiError } from '../../api/types';
import { useShortlist } from '../../query/flow';
import { CardListSkeleton } from '../Skeletons';
import { DotsButton } from '../DotsButton';
import { shortId } from '../../lib/format';
import { MatchCard } from './MatchCard';

const PAGE_SIZE = 20;

export function ShortlistSection({ roleId, version }: { roleId: string; version: number }) {
  const [run, setRun] = useState(false);
  const query = useShortlist(roleId, PAGE_SIZE, run, version);

  if (!run) {
    return (
      <Stack spacing={1.5} sx={{ alignItems: 'flex-start' }}>
        <Typography variant="h6">Explainable shortlist</Typography>
        <Typography variant="body2" color="text.secondary">
          Rank candidates in your pool against this rubric — every score traces back to evidence.
        </Typography>
        <DotsButton variant="contained" onClick={() => setRun(true)}>
          Generate shortlist
        </DotsButton>
      </Stack>
    );
  }

  if (query.isPending) {
    return (
      <Stack spacing={2}>
        <Typography variant="h6">Ranking candidates…</Typography>
        <CardListSkeleton count={3} />
      </Stack>
    );
  }

  if (query.isError) {
    const status = query.error instanceof ApiError ? query.error.status : 0;
    const message =
      status === 501
        ? 'Matching needs the configured environment (database + embeddings). Connect them to rank candidates.'
        : query.error instanceof Error
          ? query.error.message
          : 'Could not generate the shortlist.';
    return (
      <Stack spacing={2}>
        <Typography variant="h6">Explainable shortlist</Typography>
        <Alert severity="info" action={<DotsButton onClick={() => query.refetch()}>Retry</DotsButton>}>
          {message}
        </Alert>
      </Stack>
    );
  }

  const { matches, exclusions, poolDepth } = query.data.shortlist;
  return (
    <Stack spacing={2}>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'baseline' }}>
        <Typography variant="h6">Explainable shortlist</Typography>
        <Chip size="small" label={`${poolDepth} in pool`} />
      </Stack>

      {matches.length === 0 ? (
        <Alert severity="info">No candidates cleared the rubric and hard filters yet.</Alert>
      ) : (
        <Stack spacing={2}>
          {matches.map((m, i) => (
            <MatchCard key={m.id || m.candidateId} match={m} rank={i + 1} />
          ))}
        </Stack>
      )}

      {exclusions.length > 0 && (
        <Accordion variant="outlined" disableGutters>
          <AccordionSummary expandIcon={<ExpandMoreOutlined />}>
            <Typography variant="body2" color="text.secondary">
              {exclusions.length} candidate{exclusions.length > 1 ? 's' : ''} filtered out
            </Typography>
          </AccordionSummary>
          <AccordionDetails>
            <Stack spacing={1}>
              {exclusions.map((e, i) => (
                <Box key={i}>
                  <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                    <Chip size="small" variant="outlined" label={e.gate} />
                    <Typography variant="body2" component="span">
                      {shortId(e.candidateId)}
                    </Typography>
                  </Stack>
                  <Typography variant="caption" color="text.secondary">
                    {e.reason}
                  </Typography>
                </Box>
              ))}
            </Stack>
          </AccordionDetails>
        </Accordion>
      )}
    </Stack>
  );
}
