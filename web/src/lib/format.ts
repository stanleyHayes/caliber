import type { ApplicationStatus, Confidence, InterviewVerdict, PassportStatus, Seniority } from '../api/types';

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

export const verdictLabel: Record<InterviewVerdict, string> = {
  INTERVIEW_VERDICT_UNSPECIFIED: 'Unscored',
  INTERVIEW_VERDICT_ADVANCE: 'Advance',
  INTERVIEW_VERDICT_HOLD: 'Hold',
  INTERVIEW_VERDICT_DECLINE: 'Decline',
};

export const verdictColor: Record<InterviewVerdict, 'default' | 'success' | 'warning' | 'error'> = {
  INTERVIEW_VERDICT_UNSPECIFIED: 'default',
  INTERVIEW_VERDICT_ADVANCE: 'success',
  INTERVIEW_VERDICT_HOLD: 'warning',
  INTERVIEW_VERDICT_DECLINE: 'error',
};

export const applicationStatusLabel: Record<ApplicationStatus, string> = {
  APPLICATION_STATUS_UNSPECIFIED: 'Unknown',
  APPLICATION_STATUS_DRAFTED: 'Drafted',
  APPLICATION_STATUS_SUBMITTED: 'Submitted',
  APPLICATION_STATUS_SCREENING: 'Screening',
  APPLICATION_STATUS_SCREENED: 'Screened',
};

export const applicationStatusColor: Record<ApplicationStatus, 'default' | 'info' | 'warning' | 'success'> = {
  APPLICATION_STATUS_UNSPECIFIED: 'default',
  APPLICATION_STATUS_DRAFTED: 'default',
  APPLICATION_STATUS_SUBMITTED: 'info',
  APPLICATION_STATUS_SCREENING: 'warning',
  APPLICATION_STATUS_SCREENED: 'success',
};

export const passportLabel: Record<PassportStatus, string> = {
  PASSPORT_STATUS_UNSPECIFIED: 'New',
  PASSPORT_STATUS_CV_ONLY: 'CV only',
  PASSPORT_STATUS_SCREENED: 'Screened',
  PASSPORT_STATUS_VERIFIED: 'Verified',
};

export const passportColor: Record<PassportStatus, 'default' | 'info' | 'warning' | 'success'> = {
  PASSPORT_STATUS_UNSPECIFIED: 'default',
  PASSPORT_STATUS_CV_ONLY: 'warning',
  PASSPORT_STATUS_SCREENED: 'info',
  PASSPORT_STATUS_VERIFIED: 'success',
};
