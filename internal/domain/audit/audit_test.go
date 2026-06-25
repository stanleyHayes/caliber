package audit

import (
	"context"
	"testing"
	"time"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func TestActionConstants(t *testing.T) {
	cases := map[string]string{
		ActionApproveRejection: "approve_rejection",
		ActionOverrideScore:    "override_score",
		ActionAgentSubmit:      "agent_submit",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("action constant = %q, want %q", got, want)
		}
	}
}

func TestNewAuditEntry(t *testing.T) {
	actor := kernel.NewID()
	entityID := kernel.NewID()
	ts := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		actor       kernel.ID
		action      string
		entity      string
		entityID    kernel.ID
		before      string
		after       string
		wantErr     bool
		wantErrKind kernel.Kind
	}{
		{
			name:     "valid full entry",
			actor:    actor,
			action:   ActionOverrideScore,
			entity:   "Candidate",
			entityID: entityID,
			before:   `{"score":1}`,
			after:    `{"score":5}`,
		},
		{
			name:     "valid with zero entityID and empty snapshots",
			actor:    actor,
			action:   ActionAgentSubmit,
			entity:   "Role",
			entityID: kernel.ID(""),
		},
		{
			name:        "zero actor",
			actor:       kernel.ID(""),
			action:      ActionApproveRejection,
			entity:      "Candidate",
			entityID:    entityID,
			wantErr:     true,
			wantErrKind: kernel.KindInvalid,
		},
		{
			name:        "empty action",
			actor:       actor,
			action:      "",
			entity:      "Candidate",
			entityID:    entityID,
			wantErr:     true,
			wantErrKind: kernel.KindInvalid,
		},
		{
			name:        "whitespace action",
			actor:       actor,
			action:      "   ",
			entity:      "Candidate",
			entityID:    entityID,
			wantErr:     true,
			wantErrKind: kernel.KindInvalid,
		},
		{
			name:        "empty entity",
			actor:       actor,
			action:      ActionOverrideScore,
			entity:      "",
			entityID:    entityID,
			wantErr:     true,
			wantErrKind: kernel.KindInvalid,
		},
		{
			name:        "whitespace entity",
			actor:       actor,
			action:      ActionOverrideScore,
			entity:      "\t",
			entityID:    entityID,
			wantErr:     true,
			wantErrKind: kernel.KindInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewAuditEntry(tt.actor, tt.action, tt.entity, tt.entityID, tt.before, tt.after, ts)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if kernel.KindOf(err) != tt.wantErrKind {
					t.Fatalf("error kind = %v, want %v", kernel.KindOf(err), tt.wantErrKind)
				}
				if got != nil {
					t.Fatalf("expected nil entry on error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID.IsZero() {
				t.Error("expected non-zero generated ID")
			}
			if got.ActorUserID != tt.actor {
				t.Errorf("ActorUserID = %v, want %v", got.ActorUserID, tt.actor)
			}
			if got.Action != tt.action {
				t.Errorf("Action = %q, want %q", got.Action, tt.action)
			}
			if got.Entity != tt.entity {
				t.Errorf("Entity = %q, want %q", got.Entity, tt.entity)
			}
			if got.EntityID != tt.entityID {
				t.Errorf("EntityID = %v, want %v", got.EntityID, tt.entityID)
			}
			if got.BeforeJSON != tt.before {
				t.Errorf("BeforeJSON = %q, want %q", got.BeforeJSON, tt.before)
			}
			if got.AfterJSON != tt.after {
				t.Errorf("AfterJSON = %q, want %q", got.AfterJSON, tt.after)
			}
			if !got.Timestamp.Equal(ts) {
				t.Errorf("Timestamp = %v, want %v", got.Timestamp, ts)
			}
		})
	}
}

func TestNewAuditEntryGeneratesUniqueIDs(t *testing.T) {
	actor := kernel.NewID()
	ts := time.Now()
	a, err := NewAuditEntry(actor, ActionAgentSubmit, "Role", kernel.NewID(), "", "", ts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := NewAuditEntry(actor, ActionAgentSubmit, "Role", kernel.NewID(), "", "", ts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ID == b.ID {
		t.Errorf("expected unique IDs, both = %v", a.ID)
	}
}

// stubRepo is an in-memory AuditRepository used to verify the port is
// implementable using only the domain surface.
type stubRepo struct {
	entries []*AuditEntry
}

func (s *stubRepo) Append(_ context.Context, entry *AuditEntry) error {
	s.entries = append(s.entries, entry)
	return nil
}

func (s *stubRepo) List(_ context.Context, entity string, entityID kernel.ID, page kernel.Page) ([]*AuditEntry, int64, error) {
	var matched []*AuditEntry
	for _, e := range s.entries {
		if e.Entity == entity && e.EntityID == entityID {
			matched = append(matched, e)
		}
	}
	total := int64(len(matched))
	start := min(page.Offset(), len(matched))
	end := min(start+page.Limit(), len(matched))
	return matched[start:end], total, nil
}

func TestAuditRepositoryPort(t *testing.T) {
	var repo AuditRepository = &stubRepo{}
	ctx := context.Background()
	actor := kernel.NewID()
	entityID := kernel.NewID()
	ts := time.Now()

	for range 3 {
		e, err := NewAuditEntry(actor, ActionOverrideScore, "Candidate", entityID, "", "", ts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := repo.Append(ctx, e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	// An entry for a different entity must not be returned.
	other, _ := NewAuditEntry(actor, ActionAgentSubmit, "Role", kernel.NewID(), "", "", ts)
	if err := repo.Append(ctx, other); err != nil {
		t.Fatalf("append other: %v", err)
	}

	list, total, err := repo.List(ctx, "Candidate", entityID, kernel.NewPage(1, 2))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(list) != 2 {
		t.Errorf("page len = %d, want 2", len(list))
	}

	list2, _, err := repo.List(ctx, "Candidate", entityID, kernel.NewPage(2, 2))
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if len(list2) != 1 {
		t.Errorf("page 2 len = %d, want 1", len(list2))
	}

	empty, total, err := repo.List(ctx, "Candidate", kernel.NewID(), kernel.NewPage(1, 10))
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if total != 0 || len(empty) != 0 {
		t.Errorf("expected empty result, got total=%d len=%d", total, len(empty))
	}
}
