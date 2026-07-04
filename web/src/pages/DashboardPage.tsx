import AddCircleOutlineRounded from '@mui/icons-material/AddCircleOutlineRounded';
import ArrowForwardRounded from '@mui/icons-material/ArrowForwardRounded';
import AssignmentTurnedInOutlined from '@mui/icons-material/AssignmentTurnedInOutlined';
import BadgeOutlined from '@mui/icons-material/BadgeOutlined';
import FactCheckOutlined from '@mui/icons-material/FactCheckOutlined';
import HubOutlined from '@mui/icons-material/HubOutlined';
import KeyboardVoiceOutlined from '@mui/icons-material/KeyboardVoiceOutlined';
import PersonSearchOutlined from '@mui/icons-material/PersonSearchOutlined';
import PsychologyAltOutlined from '@mui/icons-material/PsychologyAltOutlined';
import RadarRounded from '@mui/icons-material/RadarRounded';
import ShieldOutlined from '@mui/icons-material/ShieldOutlined';
import WorkOutlineRounded from '@mui/icons-material/WorkOutlineRounded';
import { Box, Button, Card, CardContent, Chip, Skeleton, Stack, Typography } from '@mui/material';
import type { SvgIconProps } from '@mui/material/SvgIcon';
import type { ComponentType } from 'react';
import { Link } from 'react-router-dom';

import type { UserRole } from '../api/types';
import { useMe } from '../query/auth';
import { useAuthStore } from '../stores/auth';
import { fonts } from '../theme/tokens';

const ROLE_LABEL: Record<UserRole, string> = {
  USER_ROLE_UNSPECIFIED: 'Member',
  USER_ROLE_EMPLOYER: 'Employer',
  USER_ROLE_RECRUITER: 'Recruiter',
  USER_ROLE_CANDIDATE: 'Candidate',
};

const NEXT_BY_ROLE: Record<UserRole, string> = {
  USER_ROLE_UNSPECIFIED: 'Your workspace is being set up.',
  USER_ROLE_EMPLOYER: 'Describe a role in plain language to generate an explainable shortlist.',
  USER_ROLE_RECRUITER: 'Describe a role in plain language to generate an explainable shortlist.',
  USER_ROLE_CANDIDATE: 'Complete your Talent Passport to get matched to roles.',
};

type ActionCard = {
  title: string;
  body: string;
  label: string;
  to: string;
  Icon: ComponentType<SvgIconProps>;
  tone: string;
};

const EMPLOYER_ACTIONS: ActionCard[] = [
  {
    title: 'Role brief',
    body: 'Turn a plain-language hiring need into a structured role spec and scored rubric.',
    label: 'Describe a role',
    to: '/roles/new',
    Icon: AddCircleOutlineRounded,
    tone: '#2DD4BF',
  },
  {
    title: 'Open roles',
    body: 'Return to saved role specs, shortlist evidence, and decision notes.',
    label: 'Your roles',
    to: '/roles',
    Icon: WorkOutlineRounded,
    tone: '#60A5FA',
  },
  {
    title: 'Talent Radar',
    body: 'Scan supply, demand, alerts, and shortlist velocity before the next hiring move.',
    label: 'Open radar',
    to: '/radar',
    Icon: RadarRounded,
    tone: '#F59E0B',
  },
];

const CANDIDATE_ACTIONS: ActionCard[] = [
  {
    title: 'Talent Passport',
    body: 'Create the profile Caliber can use for explainable role matching.',
    label: 'Set up your passport',
    to: '/profile',
    Icon: BadgeOutlined,
    tone: '#2DD4BF',
  },
  {
    title: 'Screening space',
    body: 'Answer adaptive questions and receive a structured report card.',
    label: 'Screening interview',
    to: '/interview',
    Icon: KeyboardVoiceOutlined,
    tone: '#60A5FA',
  },
  {
    title: 'Autonomous agent',
    body: 'Let your candidate agent prepare applications from your evidence.',
    label: 'Run your agent',
    to: '/agent',
    Icon: PsychologyAltOutlined,
    tone: '#F59E0B',
  },
];

