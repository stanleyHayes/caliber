package candidateagent

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// ApplicationRepository is the persistence PORT for applications. Adapters
// implement it; the domain depends only on this interface.
type ApplicationRepository interface {
	// Create persists a new application.
	Create(ctx context.Context, app *Application) error
	// ByID loads an application by its identifier, returning a kernel.NotFound
	// error when no application exists with that id.
	ByID(ctx context.Context, id kernel.ID) (*Application, error)
	// Update persists changes to an existing application.
	Update(ctx context.Context, app *Application) error
	// ByCandidate lists applications for a candidate, newest first, paginated.
	// It returns the page of applications and the total count across all pages.
	ByCandidate(ctx context.Context, candidateID kernel.ID, page kernel.Page) ([]*Application, int64, error)
}

//go:generate mockgen -source=port.go -destination=../../mocks/candidateagent.go -package=mocks
