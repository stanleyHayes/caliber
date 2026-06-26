# Fairness & bias-safety methodology (CAL-085)

Project Caliber makes employment-relevant recommendations, so fairness is a
correctness property, not a feature. This document records how the platform
keeps match scores independent of protected attributes, and how that property
is tested. Baseline: Ghana Data Protection Act, 2012; human-in-the-loop on every
consequential decision.

## Protected attributes

The canonical, immutable set lives in `internal/domain/matching/bias.go`
(`ProtectedAttributes()`):

```
age · disability · ethnicity · gender · marital_status · nationality · religion
```

These must never influence scoring, ranking, gating, or recall.

## Defense in depth

Fairness is enforced at four layers, weakest-link-first:

1. **Structural — not modelled.** The candidate and profile domain entities
   (`talent.Candidate`, `talent.TalentProfile`) have no protected-attribute
   fields. An attribute that does not exist in the model cannot become a
   ranking signal. This is the strongest guarantee and the reason the
   metamorphic test below holds by construction.

2. **Signal validation — `EnsureBiasSafe`.** Before any scoring, the shortlister
   validates that every ranking/gating signal key (the rubric competency names)
   is bias-safe. A protected attribute among the signals aborts the run with a
   `kernel.Invalid` error (`Shortlister.GenerateShortlist`). The check is
   case-insensitive and whitespace-trimmed.

3. **Input minimisation — prompt construction.** The text sent to the scorer
   (`scoringPrompt`) and to the embedder (`roleText`) is built **only** from the
   role and from the candidate's *competencies* (name, level, evidence). Summary,
   location, intake, salary, and identity never enter the model's view. CV text
   that reaches the model is additionally fenced and sanitised at the LLM
   boundary (see `internal/domain/guard`, CAL-119).

4. **Model instruction.** The scoring system prompt explicitly states: *score
   only on the rubric competencies and the candidate's evidence — never on
   protected attributes.* This is defence-in-depth for any protected term that a
   candidate's own evidence quote might contain.

## Hard filters are logistical, never protected

The shortlist hard filters gate on `location` (work logistics), `salary_floor`,
and `must_have_competency`. The `location` gate is deliberately distinct from the
protected attribute `nationality`: it matches work-location tokens and a remote
role bypasses it entirely. Every gate excludes only on **positive evidence** of a
conflict — unknown or unscored data never excludes — which upholds the
no-fabrication invariant and favours human review over false rejection.

## How it is tested

`internal/app/matching/fairness_test.go` (metamorphic, runs in CI):

- **`TestScoringIsInvariantToProtectedAttributes`** — two candidates with
  identical competencies are scored through the real pipeline; one carries a
  summary saturated with protected attributes. The test captures the exact text
  sent to the scorer and embedder and asserts (a) the two scoring prompts are
  byte-identical and (b) no protected-attribute term appears in either model
  input. Perturbing only the protected dimensions leaves the model's view
  unchanged.
- **`TestBiasedRubricIsRejectedBeforeScoring`** — a rubric naming a protected
  attribute aborts the run before any embedding, recall, or scoring.
- **`TestHardFilterGatesAreBiasSafe`** — the gate identifiers themselves pass
  `EnsureBiasSafe`.

`internal/domain/matching/bias_test.go` covers the protected-attribute set and
`EnsureBiasSafe` (exact, case, whitespace, and lookalike cases).

## Explainability & contest

Every `Match` carries a per-competency breakdown, narrative rationale, and
watch-outs — all competency-derived, none protected. Candidates can view and
contest their assessment (CAL-083). Because the breakdown is built only from
rubric competencies, the surfaced explanation cannot reference a protected
attribute.

## Limitations & future work

- Proxy variables (e.g. a competency that correlates with a protected class) are
  not yet audited; a disparate-impact analysis over real outcome data is future
  work once the POC has data.
- Evidence quotes are passed verbatim to the scorer; the model-instruction layer
  (not redaction) handles protected terms a candidate volunteers, to avoid
  distorting their own words. Redaction-for-scoring is a considered enhancement.
