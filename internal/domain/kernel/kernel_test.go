package kernel

import (
	"errors"
	"testing"
)

func TestNewIDUniqueAndNonZero(t *testing.T) {
	a, b := NewID(), NewID()
	if a.IsZero() || b.IsZero() {
		t.Fatal("NewID returned a zero id")
	}
	if a == b {
		t.Fatal("NewID returned duplicate ids")
	}
	if len(a.String()) != 32 {
		t.Errorf("ID length = %d, want 32 hex chars", len(a.String()))
	}
}

func TestIDIsZero(t *testing.T) {
	if !ID("").IsZero() {
		t.Error(`ID("").IsZero() = false, want true`)
	}
	if !ID("  ").IsZero() {
		t.Error("whitespace ID should be zero")
	}
	if ID("x").IsZero() {
		t.Error(`ID("x").IsZero() = true, want false`)
	}
}

func TestErrorKindsAndWrap(t *testing.T) {
	cause := errors.New("boom")
	e := Wrap(cause, KindNotFound, "missing")
	if KindOf(e) != KindNotFound {
		t.Errorf("KindOf = %v, want KindNotFound", KindOf(e))
	}
	if !errors.Is(e, cause) {
		t.Error("wrapped error should match its cause via errors.Is")
	}
	if got := e.Error(); got != "missing: boom" {
		t.Errorf("Error() = %q, want %q", got, "missing: boom")
	}
	if KindOf(errors.New("plain")) != KindInternal {
		t.Error("non-domain error should be KindInternal")
	}
	if Invalidf("bad %d", 7).Error() != "bad 7" {
		t.Error("Invalidf formatting failed")
	}
}

func TestSalaryBandValidate(t *testing.T) {
	tests := []struct {
		name string
		band SalaryBand
		ok   bool
	}{
		{"zero/unspecified", SalaryBand{}, true},
		{"valid", SalaryBand{Currency: "GHS", Low: 1000, High: 2000}, true},
		{"high<low", SalaryBand{Currency: "GHS", Low: 2000, High: 1000}, false},
		{"negative", SalaryBand{Currency: "GHS", Low: -1}, false},
		{"missing currency", SalaryBand{Low: 1000, High: 2000}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.band.Validate()
			if tt.ok && err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
			if !tt.ok && err == nil {
				t.Error("Validate() = nil, want error")
			}
		})
	}
}

func TestNewPageClamps(t *testing.T) {
	if p := NewPage(0, 0); p.Number != 1 || p.Size != DefaultPageSize {
		t.Errorf("NewPage(0,0) = %+v, want {1,%d}", p, DefaultPageSize)
	}
	if p := NewPage(3, 5000); p.Size != MaxPageSize {
		t.Errorf("size not clamped: %d", p.Size)
	}
	if p := NewPage(3, 10); p.Offset() != 20 || p.Limit() != 10 {
		t.Errorf("Offset/Limit = %d/%d, want 20/10", p.Offset(), p.Limit())
	}
}

func TestConfidence(t *testing.T) {
	if ConfidenceUnknown.Valid() {
		t.Error("ConfidenceUnknown should be invalid")
	}
	if !ConfidenceHigh.Valid() {
		t.Error("ConfidenceHigh should be valid")
	}
	if ConfidenceMedium.String() != "medium" {
		t.Errorf("String() = %q, want medium", ConfidenceMedium.String())
	}
	if ConfidenceUnknown.String() != "unknown" {
		t.Errorf("String() = %q, want unknown", ConfidenceUnknown.String())
	}
}

func TestTotalPages(t *testing.T) {
	cases := []struct {
		total int64
		size  int
		want  int
	}{
		{0, 20, 0},
		{1, 20, 1},
		{20, 20, 1},
		{21, 20, 2},
		{100, 0, 0},
	}
	for _, c := range cases {
		if got := TotalPages(c.total, c.size); got != c.want {
			t.Errorf("TotalPages(%d,%d) = %d, want %d", c.total, c.size, got, c.want)
		}
	}
}
