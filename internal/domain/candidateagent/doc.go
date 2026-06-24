// Package candidateagent is the pure domain for the autonomous candidate agent
// (Flow C). It owns the lifecycle of agent- and manually-authored job
// applications.
//
// HARD INVARIANT — no fabrication: an agent-authored application MUST reference
// a verified profile and may draw only on verified content. The agent never
// invents experience, credentials, or summaries that are not grounded in a
// verified profile. EnsureFromProfile and NewAgentApplication encode this rule
// at construction time.
//
// This package is pure: it imports only the shared kernel and the Go standard
// library. It references sibling entities (roles, candidates, profiles) by
// kernel.ID and never imports sibling domain packages.
package candidateagent
