package matching

import (
	"errors"
	"testing"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func TestMatchBreakdownItem_Validate(t *testing.T) {
	tests := []struct {
		name    string
		item    MatchBreakdownItem
		wantErr bool
	}{
		{"valid", MatchBreakdownItem{Competency: "Go", Score: 4.5, Evidence: "shipped"}, false},
		{"valid score zero", MatchBreakdownItem{Competency: "Go", Score: 0}, false},
		{"valid score five", MatchBreakdownItem{Competency: "Go", Score: 5}, false},
		{"empty competency", MatchBreakdownItem{Competency: "", Score: 3}, true},
		{"whitespace competency", MatchBreakdownItem{Competency: "   ", Score: 3}, true},
		{"score below range", MatchBreakdownItem{Competency: "Go", Score: -0.1}, true},
		{"score above range", MatchBreakdownItem{Competency: "Go", Score: 5.1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.item.Validate()
			if tt.wantErr != (err != nil) {
				t.Fatalf("Validate() err=%v, wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr && kernel.KindOf(err) != kernel.KindInvalid {
				t.Fatalf("expected KindInvalid, got %v", kernel.KindOf(err))
			}
		})
	}
}

func TestNewMatch(t *testing.T) {
	role := kernel.NewID()
	cand := kernel.NewID()
	good := []MatchBreakdownItem{{Competency: "Go", Score: 4}}

	tests := []struct {
		name      string
		roleID    kernel.ID
		candID    kernel.ID
		overall   float64
		conf      kernel.Confidence
		breakdown []MatchBreakdownItem
		wantErr   bool
	}{
		{"valid", role, cand, 0.8, kernel.ConfidenceHigh, good, false},
		{"valid overall zero", role, cand, 0, kernel.ConfidenceLow, nil, false},
		{"valid overall one", role, cand, 1, kernel.ConfidenceMedium, good, false},
		{"zero role", kernel.ID(""), cand, 0.5, kernel.ConfidenceHigh, good, true},
		{"zero candidate", role, kernel.ID(""), 0.5, kernel.ConfidenceHigh, good, true},
		{"overall below range", role, cand, -0.01, kernel.ConfidenceHigh, good, true},
		{"overall above range", role, cand, 1.01, kernel.ConfidenceHigh, good, true},
		{"invalid confidence", role, cand, 0.5, kernel.ConfidenceUnknown, good, true},
		{"invalid breakdown", role, cand, 0.5, kernel.ConfidenceHigh, []MatchBreakdownItem{{Competency: "", Score: 9}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMatch(tt.roleID, tt.candID, tt.overall, tt.conf, tt.breakdown, "rationale", []string{"caveat"}, true)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if kernel.KindOf(err) != kernel.KindInvalid {
					t.Fatalf("expected KindInvalid, got %v", kernel.KindOf(err))
				}
				if m != nil {
					t.Fatalf("expected nil match on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m.ID.IsZero() {
				t.Fatalf("expected generated ID")
			}
			if m.RoleID != tt.roleID || m.CandidateID != tt.candID {
				t.Fatalf("ids not set correctly")
			}
			if m.OverallScore != tt.overall || m.Confidence != tt.conf {
				t.Fatalf("fields not set correctly")
			}
			if !m.ThinEvidence {
				t.Fatalf("expected ThinEvidence true")
			}
			if m.Rationale != "rationale" {
				t.Fatalf("rationale not set")
			}
		})
	}
}

func TestNewMatch_DefensiveCopy(t *testing.T) {
	role := kernel.NewID()
	cand := kernel.NewID()
	breakdown := []MatchBreakdownItem{{Competency: "Go", Score: 4}}
	outs := []string{"watch"}

	m, err := NewMatch(role, cand, 0.5, kernel.ConfidenceHigh, breakdown, "r", outs, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mutate the inputs; the match must not be affected.
	breakdown[0].Competency = "MUTATED"
	outs[0] = "MUTATED"

	if m.Breakdown[0].Competency != "Go" {
		t.Fatalf("breakdown not defensively copied")
	}
	if m.WatchOuts[0] != "watch" {
		t.Fatalf("watchOuts not defensively copied")
	}
}

func TestNewMatch_UniqueIDs(t *testing.T) {
	role := kernel.NewID()
	cand := kernel.NewID()
	a, err := NewMatch(role, cand, 0.5, kernel.ConfidenceHigh, nil, "", nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := NewMatch(role, cand, 0.5, kernel.ConfidenceHigh, nil, "", nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ID == b.ID {
		t.Fatalf("expected unique IDs")
	}
}

func TestNewMatch_ErrorIsKernelError(t *testing.T) {
	_, err := NewMatch(kernel.ID(""), kernel.NewID(), 0.5, kernel.ConfidenceHigh, nil, "", nil, false)
	var ke *kernel.Error
	if !errors.As(err, &ke) {
		t.Fatalf("expected *kernel.Error, got %T", err)
	}
}
