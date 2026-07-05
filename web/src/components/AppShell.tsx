import BadgeOutlined from '@mui/icons-material/BadgeOutlined';
import DashboardRounded from '@mui/icons-material/DashboardRounded';
import DonutSmallOutlined from '@mui/icons-material/DonutSmallOutlined';
import LogoutRounded from '@mui/icons-material/LogoutRounded';
import RadarRounded from '@mui/icons-material/RadarRounded';
import LoginRounded from '@mui/icons-material/LoginRounded';
import RocketLaunchOutlined from '@mui/icons-material/RocketLaunchOutlined';
import WorkOutlineRounded from '@mui/icons-material/WorkOutlineRounded';
import {
  AppBar,
  Avatar,
  Box,
  Button,
  Container,
  Divider,
  IconButton,
  ListItemIcon,
  Menu,
  MenuItem,
  Toolbar,
  Tooltip,
  Typography,
} from '@mui/material';
import type { SvgIconProps } from '@mui/material/SvgIcon';
import { AnimatePresence, motion } from 'motion/react';
import type { ComponentType } from 'react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, Outlet, useLocation } from 'react-router-dom';

import type { UserRole } from '../api/types';
import { canViewDashboard } from '../lib/permissions';
import { useLogout } from '../query/auth';
import { useAuthStore } from '../stores/auth';
import { fonts } from '../theme/tokens';
import { ModeToggle } from './ModeToggle';

const ROLE_LABEL: Record<UserRole, string> = {
  USER_ROLE_UNSPECIFIED: 'Member',
  USER_ROLE_EMPLOYER: 'Employer',
  USER_ROLE_RECRUITER: 'Recruiter',
  USER_ROLE_CANDIDATE: 'Candidate',
};

function initials(name: string) {
  const value = name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? '')
    .join('');

  return value || 'C';
}

function AccountMenuRow({
  Icon,
  title,
  description,
}: {
  Icon: ComponentType<SvgIconProps>;
  title: string;
  description: string;
}) {
  return (
    <>
      <ListItemIcon sx={{ minWidth: 32, mt: 0.25 }}>
        <Icon fontSize="small" />
      </ListItemIcon>
      <Box sx={{ minWidth: 0 }}>
        <Typography sx={{ fontFamily: fonts.body, fontWeight: 800, fontSize: 14, lineHeight: 1.25 }}>
          {title}
        </Typography>
        <Typography sx={{ fontFamily: fonts.body, color: 'text.secondary', fontSize: 12.5, lineHeight: 1.35 }}>
          {description}
        </Typography>
      </Box>
    </>
  );
}

