import type { Confidence, Seniority } from '../api/types';

export const pct = (v: number) => `${Math.round(v * 100)}%`;

export const seniorityLabel: Record<Seniority, string> = {
  SENIORITY_UNSPECIFIED: 'Unspecified',
  SENIORITY_JUNIOR: 'Junior',
  SENIORITY_MID: 'Mid',
  SENIORITY_SENIOR: 'Senior',
  SENIORITY_LEAD: 'Lead',
};

export const confidenceLabel: Record<Confidence, string> = {
  CONFIDENCE_UNSPECIFIED: 'Unrated',
  CONFIDENCE_LOW: 'Low',
  CONFIDENCE_MEDIUM: 'Medium',
  CONFIDENCE_HIGH: 'High',
};

export const confidenceColor: Record<Confidence, 'default' | 'info' | 'warning' | 'success'> = {
  CONFIDENCE_UNSPECIFIED: 'default',
  CONFIDENCE_LOW: 'warning',
  CONFIDENCE_MEDIUM: 'info',
  CONFIDENCE_HIGH: 'success',
};

export const shortId = (id: string) => (id.length > 8 ? id.slice(0, 8) : id);
