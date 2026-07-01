/**
 * Lighthouse CI configuration for Project Caliber (CAL-129).
 *
 * Runs against the built static site (dist/) on every PR and enforces the
 * performance budgets defined in lighthouse-budget.json (CAL-125). Category
 * scores are advisory warnings; the budget is the hard regression gate.
 */
module.exports = {
  ci: {
    collect: {
      staticDistDir: './dist',
      url: ['/', '/login', '/register', '/404'],
      numberOfRuns: 3,
    },
    assert: {
      assertions: {
        'categories:performance': ['warn', { minScore: 0.9 }],
        'categories:accessibility': ['warn', { minScore: 0.9 }],
      },
      budgetPath: './lighthouse-budget.json',
    },
    upload: {
      target: 'filesystem',
      outputDir: '.lighthouseci',
    },
  },
};