export function AppShell() {
  const { t } = useTranslation();
  const location = useLocation();
  const user = useAuthStore((s) => s.user);
  const accessToken = useAuthStore((s) => s.accessToken);
  const logout = useLogout();
  const [accountAnchor, setAccountAnchor] = useState<HTMLElement | null>(null);
  const accountMenuOpen = Boolean(accountAnchor);
  const closeAccountMenu = () => setAccountAnchor(null);
  const showRadar = Boolean(accessToken && canViewDashboard(user?.role));
  const roleDestination =
    user?.role === 'USER_ROLE_CANDIDATE'
      ? {
          to: '/profile',
          Icon: BadgeOutlined,
          title: t('nav.accountProfileTitle'),
          description: t('nav.accountProfileBody'),
        }
      : {
          to: '/roles',
          Icon: WorkOutlineRounded,
          title: t('nav.accountRolesTitle'),
          description: t('nav.accountRolesBody'),
        };

  return (
    <Box data-testid="app-shell" sx={{ minHeight: '100dvh', bgcolor: 'background.default' }}>
      <Box
        component="a"
        href="#main-content"
        sx={{
          position: 'absolute',
          left: 8,
          top: -48,
          px: 2,
          py: 1,
          bgcolor: 'background.paper',
          color: 'text.primary',
          border: 1,
          borderColor: 'divider',
          borderRadius: 1,
          zIndex: (t) => t.zIndex.tooltip + 1,
          transition: 'top 0.15s ease',
          '&:focus-visible': { top: 8 },
        }}
      >
        {t('nav.skipToMain')}
      </Box>
      <AppBar
        component="header"
        position="sticky"
        color="default"
        elevation={0}
        sx={{
          borderBottom: 1,
          borderColor: 'rgba(17, 20, 24, 0.08)',
          bgcolor: 'rgba(255,255,255,0.78)',
          backdropFilter: 'blur(18px)',
          '.dark &': { bgcolor: 'rgba(11, 14, 17, 0.74)', borderColor: 'rgba(255,255,255,0.1)' },
        }}
      >
        <Toolbar sx={{ gap: 2, minHeight: { xs: 64, md: 72 } }}>
          <Box
            component={Link}
            to={accessToken ? '/app' : '/'}
            sx={{
              flexGrow: 1,
              minWidth: 0,
              display: 'inline-flex',
              alignItems: 'center',
              gap: 1.25,
              textDecoration: 'none',
              color: 'text.primary',
            }}
          >
            <Box
              aria-hidden
              sx={{
                display: 'grid',
                placeItems: 'center',
                width: 34,
                height: 34,
                borderRadius: 1.5,
                color: '#fff',
                bgcolor: 'primary.main',
                boxShadow: '0 10px 26px rgba(0, 102, 204, 0.28)',
              }}
            >
              <DonutSmallOutlined fontSize="small" />
            </Box>
            <Typography
              variant="h6"
              component="span"
              sx={{
                fontFamily: fonts.body,
                fontWeight: 800,
                whiteSpace: 'nowrap',
              }}
            >
              {t('brand.name')}
            </Typography>
          </Box>
          <Box
            component="nav"
            aria-label="Primary"
            sx={{
              display: 'flex',
              alignItems: 'center',
              flexWrap: 'wrap',
              justifyContent: 'flex-end',
              rowGap: 0.5,
              columnGap: 1,
            }}
          >
            {showRadar && (
              <Button component={Link} to="/radar" color="inherit" size="small">
                {t('nav.radar')}
              </Button>
            )}
            <ModeToggle />
            {accessToken ? (
              <>
                <Tooltip title={t('nav.accountMenu')}>
                  <IconButton
                    onClick={(event) => setAccountAnchor(event.currentTarget)}
                    size="small"
                    aria-label={t('nav.accountMenu')}
                    aria-controls={accountMenuOpen ? 'account-menu' : undefined}
                    aria-haspopup="menu"
                    aria-expanded={accountMenuOpen ? 'true' : undefined}
                    sx={{
                      p: 0.4,
                      border: '1px solid',
                      borderColor: 'divider',
                      bgcolor: 'background.paper',
                      '&:hover': { bgcolor: 'action.hover' },
                    }}
                  >
                    <Avatar
                      sx={{
                        width: 32,
                        height: 32,
                        fontFamily: fonts.body,
                        fontWeight: 800,
                        fontSize: 13,
                        bgcolor: 'primary.main',
                        color: 'primary.contrastText',
                      }}
                    >
                      {initials(user?.name ?? t('brand.name'))}
                    </Avatar>
                  </IconButton>
                </Tooltip>
                <Menu
                  id="account-menu"
                  anchorEl={accountAnchor}
                  open={accountMenuOpen}
                  onClose={closeAccountMenu}
                  anchorOrigin={{ horizontal: 'right', vertical: 'bottom' }}
                  transformOrigin={{ horizontal: 'right', vertical: 'top' }}
                  slotProps={{
                    paper: {
                      sx: {
                        mt: 1.25,
                        width: 320,
                        maxWidth: 'calc(100vw - 24px)',
                        borderRadius: '8px',
                        border: '1px solid',
                        borderColor: 'divider',
                        boxShadow: '0 24px 70px rgba(0,0,0,0.24)',
                      },
                    },
                    list: { 'aria-label': t('nav.accountMenu') },
                  }}
                >
                  <Box sx={{ px: 2, py: 1.5 }}>
                    <Typography sx={{ fontFamily: fonts.body, fontWeight: 850, fontSize: 14 }}>
                      {user?.name ?? t('brand.name')}
                    </Typography>
                    {user?.email && (
                      <Typography sx={{ mt: 0.25, fontFamily: fonts.body, color: 'text.secondary', fontSize: 12.5 }} noWrap>
                        {user.email}
                      </Typography>
                    )}
                    <Typography sx={{ mt: 0.5, fontFamily: fonts.body, color: 'text.secondary', fontSize: 12.5 }}>
                      {user ? ROLE_LABEL[user.role] : ROLE_LABEL.USER_ROLE_UNSPECIFIED}
                    </Typography>
                  </Box>
                  <Divider />
                  <MenuItem component={Link} to="/app" onClick={closeAccountMenu} sx={{ alignItems: 'flex-start', gap: 1.5, p: 1.5 }}>
                    <AccountMenuRow Icon={DashboardRounded} title={t('nav.accountDashboardTitle')} description={t('nav.accountDashboardBody')} />
                  </MenuItem>
                  <MenuItem
                    component={Link}
                    to={roleDestination.to}
                    onClick={closeAccountMenu}
                    sx={{ alignItems: 'flex-start', gap: 1.5, p: 1.5 }}
                  >
                    <AccountMenuRow Icon={roleDestination.Icon} title={roleDestination.title} description={roleDestination.description} />
                  </MenuItem>
                  {showRadar && (
                    <MenuItem component={Link} to="/radar" onClick={closeAccountMenu} sx={{ alignItems: 'flex-start', gap: 1.5, p: 1.5 }}>
                      <AccountMenuRow Icon={RadarRounded} title={t('nav.accountRadarTitle')} description={t('nav.accountRadarBody')} />
                    </MenuItem>
                  )}
                  <Divider />
                  <MenuItem
                    disabled={logout.isPending}
                    onClick={() => {
                      closeAccountMenu();
                      logout.mutate();
                    }}
                    sx={{ alignItems: 'flex-start', gap: 1.5, p: 1.5 }}
                  >
                    <AccountMenuRow Icon={LogoutRounded} title={t('nav.accountSignOutTitle')} description={t('nav.accountSignOutBody')} />
                  </MenuItem>
                </Menu>
              </>
            ) : (
              <>
                <Button component={Link} to="/login" color="inherit" size="small" startIcon={<LoginRounded />}>
                  {t('nav.signIn')}
                </Button>
                <Button component={Link} to="/register" variant="contained" size="small" startIcon={<RocketLaunchOutlined />}>
                  {t('nav.getStarted')}
                </Button>
              </>
            )}
          </Box>
        </Toolbar>
      </AppBar>
      <Container component="main" id="main-content" maxWidth="lg" sx={{ py: { xs: 3, md: 5 } }}>
        <AnimatePresence mode="wait">
          <motion.div
            key={location.pathname}
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -8 }}
            transition={{ duration: 0.2, ease: 'easeOut' }}
          >
            <Outlet />
          </motion.div>
        </AnimatePresence>
      </Container>
    </Box>
  );
}
