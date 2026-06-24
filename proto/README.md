# Caliber API Contracts (protobuf)

Single source of truth for the Caliber API. gRPC services + `grpc-gateway` REST/JSON
+ generated Go stubs and TypeScript client types (see [buf.gen.yaml](../buf.gen.yaml)).

## Layout
```
proto/caliber/v1/
├── common.proto          # pagination, value objects, shared enums
├── identity.proto        # auth + the two POC roles (EPIC-02)
├── role.proto            # Role Spec + Rubric generation — Flow A.1 (EPIC-05)
├── talent.proto          # CV parsing + Talent Passport (EPIC-06)
├── matching.proto        # explainable ranked shortlist — Flow A (EPIC-07)
├── interview.proto       # streamed adaptive interview — Flow B (EPIC-09)
├── candidate_agent.proto # autonomous agent + time-advance — Flow C (EPIC-10)
├── dashboard.proto       # Talent Radar god-view (EPIC-11)
└── audit.proto           # audit trail (EPIC-12)
```

Message shapes for Role Spec/Rubric, Match, and Report Card are the **locked
contracts** from Appendix A of `Caliber_POC_Build_Spec.pdf` — do not rename fields.

## Conventions
- Package `caliber.v1`; buf `STANDARD` lint; versioned package suffix.
- Every collection RPC takes a `PageRequest` and returns a `PageResponse` (pagination is a platform standard).
- Enums use `*_UNSPECIFIED = 0` and value names prefixed by the enum name.
- REST mappings via `google.api.http` annotations; the interview uses a server stream.

## Workflow (once the Go module exists — CAL-001)
```bash
buf dep update     # resolve googleapis + grpc-gateway, write buf.lock
buf lint           # STANDARD lint
buf generate       # -> internal/gen/** (Go, gRPC, gateway) + docs/openapi
buf breaking --against '.git#branch=main'   # CI guard
```
Tooling tracks the **latest stable** versions (buf, protoc plugins via remote `buf.build/...`).
