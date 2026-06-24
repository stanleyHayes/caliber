// Package adapters contains the driving (inbound) and driven (outbound) edges
// of the hexagon:
//
//	inbound/grpc   gRPC service handlers + grpc-gateway (REST)
//	inbound/http   chi: gateway mux mount, health, auth middleware, interview stream
//	inbound/jobs   Asynq task handlers
//	outbound/postgres   sqlc + pgx repository adapters (implement domain ports)
//	outbound/llm        Anthropic Claude gateway (implements LLMClient)
//	outbound/embeddings OpenAI embedder (implements Embedder)
//	outbound/queue      Asynq enqueuer (implements TaskDispatcher)
//	outbound/auth       JWT issuer/verifier, Argon2id hasher
package adapters
