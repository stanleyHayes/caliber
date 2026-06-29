package queue

import (
	"context"

	appqueue "github.com/xcreativs/caliber/internal/app/queue"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// Noop is a dev-path dispatcher that does not enqueue anything and returns an
// empty task ID. API handlers that receive a Noop (or nil) dispatcher fall back
// to running the work synchronously, preserving the original in-memory dev path.
type Noop struct{}

// NewNoop builds a no-op dispatcher.
func NewNoop() *Noop { return &Noop{} }

// DispatchCandidateAgentRun implements appqueue.TaskDispatcher.
func (n *Noop) DispatchCandidateAgentRun(context.Context, kernel.ID, ...appqueue.DispatchOption) (string, error) {
	return "", nil
}

// DispatchInterviewScoring implements appqueue.TaskDispatcher.
func (n *Noop) DispatchInterviewScoring(context.Context, kernel.ID, ...appqueue.DispatchOption) (string, error) {
	return "", nil
}

// DispatchBatchRematch implements appqueue.TaskDispatcher.
func (n *Noop) DispatchBatchRematch(context.Context, kernel.ID, ...appqueue.DispatchOption) (string, error) {
	return "", nil
}

// Close implements appqueue.TaskDispatcher.
func (n *Noop) Close() error { return nil }

// IsNoop reports whether d is the no-op dispatcher (or nil).
func IsNoop(d appqueue.TaskDispatcher) bool {
	if d == nil {
		return true
	}
	_, ok := d.(*Noop)
	return ok
}
