// DTOs mirror the grpc-gateway JSON (camelCase) of caliber.v1 IdentityService.

export type UserRole =
  | 'USER_ROLE_UNSPECIFIED'
  | 'USER_ROLE_EMPLOYER'
  | 'USER_ROLE_RECRUITER'
  | 'USER_ROLE_CANDIDATE';

export interface User {
  id: string;
  email: string;
  role: UserRole;
  name: string;
  createdAt: string;
}

export interface TokenPair {
  accessToken: string;
  refreshToken: string;
  accessExpiresIn: number;
}

export interface AuthResponse {
  user: User;
  tokens: TokenPair;
}

export interface RefreshResponse {
  tokens: TokenPair;
}

export interface MeResponse {
  user: User;
}

export interface LoginInput {
  email: string;
  password: string;
}

export interface RegisterInput {
  name: string;
  email: string;
  password: string;
  role: UserRole;
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

// ----- Flow A: Role Spec/Rubric + Matching (caliber.v1 Role/Matching) -----

export type Seniority =
  | 'SENIORITY_UNSPECIFIED'
  | 'SENIORITY_JUNIOR'
  | 'SENIORITY_MID'
  | 'SENIORITY_SENIOR'
  | 'SENIORITY_LEAD';

export type RoleStatus =
  | 'ROLE_STATUS_UNSPECIFIED'
  | 'ROLE_STATUS_DRAFT'
  | 'ROLE_STATUS_OPEN'
  | 'ROLE_STATUS_CLOSED';

export type Confidence =
  | 'CONFIDENCE_UNSPECIFIED'
  | 'CONFIDENCE_LOW'
  | 'CONFIDENCE_MEDIUM'
  | 'CONFIDENCE_HIGH';

export interface SalaryBand {
  currency: string;
  low: number;
  high: number;
}

export interface Competency {
  name: string;
  weight: number;
  mustHave: boolean;
}

export interface Rubric {
  competencies: Competency[];
}

export interface RoleSpec {
  title: string;
  location: string;
  seniority: Seniority;
  availability: string;
  responsibilities: string[];
  mustHaves: string[];
  niceToHaves: string[];
  salaryBand: SalaryBand;
}

export interface Role {
  id: string;
  employerId: string;
  title: string;
  status: RoleStatus;
  spec: RoleSpec;
  rubric: Rubric;
  createdAt: string;
}

export interface GenerateRoleResponse {
  role: Role;
  availableMatches: number;
}

export interface MatchBreakdownItem {
  competency: string;
  score: number; // 0..5
  evidence: string;
}

export interface Match {
  id: string;
  roleId: string;
  candidateId: string;
  overallScore: number; // 0..1
  confidence: Confidence;
  breakdown: MatchBreakdownItem[];
  rationale: string;
  watchOuts: string[];
  thinEvidence: boolean;
}

export interface CandidateExclusion {
  candidateId: string;
  gate: string;
  reason: string;
}

export interface Shortlist {
  matches: Match[];
  poolDepth: number;
  exclusions: CandidateExclusion[];
}

export interface ShortlistResponse {
  shortlist: Shortlist;
}

// ----- Flow B: AI screening interview (caliber.v1 InterviewService) -----

export type InterviewVerdict =
  | 'INTERVIEW_VERDICT_UNSPECIFIED'
  | 'INTERVIEW_VERDICT_ADVANCE'
  | 'INTERVIEW_VERDICT_HOLD'
  | 'INTERVIEW_VERDICT_DECLINE';

export interface InterviewQuestion {
  interviewId: string;
  ordinal: number;
  text: string;
  competencyTag: string;
}

export interface InterviewStatusEvent {
  state: string;
  message: string;
}

export interface InterviewCompetencyScore {
  competency: string;
  score: number; // 0..5
  evidence: string;
}

export interface InterviewReportCard {
  interviewId: string;
  roleId: string;
  candidateId: string;
  verdict: InterviewVerdict;
  confidence: Confidence;
  scores: InterviewCompetencyScore[];
  recommendedNextStep: string;
}

// One server-stream event (the StartInterviewResponse oneof, camelCase).
export interface InterviewEvent {
  status?: InterviewStatusEvent;
  question?: InterviewQuestion;
  reportCard?: InterviewReportCard;
}

// ----- Flow C: candidate autonomous agent (caliber.v1 CandidateAgentService) -----

export type ApplicationSource =
  | 'APPLICATION_SOURCE_UNSPECIFIED'
  | 'APPLICATION_SOURCE_MANUAL'
  | 'APPLICATION_SOURCE_AGENT';

export type ApplicationStatus =
  | 'APPLICATION_STATUS_UNSPECIFIED'
  | 'APPLICATION_STATUS_DRAFTED'
  | 'APPLICATION_STATUS_SUBMITTED'
  | 'APPLICATION_STATUS_SCREENING'
  | 'APPLICATION_STATUS_SCREENED';

export interface Application {
  id: string;
  roleId: string;
  candidateId: string;
  source: ApplicationSource;
  tailoredSummary: string;
  status: ApplicationStatus;
}

export interface WakeUpView {
  newMatches: number;
  applicationsSubmitted: number;
  screeningsCompleted: number;
  employersInterested: number;
  highlights: string[];
}

export interface TimeAdvanceResponse {
  wakeUp: WakeUpView;
}

export interface ListApplicationsResponse {
  applications: Application[];
}