const REVIEW_STEPS = [
  { label: 'Intake', body: 'Plain language role brief', Icon: AssignmentTurnedInOutlined },
  { label: 'Evidence', body: 'Traceable scoring only', Icon: FactCheckOutlined },
  { label: 'Decision', body: 'Human-reviewed shortlist', Icon: PersonSearchOutlined },
] as const;

const HERO_LABELS: Record<UserRole, string[]> = {
  USER_ROLE_UNSPECIFIED: ['Start setup', 'Review workspace', 'View radar'],
  USER_ROLE_EMPLOYER: ['Start role brief', 'Review roles', 'View radar'],
  USER_ROLE_RECRUITER: ['Start role brief', 'Review roles', 'View radar'],
  USER_ROLE_CANDIDATE: ['Start passport', 'Open interview', 'Open agent'],
};

function actionsFor(role: UserRole) {
  if (role === 'USER_ROLE_CANDIDATE') {
    return CANDIDATE_ACTIONS;
  }

  return EMPLOYER_ACTIONS;
}

function CandidateDashboard({ name }: { name?: string }) {
  return (
    <Stack spacing={3} sx={{ fontFamily: fonts.body, maxWidth: 1120, mx: 'auto', color: 'text.primary' }}>
      <Box
        sx={{
          display: 'grid',
          gap: { xs: 2.5, md: 3 },
          gridTemplateColumns: { xs: '1fr', lg: 'minmax(0, 1fr) 360px' },
          alignItems: 'stretch',
        }}
      >
        <Box
          sx={{
            position: 'relative',
            overflow: 'hidden',
            minHeight: { xs: 460, md: 520 },
            border: '1px solid',
            borderColor: 'divider',
            borderRadius: '8px',
            bgcolor: 'background.paper',
            p: { xs: 3, md: 4.5 },
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'space-between',
            boxShadow: '0 26px 90px rgba(0, 0, 0, 0.24)',
            '&::before': {
              content: '""',
              position: 'absolute',
              inset: 0,
              background:
                'linear-gradient(135deg, rgba(45, 212, 191, 0.16), transparent 36%), linear-gradient(180deg, rgba(245, 158, 11, 0.12), transparent 62%)',
              pointerEvents: 'none',
            },
          }}
        >
          <Stack spacing={3} sx={{ position: 'relative', maxWidth: 760 }}>
            <Stack direction="row" spacing={1} useFlexGap sx={{ flexWrap: 'wrap', alignItems: 'center' }}>
              <Chip label="Candidate" color="primary" size="small" sx={{ borderRadius: '999px', fontFamily: fonts.body, fontWeight: 800 }} />
              <Chip
                icon={<ShieldOutlined />}
                label="Consent-led profile"
                variant="outlined"
                size="small"
                sx={{ borderRadius: '999px', fontFamily: fonts.body }}
              />
            </Stack>

            <Box>
              <Typography
                component="h1"
                sx={{
                  fontFamily: fonts.body,
                  fontSize: { xs: 40, sm: 54, md: 70 },
                  lineHeight: 0.96,
                  fontWeight: 850,
                  letterSpacing: 0,
                  maxWidth: 780,
                }}
              >
                Welcome{name ? `, ${name}` : ''}. Build the profile that carries your proof.
              </Typography>
              <Typography
                sx={{
                  mt: 2,
                  fontFamily: fonts.body,
                  color: 'text.secondary',
                  fontSize: { xs: 17, md: 20 },
                  lineHeight: 1.55,
                  maxWidth: 700,
                }}
              >
                Complete your Talent Passport, rehearse the screening room, then let your candidate agent act from evidence you control.
              </Typography>
            </Box>
          </Stack>

          <Stack
            direction={{ xs: 'column', sm: 'row' }}
            spacing={1.25}
            useFlexGap
            sx={{ position: 'relative', flexWrap: 'wrap', mt: 4 }}
          >
            <Button
              component={Link}
              to="/profile"
              variant="contained"
              endIcon={<ArrowForwardRounded />}
              sx={{ minHeight: 50, borderRadius: '8px', px: 2.6, fontFamily: fonts.body }}
            >
              Set up your passport
            </Button>
            <Button
              component={Link}
              to="/interview"
              variant="outlined"
              sx={{ minHeight: 50, borderRadius: '8px', px: 2.6, fontFamily: fonts.body }}
            >
              Screening interview
            </Button>
            <Button
              component={Link}
              to="/agent"
              variant="outlined"
              sx={{ minHeight: 50, borderRadius: '8px', px: 2.6, fontFamily: fonts.body }}
            >
              Run your agent
            </Button>
          </Stack>
        </Box>

        <Card
          variant="outlined"
          sx={{
            borderRadius: '8px',
            bgcolor: 'background.paper',
            overflow: 'hidden',
            boxShadow: '0 24px 70px rgba(0, 0, 0, 0.18)',
          }}
        >
          <CardContent sx={{ p: { xs: 2.5, md: 3 }, height: '100%' }}>
            <Stack spacing={2.25} sx={{ height: '100%' }}>
              <Box>
                <Typography
                  component="h2"
                  sx={{
                    fontFamily: fonts.body,
                    fontSize: 13,
                    fontWeight: 850,
                    letterSpacing: 0,
                    textTransform: 'uppercase',
                    color: 'primary.main',
                  }}
                >
                  Passport readiness
                </Typography>
                <Typography sx={{ mt: 0.75, fontFamily: fonts.body, color: 'text.secondary', lineHeight: 1.55 }}>
                  A clean path from profile evidence to role-fit conversations.
                </Typography>
              </Box>

              {[
                { label: 'Evidence profile', body: 'Skills, projects, work history', Icon: BadgeOutlined, tone: '#2DD4BF' },
                { label: 'Screening context', body: 'Adaptive answers and report card', Icon: KeyboardVoiceOutlined, tone: '#60A5FA' },
                { label: 'Agent control', body: 'Applications based on your facts', Icon: PsychologyAltOutlined, tone: '#F59E0B' },
              ].map(({ label, body, Icon, tone }, index) => (
                <Box
                  key={label}
                  sx={{
                    display: 'grid',
                    gridTemplateColumns: '44px 1fr auto',
                    gap: 1.5,
                    alignItems: 'center',
                    p: 1.5,
                    borderRadius: '8px',
                    border: '1px solid',
                    borderColor: 'divider',
                    bgcolor: 'rgba(255,255,255,0.03)',
                  }}
                >
                  <Box
                    sx={{
                      width: 44,
                      height: 44,
                      borderRadius: '8px',
                      display: 'grid',
                      placeItems: 'center',
                      color: tone,
                      bgcolor: `${tone}1A`,
                    }}
                  >
                    <Icon fontSize="small" />
                  </Box>
                  <Box>
                    <Typography sx={{ fontFamily: fonts.body, fontWeight: 850, lineHeight: 1.2 }}>{label}</Typography>
                    <Typography sx={{ mt: 0.25, fontFamily: fonts.body, color: 'text.secondary', fontSize: 14 }}>{body}</Typography>
                  </Box>
                  <Typography sx={{ fontFamily: fonts.mono, color: 'text.secondary', fontSize: 12 }}>0{index + 1}</Typography>
                </Box>
              ))}

              <Box
                sx={{
                  mt: 'auto',
                  p: 2,
                  borderRadius: '8px',
                  bgcolor: 'rgba(45, 212, 191, 0.1)',
                  border: '1px solid rgba(45, 212, 191, 0.24)',
                }}
              >
                <Typography sx={{ fontFamily: fonts.body, fontWeight: 850 }}>Candidate work stays yours.</Typography>
                <Typography sx={{ mt: 0.5, fontFamily: fonts.body, color: 'text.secondary', fontSize: 14, lineHeight: 1.55 }}>
                  Caliber asks for evidence before it recommends you for roles.
                </Typography>
              </Box>
            </Stack>
          </CardContent>
        </Card>
      </Box>

      <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', md: 'repeat(3, minmax(0, 1fr))' } }}>
        {CANDIDATE_ACTIONS.map(({ title, body, to, Icon, tone }) => (
          <Card
            key={title}
            variant="outlined"
            sx={{
              borderRadius: '8px',
              bgcolor: 'background.paper',
              minHeight: 250,
              transition: 'transform 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease',
              '&:hover': {
                transform: 'translateY(-3px)',
                borderColor: tone,
                boxShadow: `0 18px 50px ${tone}1F`,
              },
            }}
          >
            <CardContent sx={{ p: 2.5, height: '100%' }}>
              <Stack spacing={2} sx={{ height: '100%', alignItems: 'flex-start' }}>
                <Box
                  sx={{
                    width: 48,
                    height: 48,
                    borderRadius: '8px',
                    display: 'grid',
                    placeItems: 'center',
                    color: tone,
                    bgcolor: `${tone}1A`,
                  }}
                >
                  <Icon />
                </Box>
                <Box>
                  <Typography component="h2" sx={{ fontFamily: fonts.body, fontWeight: 850, fontSize: 21, letterSpacing: 0 }}>
                    {title}
                  </Typography>
                  <Typography sx={{ mt: 1, fontFamily: fonts.body, color: 'text.secondary', lineHeight: 1.55 }}>{body}</Typography>
                </Box>
                <Button
                  component={Link}
                  to={to}
                  variant="text"
                  endIcon={<ArrowForwardRounded />}
                  sx={{
                    mt: 'auto',
                    px: 0,
                    minWidth: 0,
                    fontFamily: fonts.body,
                    color: tone,
                    '&:hover': { bgcolor: 'transparent', textDecoration: 'underline' },
                  }}
                >
                  Open {title.toLowerCase()}
                </Button>
              </Stack>
            </CardContent>
          </Card>
        ))}
      </Box>

      <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', md: '1.2fr 0.8fr' } }}>
        <Card variant="outlined" sx={{ borderRadius: '8px', bgcolor: 'background.paper' }}>
          <CardContent sx={{ p: { xs: 2.5, md: 3 } }}>
            <Stack spacing={2.5}>
              <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                <HubOutlined color="primary" fontSize="small" />
                <Typography
                  component="h2"
                  sx={{ fontFamily: fonts.body, fontSize: 13, fontWeight: 850, letterSpacing: 0, textTransform: 'uppercase', color: 'primary.main' }}
                >
                  Next move
                </Typography>
              </Stack>
              <Typography sx={{ fontFamily: fonts.body, fontSize: { xs: 24, md: 30 }, fontWeight: 850, lineHeight: 1.1 }}>
                Complete your Talent Passport to get matched to roles.
              </Typography>
              <Typography sx={{ fontFamily: fonts.body, color: 'text.secondary', maxWidth: 720, lineHeight: 1.55 }}>
                Start with the profile. It gives the interview and agent enough verified context to be useful without fabricating experience.
              </Typography>
              <Box>
                <Button component={Link} to="/profile" variant="contained" endIcon={<ArrowForwardRounded />}>
                  Continue passport
                </Button>
              </Box>
            </Stack>
          </CardContent>
        </Card>

        <Card variant="outlined" sx={{ borderRadius: '8px', bgcolor: 'background.paper' }}>
          <CardContent sx={{ p: { xs: 2.5, md: 3 } }}>
            <Stack spacing={2}>
              <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                <FactCheckOutlined color="primary" fontSize="small" />
                <Typography
                  component="h2"
                  sx={{ fontFamily: fonts.body, fontSize: 13, fontWeight: 850, letterSpacing: 0, textTransform: 'uppercase', color: 'primary.main' }}
                >
                  Guardrails
                </Typography>
              </Stack>
              {['No invented skills or experience', 'Consent-led profile updates', 'Explainable matches and report cards'].map((item) => (
                <Box
                  key={item}
                  sx={{
                    display: 'grid',
                    gridTemplateColumns: '10px 1fr',
                    gap: 1.25,
                    alignItems: 'center',
                    py: 1,
                    borderBottom: '1px solid',
                    borderColor: 'divider',
                    '&:last-child': { borderBottom: 0 },
                  }}
                >
                  <Box sx={{ width: 8, height: 8, borderRadius: '999px', bgcolor: 'primary.main' }} />
                  <Typography sx={{ fontFamily: fonts.body, color: 'text.secondary' }}>{item}</Typography>
                </Box>
              ))}
            </Stack>
          </CardContent>
        </Card>
      </Box>
    </Stack>
  );
}

