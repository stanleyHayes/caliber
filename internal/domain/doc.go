// Package domain is the pure core of the hexagon: entities, value objects,
// domain services, and PORTS (interfaces). It depends on NOTHING in
// internal/app, internal/adapters, or internal/platform — the dependency rule
// is enforced by depguard in .golangci.yml.
//
// Subpackages map to the spec's bounded contexts:
//
//	talent/         TalentProfile, Talent Passport, competencies
//	role/           Role, RoleSpec, Rubric
//	matching/       Match, scoring policy
//	interview/      Interview state machine, InterviewTurn, report card
//	candidateagent/ Candidate agent policy + no-fabrication invariant
//	identity/       User, roles, auth domain rules
//	audit/          AuditLog
package domain
