import ExpandMoreOutlined from '@mui/icons-material/ExpandMoreOutlined';
import FilterAltOffRounded from '@mui/icons-material/FilterAltOffRounded';
import InfoOutlined from '@mui/icons-material/InfoOutlined';
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Alert,
  Box,
  Chip,
  Link as MuiLink,
  Stack,
  Typography,
} from '@mui/material';
import { AnimatePresence, motion } from 'motion/react';
import { useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';

import { ApiError } from '../../api/types';
import { useShortlist } from '../../query/flow';
import { fonts } from '../../theme/tokens';
import { CardListSkeleton } from '../Skeletons';
import { DotsButton } from '../DotsButton';
import { shortId } from '../../lib/format';
import { PageControls } from '../PageControls';
import { MatchCard } from './MatchCard';

const PAGE_SIZE = 20;
const summaryChipSx = {
  borderRadius: '999px',
  fontFamily: fonts.body,
  fontWeight: 750,
  height: 30,
  '& .MuiChip-label': { px: 1.25 },
};

const gateChipSx = {
  ...summaryChipSx,
  height: 28,
  textTransform: 'capitalize',
};

function candidateLabel(count: number, suffix: string) {
  return `${count} candidate${count === 1 ? '' : 's'} ${suffix}`;
}

export function ShortlistSection({ roleId, version }: { roleId: string; version: number }) {
  const [run, setRun] = useState(false);
  const pageKey = `${roleId}:${version}`;
  const [pageState, setPageState] = useState({ key: pageKey, page: 1 });
  const page = pageState.key === pageKey ? pageState.page : 1;
  const setCurrentPage = (next: number) => setPageState({ key: pageKey, page: next });
  const query = useShortlist(roleId, page, PAGE_SIZE, run, version);

  if (!run) {
    return (
      <Stack spacing={1.5} sx={{ alignItems: 'flex-start' }}>
        <Typography variant="h6" component="h2">Explainable shortlist</Typography>
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
        <Typography variant="h6" component="h2">Ranking candidates…</Typography>
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
        <Typography variant="h6" component="h2">Explainable shortlist</Typography>
        <Alert severity="info" action={<DotsButton onClick={() => query.refetch()}>Retry</DotsButton>}>
          {message}
        </Alert>
      </Stack>
    );
  }

  const { matches, exclusions, poolDepth } = query.data.shortlist;
  const filteredLabel = candidateLabel(exclusions.length, 'filtered out');
  return (
    <Stack spacing={2.25}>
      <Stack
        direction={{ xs: 'column', sm: 'row' }}
        spacing={1.5}
        useFlexGap
        sx={{ alignItems: { xs: 'flex-start', sm: 'center' }, justifyContent: 'space-between' }}
      >
        <Box sx={{ minWidth: 0 }}>
          <Typography
            variant="h5"
            component="h2"
            sx={{ fontFamily: fonts.body, fontWeight: 850, letterSpacing: 0, lineHeight: 1.15 }}
          >
            Explainable shortlist
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
            Every visible decision keeps its evidence, filter, or reason attached.
          </Typography>
        </Box>
        <Stack direction="row" spacing={1} useFlexGap sx={{ flexWrap: 'wrap' }}>
          <Chip size="small" label={`${matches.length} advanced`} color={matches.length > 0 ? 'primary' : 'default'} sx={summaryChipSx} />
          <Chip size="small" label={`${poolDepth} in pool`} variant="outlined" sx={summaryChipSx} />
          {exclusions.length > 0 && <Chip size="small" label={`${exclusions.length} filtered`} variant="outlined" sx={summaryChipSx} />}
        </Stack>
      </Stack>

      {matches.length === 0 ? (
        <Box
          role="status"
          sx={{
            display: 'grid',
            gridTemplateColumns: { xs: '1fr', sm: 'auto 1fr' },
            gap: 1.5,
            alignItems: 'center',
            p: { xs: 2, sm: 2.25 },
            border: '1px solid',
            borderColor: 'rgba(0, 102, 204, 0.22)',
            borderRadius: '8px',
            bgcolor: 'rgba(0, 102, 204, 0.06)',
          }}
        >
          <Box
            sx={{
              display: 'grid',
              placeItems: 'center',
              width: 42,
              height: 42,
              borderRadius: '8px',
              bgcolor: 'rgba(0, 102, 204, 0.1)',
              color: 'primary.main',
            }}
          >
            <InfoOutlined aria-hidden="true" />
          </Box>
          <Box sx={{ minWidth: 0 }}>
            <Typography sx={{ fontWeight: 800, lineHeight: 1.25 }}>No candidates cleared the rubric and hard filters yet.</Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mt: 0.4 }}>
              Review the filtered list below to see exactly which gate stopped each candidate.
            </Typography>
          </Box>
        </Box>
      ) : (
        <Stack spacing={2}>
          <AnimatePresence mode="popLayout">
            {matches.map((m, i) => (
              <motion.div
                key={m.id || m.candidateId}
                layout
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -8 }}
                transition={{ duration: 0.18, ease: 'easeOut' }}
              >
                <MatchCard match={m} rank={(page - 1) * PAGE_SIZE + i + 1} />
              </motion.div>
            ))}
          </AnimatePresence>
          {query.data.shortlist.page && (
            <PageControls page={page} pageCount={query.data.shortlist.page.totalPages} onChange={setCurrentPage} />
          )}
        </Stack>
      )}

      {exclusions.length > 0 && (
        <Accordion
          variant="outlined"
          disableGutters
          defaultExpanded={matches.length === 0}
          sx={{
            borderRadius: '8px',
            overflow: 'hidden',
            bgcolor: 'background.paper',
            '&:before': { display: 'none' },
          }}
        >
          <AccordionSummary
            expandIcon={<ExpandMoreOutlined aria-hidden="true" />}
            sx={{
              minHeight: 72,
              px: { xs: 2, sm: 2.5 },
              '& .MuiAccordionSummary-content': { alignItems: 'center', gap: 1.5, my: 1.5 },
            }}
          >
            <Box
              sx={{
                display: 'grid',
                placeItems: 'center',
                width: 40,
                height: 40,
                borderRadius: '8px',
                bgcolor: 'rgba(107, 114, 128, 0.1)',
                color: 'text.secondary',
                flex: '0 0 auto',
              }}
            >
              <FilterAltOffRounded aria-hidden="true" />
            </Box>
            <Box sx={{ minWidth: 0, flexGrow: 1 }}>
              <Typography sx={{ fontWeight: 850, lineHeight: 1.2 }}>{filteredLabel}</Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mt: 0.35 }}>
                Hard-filter outcomes stay visible for review and tuning.
              </Typography>
            </Box>
          </AccordionSummary>
          <AccordionDetails sx={{ px: { xs: 2, sm: 2.5 }, pb: 2.5, pt: 0 }}>
            <Stack component="ol" spacing={1} sx={{ m: 0, p: 0, listStyle: 'none' }}>
              {exclusions.map((e, i) => (
                <Box
                  key={i}
                  component="li"
                  sx={{
                    display: 'grid',
                    gridTemplateColumns: { xs: '1fr', md: 'minmax(220px, 0.35fr) 1fr' },
                    gap: { xs: 0.75, md: 2 },
                    alignItems: 'start',
                    p: 1.5,
                    border: '1px solid',
                    borderColor: 'divider',
                    borderRadius: '8px',
                    bgcolor: 'rgba(17, 24, 39, 0.015)',
                  }}
                >
                  <Stack direction="row" spacing={1} useFlexGap sx={{ alignItems: 'center', flexWrap: 'wrap', minWidth: 0 }}>
                    <Chip size="small" variant="outlined" label={e.gate} sx={gateChipSx} />
                    <MuiLink
                      component={RouterLink}
                      to={`/candidates/${e.candidateId}`}
                      variant="body2"
                      sx={{ fontFamily: fonts.mono, fontWeight: 750 }}
                    >
                      {shortId(e.candidateId)}
                    </MuiLink>
                  </Stack>
                  <Typography variant="body2" color="text.secondary" sx={{ lineHeight: 1.55 }}>
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
