package talent

import (
	"testing"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func TestPassportStatusValid(t *testing.T) {
	tests := []struct {
		name string
		s    PassportStatus
		want bool
	}{
		{"unset", PassportUnset, false},
		{"cv_only", PassportCVOnly, true},
		{"screened", PassportScreened, true},
		{"verified", PassportVerified, true},
		{"out_of_range", PassportStatus(99), false},
		{"negative", PassportStatus(-1), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Valid(); got != tt.want {
				t.Fatalf("Valid()=%v want %v", got, tt.want)
			}
		})
	}
}

func TestPassportStatusString(t *testing.T) {
	tests := []struct {
		s    PassportStatus
		want string
	}{
		{PassportUnset, "unset"},
		{PassportCVOnly, "cv_only"},
		{PassportScreened, "screened"},
		{PassportVerified, "verified"},
		{PassportStatus(42), "unset"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Fatalf("String()=%q want %q", got, tt.want)
			}
		})
	}
}

func TestProfileCompetencyValidate(t *testing.T) {
	tests := []struct {
		name    string
		c       ProfileCompetency
		wantErr bool
		kind    kernel.Kind
	}{
		{"ok", ProfileCompetency{Name: "Go", Level: 4}, false, kernel.KindInternal},
		{"ok_zero_level", ProfileCompetency{Name: "Go", Level: 0}, false, kernel.KindInternal},
		{"ok_max_level", ProfileCompetency{Name: "Go", Level: 5}, false, kernel.KindInternal},
		{"empty_name", ProfileCompetency{Name: "", Level: 3}, true, kernel.KindInvalid},
		{"blank_name", ProfileCompetency{Name: "   ", Level: 3}, true, kernel.KindInvalid},
		{"level_too_low", ProfileCompetency{Name: "Go", Level: -0.1}, true, kernel.KindInvalid},
		{"level_too_high", ProfileCompetency{Name: "Go", Level: 5.1}, true, kernel.KindInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.c.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if kernel.KindOf(err) != tt.kind {
					t.Fatalf("kind=%v want %v", kernel.KindOf(err), tt.kind)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCandidateIntakeValidate(t *testing.T) {
	tests := []struct {
		name    string
		i       CandidateIntake
		wantErr bool
	}{
		{"ok", CandidateIntake{SalaryFloor: 0}, false},
		{"ok_positive", CandidateIntake{SalaryFloor: 120000}, false},
		{"negative_floor", CandidateIntake{SalaryFloor: -1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.i.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if kernel.KindOf(err) != kernel.KindInvalid {
					t.Fatalf("kind=%v", kernel.KindOf(err))
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewCandidate(t *testing.T) {
	user := kernel.NewID()
	tests := []struct {
		name    string
		userID  kernel.ID
		intake  CandidateIntake
		wantErr bool
	}{
		{"ok", user, CandidateIntake{SalaryFloor: 50000}, false},
		{"zero_user", kernel.ID(""), CandidateIntake{}, true},
		{"bad_intake", user, CandidateIntake{SalaryFloor: -5}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewCandidate(tt.userID, "NYC", tt.intake)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if c != nil {
					t.Fatal("expected nil candidate")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.ID.IsZero() {
				t.Fatal("expected generated id")
			}
			if c.UserID != tt.userID {
				t.Fatalf("UserID=%v", c.UserID)
			}
			if c.Location != "NYC" {
				t.Fatalf("Location=%q", c.Location)
			}
		})
	}
}

func TestNewTalentProfile(t *testing.T) {
	cand := kernel.NewID()
	good := []ProfileCompetency{{Name: "Go", Level: 4}, {Name: "SQL", Level: 3}}
	bad := []ProfileCompetency{{Name: "", Level: 4}}

	t.Run("ok", func(t *testing.T) {
		p, err := NewTalentProfile(cand, "summary", good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.ID.IsZero() {
			t.Fatal("expected id")
		}
		if p.CandidateID != cand {
			t.Fatalf("CandidateID=%v", p.CandidateID)
		}
		if p.PassportStatus != PassportCVOnly {
			t.Fatalf("status=%v want cv_only", p.PassportStatus)
		}
		if len(p.Competencies) != 2 {
			t.Fatalf("competencies=%d", len(p.Competencies))
		}
		// mutating the source slice must not affect the profile (defensive copy).
		good[0].Name = "MUTATED"
		if p.Competencies[0].Name != "Go" {
			t.Fatalf("profile competency mutated: %q", p.Competencies[0].Name)
		}
	})

	t.Run("zero_candidate", func(t *testing.T) {
		p, err := NewTalentProfile(kernel.ID(""), "s", good)
		if err == nil || p != nil {
			t.Fatal("expected error and nil profile")
		}
		if kernel.KindOf(err) != kernel.KindInvalid {
			t.Fatalf("kind=%v", kernel.KindOf(err))
		}
	})

	t.Run("invalid_competency", func(t *testing.T) {
		p, err := NewTalentProfile(cand, "s", bad)
		if err == nil || p != nil {
			t.Fatal("expected error and nil profile")
		}
		if kernel.KindOf(err) != kernel.KindInvalid {
			t.Fatalf("kind=%v", kernel.KindOf(err))
		}
	})

	t.Run("nil_competencies", func(t *testing.T) {
		p, err := NewTalentProfile(cand, "s", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(p.Competencies) != 0 {
			t.Fatalf("competencies=%d", len(p.Competencies))
		}
	})
}

func TestTalentProfileTransitions(t *testing.T) {
	p, err := NewTalentProfile(kernel.NewID(), "s", nil)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	p.MarkScreened()
	if p.PassportStatus != PassportScreened {
		t.Fatalf("status=%v want screened", p.PassportStatus)
	}
	p.MarkVerified()
	if p.PassportStatus != PassportVerified {
		t.Fatalf("status=%v want verified", p.PassportStatus)
	}
}

func TestTalentProfileAddCompetency(t *testing.T) {
	p, err := NewTalentProfile(kernel.NewID(), "s", nil)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := p.AddCompetency(ProfileCompetency{Name: "Go", Level: 4}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Competencies) != 1 {
		t.Fatalf("competencies=%d", len(p.Competencies))
	}

	err = p.AddCompetency(ProfileCompetency{Name: "", Level: 4})
	if err == nil {
		t.Fatal("expected error for invalid competency")
	}
	if kernel.KindOf(err) != kernel.KindInvalid {
		t.Fatalf("kind=%v", kernel.KindOf(err))
	}
	if len(p.Competencies) != 1 {
		t.Fatalf("invalid competency was appended: %d", len(p.Competencies))
	}
}
