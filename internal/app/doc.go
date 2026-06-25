// Package app holds application services / use-cases. Use-cases orchestrate the
// domain through its ports; they depend on internal/domain only (never on
// concrete adapters). Inbound adapters (gRPC) call into app; app calls outbound
// adapters via domain ports.
package app