export function DashboardPage() {
  const user = useAuthStore((s) => s.user);
  const me = useMe();

  if (!user && me.isPending) {
    return (
      <Stack spacing={2.5} sx={{ maxWidth: 1120 }}>
        <Skeleton variant="rounded" width={112} height={32} />
        <Skeleton width="52%" height={58} />
        <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', md: '1.25fr 0.75fr' } }}>
          <Skeleton variant="rounded" height={280} sx={{ borderRadius: '8px' }} />
          <Skeleton variant="rounded" height={280} sx={{ borderRadius: '8px' }} />
        </Box>
      </Stack>
    );
  }

  const role = user?.role ?? 'USER_ROLE_UNSPECIFIED';
  if (role === 'USER_ROLE_CANDIDATE') {
    return <CandidateDashboard name={user?.name} />;
  }

  const actions = actionsFor(role);

  return (
    <Stack
      spacing={3}
      sx={{
        fontFamily: fonts.body,
        maxWidth: 1120,
        mx: 'auto',
        color: 'text.primary',
      }}
    >
      <Box
        sx={{
          display: 'grid',
          gap: { xs: 2.5, md: 3 },
          gridTemplateColumns: { xs: '1fr', md: 'minmax(0, 1.4fr) minmax(320px, 0.6fr)' },
          alignItems: 'stretch',
        }}
      >
        <Box
          sx={{
            position: 'relative',
            overflow: 'hidden',
            borderRadius: '8px',
            border: '1px solid',
            borderColor: 'divider',
            bgcolor: 'background.paper',
            minHeight: { xs: 360, md: 430 },
            p: { xs: 3, md: 4 },
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'space-between',
            boxShadow: '0 24px 80px rgba(0, 0, 0, 0.22)',
            '&::before': {
              content: '""',
              position: 'absolute',
              inset: 0,
              background:
                'linear-gradient(135deg, rgba(77, 155, 255, 0.16), transparent 42%), linear-gradient(180deg, rgba(45, 212, 191, 0.08), transparent 58%)',
              pointerEvents: 'none',
            },
          }}
        >
          <Stack spacing={2.5} sx={{ position: 'relative', maxWidth: 720 }}>
            <Stack direction="row" spacing={1} useFlexGap sx={{ flexWrap: 'wrap', alignItems: 'center' }}>
              <Chip
                label={ROLE_LABEL[role]}
                color="primary"
                size="small"
                sx={{
                  borderRadius: '999px',
                  fontFamily: fonts.body,
                  fontWeight: 700,
                }}
              />
              <Chip
                icon={<ShieldOutlined />}
                label="Evidence-first"
                variant="outlined"
                size="small"
                sx={{ borderRadius: '999px', fontFamily: fonts.body }}
              />
            </Stack>

            <Box>
              <Typography
                component="h1"
                sx={{
                  fontFamily: fonts.body,
                  fontSize: { xs: 42, sm: 54, md: 68 },
                  lineHeight: 0.96,
                  fontWeight: 800,
                  letterSpacing: 0,
                  maxWidth: 760,
                }}
              >
                Welcome{user ? `, ${user.name}` : ''}.
              </Typography>
              <Typography
                sx={{
                  mt: 2,
                  fontFamily: fonts.body,
                  color: 'text.secondary',
                  fontSize: { xs: 17, md: 20 },
                  lineHeight: 1.55,
                  maxWidth: 660,
                }}
              >
                {NEXT_BY_ROLE[role]} Caliber keeps the work auditable, human-reviewed, and grounded in candidate evidence.
              </Typography>
            </Box>
          </Stack>

          <Stack
            direction={{ xs: 'column', sm: 'row' }}
            spacing={1.25}
            useFlexGap
            sx={{ position: 'relative', flexWrap: 'wrap', mt: 4 }}
          >
            {actions.map((action, index) => (
              <Button
                key={action.label}
                component={Link}
                to={action.to}
                variant={index === 0 ? 'contained' : 'outlined'}
                endIcon={index === 0 ? <ArrowForwardRounded /> : undefined}
                sx={{
                  minHeight: 48,
                  borderRadius: '8px',
                  px: 2.4,
                  fontFamily: fonts.body,
                  justifyContent: 'center',
                }}
              >
                {HERO_LABELS[role][index] ?? action.title}
              </Button>
            ))}
          </Stack>
        </Box>

        <Card
          variant="outlined"
          sx={{
            borderRadius: '8px',
            bgcolor: 'background.paper',
            overflow: 'hidden',
            boxShadow: '0 24px 70px rgba(0, 0, 0, 0.18)',
          }}
        >
          <CardContent sx={{ p: { xs: 2.5, md: 3 }, height: '100%' }}>
            <Stack spacing={2.25} sx={{ height: '100%' }}>
              <Box>
                <Typography
                  component="h2"
                  sx={{
                    fontFamily: fonts.body,
                    fontSize: 13,
                    fontWeight: 800,
                    letterSpacing: 0,
                    textTransform: 'uppercase',
                    color: 'primary.main',
                  }}
                >
                  Workspace status
                </Typography>
                <Typography sx={{ mt: 0.75, fontFamily: fonts.body, color: 'text.secondary', lineHeight: 1.55 }}>
                  Your home base for role setup, shortlisting evidence, and candidate-facing work.
                </Typography>
              </Box>

              <Stack spacing={1.25}>
                {REVIEW_STEPS.map(({ label, body, Icon }) => (
                  <Box
                    key={label}
                    sx={{
                      display: 'grid',
                      gridTemplateColumns: '38px 1fr',
                      gap: 1.5,
                      alignItems: 'center',
                      p: 1.5,
                      borderRadius: '8px',
                      border: '1px solid',
                      borderColor: 'divider',
                      bgcolor: 'rgba(255,255,255,0.03)',
                    }}
                  >
                    <Box
                      sx={{
                        width: 38,
                        height: 38,
                        borderRadius: '8px',
                        display: 'grid',
                        placeItems: 'center',
                        color: 'primary.main',
                        bgcolor: 'rgba(77, 155, 255, 0.12)',
                      }}
                    >
                      <Icon fontSize="small" />
                    </Box>
                    <Box>
                      <Typography component="h3" sx={{ fontFamily: fonts.body, fontWeight: 800, lineHeight: 1.2 }}>
                        {label}
                      </Typography>
                      <Typography sx={{ mt: 0.25, fontFamily: fonts.body, color: 'text.secondary', fontSize: 14 }}>
                        {body}
                      </Typography>
                    </Box>
                  </Box>
                ))}
              </Stack>

              <Box
                sx={{
                  mt: 'auto',
                  p: 2,
                  borderRadius: '8px',
                  bgcolor: 'rgba(45, 212, 191, 0.1)',
                  border: '1px solid rgba(45, 212, 191, 0.24)',
                }}
              >
                <Typography sx={{ fontFamily: fonts.body, fontWeight: 800 }}>No black-box shortlist.</Typography>
                <Typography sx={{ mt: 0.5, fontFamily: fonts.body, color: 'text.secondary', fontSize: 14, lineHeight: 1.55 }}>
                  Every recommendation should point back to skills, role criteria, and reviewer judgment.
                </Typography>
              </Box>
            </Stack>
          </CardContent>
        </Card>
      </Box>

      <Box
        sx={{
          display: 'grid',
          gap: 2,
          gridTemplateColumns: { xs: '1fr', md: 'repeat(3, minmax(0, 1fr))' },
        }}
      >
        {actions.map(({ title, body, label, to, Icon, tone }) => (
          <Card
            key={title}
            variant="outlined"
            sx={{
              borderRadius: '8px',
              bgcolor: 'background.paper',
              minHeight: 260,
              transition: 'transform 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease',
              '&:hover': {
                transform: 'translateY(-3px)',
                borderColor: tone,
                boxShadow: `0 18px 50px ${tone}1F`,
              },
            }}
          >
            <CardContent sx={{ p: 2.5, height: '100%' }}>
              <Stack spacing={2} sx={{ height: '100%', alignItems: 'flex-start' }}>
                <Box
                  sx={{
                    width: 46,
                    height: 46,
                    borderRadius: '8px',
                    display: 'grid',
                    placeItems: 'center',
                    color: tone,
                    bgcolor: `${tone}1A`,
                  }}
                >
                  <Icon />
                </Box>
                <Box>
                  <Typography
                    component="h2"
                    sx={{ fontFamily: fonts.body, fontWeight: 800, fontSize: 21, letterSpacing: 0 }}
                  >
                    {title}
                  </Typography>
                  <Typography sx={{ mt: 1, fontFamily: fonts.body, color: 'text.secondary', lineHeight: 1.55 }}>
                    {body}
                  </Typography>
                </Box>
                <Button
                  component={Link}
                  to={to}
                  variant="text"
                  endIcon={<ArrowForwardRounded />}
                  sx={{
                    mt: 'auto',
                    px: 0,
                    minWidth: 0,
                    fontFamily: fonts.body,
                    color: tone,
                    '&:hover': { bgcolor: 'transparent', textDecoration: 'underline' },
                  }}
                >
                  {label === 'Describe a role' ||
                  label === 'Your roles' ||
                  label === 'Set up your passport' ||
                  label === 'Screening interview' ||
                  label === 'Run your agent'
                    ? `Open ${title.toLowerCase()}`
                    : label}
                </Button>
              </Stack>
            </CardContent>
          </Card>
        ))}
      </Box>

      <Card
        variant="outlined"
        sx={{
          borderRadius: '8px',
          bgcolor: 'background.paper',
          overflow: 'hidden',
        }}
      >
        <CardContent sx={{ p: { xs: 2.5, md: 3 } }}>
          <Box
            sx={{
              display: 'grid',
              gap: 2,
              gridTemplateColumns: { xs: '1fr', md: '1fr auto' },
              alignItems: 'center',
            }}
          >
            <Box>
              <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                <HubOutlined color="primary" fontSize="small" />
                <Typography
                  component="h2"
                  sx={{
                    fontFamily: fonts.body,
                    fontSize: 13,
                    fontWeight: 800,
                    letterSpacing: 0,
                    textTransform: 'uppercase',
                    color: 'primary.main',
                  }}
                >
                  Next step
                </Typography>
              </Stack>
              <Typography sx={{ mt: 1, fontFamily: fonts.body, fontSize: { xs: 20, md: 24 }, fontWeight: 800 }}>
                {NEXT_BY_ROLE[role]}
              </Typography>
              <Typography sx={{ mt: 0.75, fontFamily: fonts.body, color: 'text.secondary', maxWidth: 720, lineHeight: 1.55 }}>
                Use the fastest path below when you know what to do next, or open the wider workspace from the cards above.
              </Typography>
            </Box>
            {(role === 'USER_ROLE_EMPLOYER' || role === 'USER_ROLE_RECRUITER') && (
              <Stack direction="row" spacing={1} useFlexGap sx={{ flexWrap: 'wrap', justifyContent: { md: 'flex-end' } }}>
                <Button component={Link} to="/roles/new" variant="contained">
                  Describe a role
                </Button>
                <Button component={Link} to="/roles" variant="outlined">
                  Your roles
                </Button>
              </Stack>
            )}
          </Box>
        </CardContent>
      </Card>
    </Stack>
  );
}
