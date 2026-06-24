package candidateagent

import (
	"errors"
	"testing"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func TestEnsureFromProfile(t *testing.T) {
	t.Run("zero profile is rejected", func(t *testing.T) {
		err := EnsureFromProfile(kernel.ID(""))
		if err == nil {
			t.Fatal("expected error for zero profile id")
		}
		if kernel.KindOf(err) != kernel.KindInvalid {
			t.Fatalf("kind = %v, want Invalid", kernel.KindOf(err))
		}
	})
	t.Run("non-zero profile is accepted", func(t *testing.T) {
		if err := EnsureFromProfile(kernel.NewID()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestNewAgentApplication(t *testing.T) {
	role := kernel.NewID()
	cand := kernel.NewID()
	prof := kernel.NewID()

	tests := []struct {
		name      string
		role      kernel.ID
		cand      kernel.ID
		prof      kernel.ID
		summary   string
		wantError bool
	}{
		{"valid", role, cand, prof, "tailored summary", false},
		{"zero role", kernel.ID(""), cand, prof, "summary", true},
		{"zero candidate", role, kernel.ID(""), prof, "summary", true},
		{"zero profile (no fabrication)", role, cand, kernel.ID(""), "summary", true},
		{"empty summary", role, cand, prof, "", true},
		{"blank summary", role, cand, prof, "   ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := NewAgentApplication(tt.role, tt.cand, tt.prof, tt.summary)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				if kernel.KindOf(err) != kernel.KindInvalid {
					t.Fatalf("kind = %v, want Invalid", kernel.KindOf(err))
				}
				if app != nil {
					t.Fatal("expected nil application on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if app.ID.IsZero() {
				t.Fatal("expected generated id")
			}
			if app.Source != SourceAgent {
				t.Fatalf("source = %v, want agent", app.Source)
			}
			if app.Status != StatusDrafted {
				t.Fatalf("status = %v, want drafted", app.Status)
			}
			if app.ProfileID != tt.prof {
				t.Fatalf("profile id = %v, want %v", app.ProfileID, tt.prof)
			}
			if app.RoleID != tt.role || app.CandidateID != tt.cand {
				t.Fatal("role/candidate ids not set")
			}
			if app.TailoredSummary != tt.summary {
				t.Fatalf("summary = %q, want %q", app.TailoredSummary, tt.summary)
			}
		})
	}
}

func TestNewManualApplication(t *testing.T) {
	role := kernel.NewID()
	cand := kernel.NewID()

	tests := []struct {
		name      string
		role      kernel.ID
		cand      kernel.ID
		summary   string
		wantError bool
	}{
		{"valid", role, cand, "summary", false},
		{"zero role", kernel.ID(""), cand, "summary", true},
		{"zero candidate", role, kernel.ID(""), "summary", true},
		{"blank summary", role, cand, "  ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := NewManualApplication(tt.role, tt.cand, tt.summary)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				if kernel.KindOf(err) != kernel.KindInvalid {
					t.Fatalf("kind = %v, want Invalid", kernel.KindOf(err))
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if app.Source != SourceManual {
				t.Fatalf("source = %v, want manual", app.Source)
			}
			if app.Status != StatusDrafted {
				t.Fatalf("status = %v, want drafted", app.Status)
			}
			if !app.ProfileID.IsZero() {
				t.Fatal("manual application should have zero profile id")
			}
		})
	}
}

func TestApplicationLifecycleHappyPath(t *testing.T) {
	app, err := NewManualApplication(kernel.NewID(), kernel.NewID(), "summary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := app.Submit(); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if app.Status != StatusSubmitted {
		t.Fatalf("status = %v, want submitted", app.Status)
	}
	if err := app.MarkScreening(); err != nil {
		t.Fatalf("MarkScreening: %v", err)
	}
	if app.Status != StatusScreening {
		t.Fatalf("status = %v, want screening", app.Status)
	}
	if err := app.MarkScreened(); err != nil {
		t.Fatalf("MarkScreened: %v", err)
	}
	if app.Status != StatusScreened {
		t.Fatalf("status = %v, want screened", app.Status)
	}
}

func TestApplicationInvalidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from ApplicationStatus
		do   func(*Application) error
	}{
		{"submit when submitted", StatusSubmitted, (*Application).Submit},
		{"submit when screening", StatusScreening, (*Application).Submit},
		{"submit when screened", StatusScreened, (*Application).Submit},
		{"screening when drafted", StatusDrafted, (*Application).MarkScreening},
		{"screening when screening", StatusScreening, (*Application).MarkScreening},
		{"screening when screened", StatusScreened, (*Application).MarkScreening},
		{"screened when drafted", StatusDrafted, (*Application).MarkScreened},
		{"screened when submitted", StatusSubmitted, (*Application).MarkScreened},
		{"screened when screened", StatusScreened, (*Application).MarkScreened},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &Application{Status: tt.from}
			err := tt.do(app)
			if err == nil {
				t.Fatal("expected invalid transition error")
			}
			if kernel.KindOf(err) != kernel.KindInvalid {
				t.Fatalf("kind = %v, want Invalid", kernel.KindOf(err))
			}
			if app.Status != tt.from {
				t.Fatalf("status mutated to %v on failed transition", app.Status)
			}
		})
	}
}

func TestEnsureFromProfileErrorIsKernelError(t *testing.T) {
	err := EnsureFromProfile(kernel.ID(""))
	var ke *kernel.Error
	if !errors.As(err, &ke) {
		t.Fatalf("expected *kernel.Error, got %T", err)
	}
}
