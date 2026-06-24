package candidateagent

import (
	"testing"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func TestWakeUpViewValidate(t *testing.T) {
	base := WakeUpView{
		NewMatches:            3,
		ApplicationsSubmitted: 2,
		ScreeningsCompleted:   1,
		EmployersInterested:   1,
		Highlights:            []string{"shortlisted for Acme"},
	}

	tests := []struct {
		name      string
		mutate    func(WakeUpView) WakeUpView
		wantError bool
	}{
		{"valid", func(v WakeUpView) WakeUpView { return v }, false},
		{"zero valued is valid", func(WakeUpView) WakeUpView { return WakeUpView{} }, false},
		{"negative new matches", func(v WakeUpView) WakeUpView { v.NewMatches = -1; return v }, true},
		{"negative submitted", func(v WakeUpView) WakeUpView { v.ApplicationsSubmitted = -1; return v }, true},
		{"negative screenings", func(v WakeUpView) WakeUpView { v.ScreeningsCompleted = -1; return v }, true},
		{"negative employers", func(v WakeUpView) WakeUpView { v.EmployersInterested = -1; return v }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mutate(base).Validate()
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
		})
	}
}
