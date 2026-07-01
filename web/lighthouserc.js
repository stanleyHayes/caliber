/**
 * Lighthouse CI configuration placeholder for CAL-129.
 *
 * The budget file (lighthouse-budget.json) is the active performance contract
 * used by this story (CAL-125). CAL-129 will wire @lhci/cli into CI to enforce
 * it on every PR.
 */
module.exports = {
  ci: {
    collect: {
      staticDistDir: './dist',
      url: ['http://localhost/', 'http://localhost/login', 'http://localhost/register'],
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
      target: 'temporary',
    },
  },
};
