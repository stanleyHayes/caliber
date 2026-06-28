import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { InterviewReportCard } from '../../api/types';
import { ReportCardView } from './ReportCardView';

const report: InterviewReportCard = {
  interviewId: 'iv1',
  roleId: 'r1',
  candidateId: 'c1',
  verdict: 'INTERVIEW_VERDICT_ADVANCE',
  confidence: 'CONFIDENCE_HIGH',
  scores: [{ competency: 'Go', score: 4.5, evidence: 'built a payments service' }],
  recommendedNextStep: 'Schedule an onsite.',
};

describe('ReportCardView', () => {
  it('renders the verdict, confidence, evidence-backed scores, and next step', () => {
    render(<ReportCardView report={report} />);
    expect(screen.getByText('Advance')).toBeInTheDocument();
    expect(screen.getByText('High confidence')).toBeInTheDocument();
    expect(screen.getByText('Go')).toBeInTheDocument();
    expect(screen.getByText('“built a payments service”')).toBeInTheDocument();
    expect(screen.getByText('Schedule an onsite.')).toBeInTheDocument();
  });

  it('handles an empty score set gracefully', () => {
    render(<ReportCardView report={{ ...report, scores: [] }} />);
    expect(screen.getByText('No per-competency scores were produced.')).toBeInTheDocument();
  });
});
