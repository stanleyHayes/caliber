import { describe, expect, it } from 'vitest';

import { confidenceLabel, passportLabel, pct, shortId, verdictColor } from './format';

describe('format helpers', () => {
  it('pct rounds a 0..1 ratio to a whole percentage', () => {
    expect(pct(0.873)).toBe('87%');
    expect(pct(0)).toBe('0%');
    expect(pct(1)).toBe('100%');
  });

  it('shortId truncates long ids to 8 chars and leaves short ones alone', () => {
    expect(shortId('0123456789')).toBe('01234567');
    expect(shortId('abc')).toBe('abc');
  });

  it('maps enum values to human labels', () => {
    expect(confidenceLabel.CONFIDENCE_HIGH).toBe('High');
    expect(passportLabel.PASSPORT_STATUS_SCREENED).toBe('Screened');
  });

  it('maps verdicts to MUI chip colors', () => {
    expect(verdictColor.INTERVIEW_VERDICT_ADVANCE).toBe('success');
    expect(verdictColor.INTERVIEW_VERDICT_DECLINE).toBe('error');
  });
});
