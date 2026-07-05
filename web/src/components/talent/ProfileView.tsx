import AssignmentTurnedInOutlined from '@mui/icons-material/AssignmentTurnedInOutlined';
import FormatQuoteRounded from '@mui/icons-material/FormatQuoteRounded';
import VerifiedOutlined from '@mui/icons-material/VerifiedOutlined';
import { Box, Card, CardContent, Chip, Divider, LinearProgress, Stack, Typography } from '@mui/material';

import type { TalentProfile } from '../../api/types';
import { passportColor, passportLabel, shortId } from '../../lib/format';
import { fonts } from '../../theme/tokens';

export function ProfileView({ profile }: { profile: TalentProfile }) {
  const averageLevel =
    profile.competencies.length > 0
      ? profile.competencies.reduce((sum, competency) => sum + competency.level, 0) / profile.competencies.length
      : 0;
  const citedCompetencies = profile.competencies.filter((competency) => competency.evidenceQuote.trim().length > 0).length;

  return (
    <Card
      variant="outlined"
      sx={{
        borderRadius: '8px',
        overflow: 'hidden',
        bgcolor: 'background.paper',
        boxShadow: '0 22px 70px rgba(17, 24, 39, 0.08)',
      }}
    >
      <CardContent sx={{ p: 0, '&:last-child': { pb: 0 } }}>
        <Stack>
          <Box
            sx={{
              p: { xs: 2.5, sm: 3 },
              borderBottom: '1px solid',
              borderColor: 'divider',
              bgcolor: 'rgba(0, 102, 204, 0.055)',
            }}
          >
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} sx={{ alignItems: { xs: 'flex-start', sm: 'center' } }}>
              <Box
                aria-hidden="true"
                sx={{
                  display: 'grid',
                  placeItems: 'center',
                  width: 48,
                  height: 48,
                  borderRadius: '8px',
                  bgcolor: 'primary.main',
                  color: '#fff',
                  flex: '0 0 auto',
                }}
              >
                <VerifiedOutlined />
              </Box>
              <Box sx={{ minWidth: 0, flexGrow: 1 }}>
                <Typography
                  sx={{
                    fontFamily: fonts.mono,
                    fontSize: 12,
                    fontWeight: 750,
                    letterSpacing: 0,
                    textTransform: 'uppercase',
                    color: 'text.secondary',
                  }}
                >
                  Evidence passport
                </Typography>
                <Typography variant="h5" component="h2" sx={{ mt: 0.4, fontFamily: fonts.body, fontWeight: 850, letterSpacing: 0, lineHeight: 1.1 }}>
                  Your Talent Passport
                </Typography>
              </Box>
              <Stack direction="row" spacing={1} useFlexGap sx={{ flexWrap: 'wrap' }}>
                <Chip color={passportColor[profile.passportStatus]} label={passportLabel[profile.passportStatus]} sx={{ borderRadius: '999px', fontWeight: 800 }} />
                <Chip variant="outlined" label={shortId(profile.candidateId)} sx={{ borderRadius: '999px', fontFamily: fonts.mono, fontWeight: 750 }} />
              </Stack>
            </Stack>
          </Box>

          <Box sx={{ p: { xs: 2.5, sm: 3 } }}>
            <Box
              sx={{
                p: { xs: 2, sm: 2.25 },
                border: '1px solid',
                borderColor: 'divider',
                borderRadius: '8px',
                bgcolor: 'rgba(17, 24, 39, 0.025)',
              }}
            >
              <Typography sx={{ fontWeight: 850, lineHeight: 1.25 }}>Profile brief</Typography>
              {profile.summary ? (
                <Typography color="text.secondary" sx={{ mt: 1, lineHeight: 1.65 }}>
                  {profile.summary}
                </Typography>
              ) : (
                <Typography color="text.secondary" sx={{ mt: 1, lineHeight: 1.65 }}>
                  No summary has been extracted yet.
                </Typography>
              )}
            </Box>

            <Stack
              direction={{ xs: 'column', sm: 'row' }}
              spacing={1.5}
              sx={{ mt: 2.5 }}
            >
              {[
                { label: 'Competencies', value: profile.competencies.length.toString() },
                { label: 'Average signal', value: `${averageLevel.toFixed(1)} / 5` },
                { label: 'Cited claims', value: citedCompetencies.toString() },
              ].map((item) => (
                <Box
                  key={item.label}
                  sx={{
                    flex: 1,
                    minWidth: 0,
                    p: 1.75,
                    border: '1px solid',
                    borderColor: 'divider',
                    borderRadius: '8px',
                    bgcolor: 'background.paper',
                  }}
                >
                  <Typography sx={{ fontFamily: fonts.mono, fontSize: 11, color: 'text.secondary', textTransform: 'uppercase' }}>
                    {item.label}
                  </Typography>
                  <Typography sx={{ mt: 0.5, fontSize: 24, fontWeight: 850, lineHeight: 1 }}>
                    {item.value}
                  </Typography>
                </Box>
              ))}
            </Stack>

            <Divider sx={{ my: 3 }} />

            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} sx={{ alignItems: { xs: 'flex-start', sm: 'center' }, justifyContent: 'space-between', mb: 1.75 }}>
              <Box>
                <Typography sx={{ fontWeight: 850, lineHeight: 1.2 }}>Competency evidence</Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mt: 0.35 }}>
                  Every score stays tied to the source material already in the passport.
                </Typography>
              </Box>
              <Chip
                icon={<AssignmentTurnedInOutlined aria-hidden="true" />}
                label={`${citedCompetencies}/${profile.competencies.length} cited`}
                variant="outlined"
                sx={{ borderRadius: '999px', fontWeight: 750 }}
              />
            </Stack>

            {profile.competencies.length === 0 ? (
              <Typography variant="body2" color="text.secondary">
                No competencies have been extracted yet.
              </Typography>
            ) : (
              <Stack spacing={1.5}>
                {profile.competencies.map((c, i) => (
                  <Box
                    key={i}
                    sx={{
                      p: { xs: 1.75, sm: 2 },
                      border: '1px solid',
                      borderColor: 'divider',
                      borderRadius: '8px',
                      bgcolor: 'rgba(17, 24, 39, 0.015)',
                    }}
                  >
                    <Stack direction="row" spacing={1.5} sx={{ justifyContent: 'space-between', alignItems: 'baseline', mb: 1 }}>
                      <Typography sx={{ fontWeight: 850, lineHeight: 1.2 }}>{c.name}</Typography>
                      <Typography sx={{ fontFamily: fonts.mono, fontSize: 13, color: 'text.secondary', whiteSpace: 'nowrap' }}>
                        {c.level.toFixed(1)} / 5
                      </Typography>
                    </Stack>
                    <LinearProgress
                      variant="determinate"
                      value={(c.level / 5) * 100}
                      sx={{
                        height: 8,
                        borderRadius: '999px',
                        bgcolor: 'rgba(0, 102, 204, 0.18)',
                        '& .MuiLinearProgress-bar': { borderRadius: '999px' },
                      }}
                    />
                    {c.evidenceQuote && (
                      <Box sx={{ display: 'grid', gridTemplateColumns: 'auto 1fr', gap: 1.25, mt: 1.4, alignItems: 'start' }}>
                        <FormatQuoteRounded aria-hidden="true" sx={{ color: 'primary.main', fontSize: 20, mt: 0.1 }} />
                        <Box sx={{ minWidth: 0 }}>
                          <Typography variant="body2" color="text.secondary" sx={{ lineHeight: 1.55 }}>
                            “{c.evidenceQuote}”
                          </Typography>
                          {c.sourceSpan && (
                            <Typography sx={{ mt: 0.5, fontFamily: fonts.mono, fontSize: 11, color: 'text.secondary' }}>
                              Source: {c.sourceSpan}
                            </Typography>
                          )}
                        </Box>
                      </Box>
                    )}
                  </Box>
                ))}
              </Stack>
            )}
          </Box>
        </Stack>
      </CardContent>
    </Card>
  );
}
