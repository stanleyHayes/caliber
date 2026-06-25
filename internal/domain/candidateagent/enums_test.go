package candidateagent

import "testing"

func TestApplicationSourceValid(t *testing.T) {
	tests := []struct {
		name string
		src  ApplicationSource
		want bool
	}{
		{"unknown", SourceUnknown, false},
		{"manual", SourceManual, true},
		{"agent", SourceAgent, true},
		{"out of range", ApplicationSource(99), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.src.Valid(); got != tt.want {
				t.Fatalf("Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplicationSourceString(t *testing.T) {
	tests := []struct {
		src  ApplicationSource
		want string
	}{
		{SourceManual, "manual"},
		{SourceAgent, "agent"},
		{SourceUnknown, "unknown"},
		{ApplicationSource(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.src.String(); got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplicationStatusValid(t *testing.T) {
	tests := []struct {
		name string
		st   ApplicationStatus
		want bool
	}{
		{"unknown", StatusUnknown, false},
		{"drafted", StatusDrafted, true},
		{"submitted", StatusSubmitted, true},
		{"screening", StatusScreening, true},
		{"screened", StatusScreened, true},
		{"out of range", ApplicationStatus(99), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.st.Valid(); got != tt.want {
				t.Fatalf("Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplicationStatusString(t *testing.T) {
	tests := []struct {
		st   ApplicationStatus
		want string
	}{
		{StatusDrafted, "drafted"},
		{StatusSubmitted, "submitted"},
		{StatusScreening, "screening"},
		{StatusScreened, "screened"},
		{StatusUnknown, "unknown"},
		{ApplicationStatus(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.st.String(); got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
