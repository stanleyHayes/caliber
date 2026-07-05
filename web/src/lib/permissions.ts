import type { UserRole } from '../api/types';

const reviewerRoles = new Set<UserRole>(['USER_ROLE_EMPLOYER', 'USER_ROLE_RECRUITER']);

export function canManageRoles(role: UserRole | undefined): boolean {
  return Boolean(role && reviewerRoles.has(role));
}

export function canViewDashboard(role: UserRole | undefined): boolean {
  return canManageRoles(role);
}

export function canScreenSelf(role: UserRole | undefined): boolean {
  return role === 'USER_ROLE_CANDIDATE';
}

export function canRunAgent(role: UserRole | undefined): boolean {
  return role === 'USER_ROLE_CANDIDATE';
}

export function canManageProfile(role: UserRole | undefined): boolean {
  return role === 'USER_ROLE_CANDIDATE';
}
