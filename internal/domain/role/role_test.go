package role

import (
	"testing"
	"time"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func validRubric() Rubric {
	return Rubric{Competencies: []Competency{
		{Name: "Go", Weight: 0.5, MustHave: true},
		{Name: "SQL", Weight: 0.5},
	}}
}

func validSpec() RoleSpec {
	return RoleSpec{Title: "Backend Engineer", Location: "Accra", Seniority: SeniorityMid}
}

func TestSeniority(t *testing.T) {
	if SeniorityUnspecified.Valid() {
		t.Error("unspecified should be invalid")
	}
	for s, want := range map[Seniority]string{SeniorityJunior: "junior", SeniorityMid: "mid", SenioritySenior: "senior", SeniorityLead: "lead", SeniorityUnspecified: "unspecified"} {
		if s.String() != want {
			t.Errorf("String(%d) = %q, want %q", s, s.String(), want)
		}
	}
	for in, want := range map[string]Seniority{"Junior": SeniorityJunior, "senior": SenioritySenior, "LEAD": SeniorityLead} {
		got, err := ParseSeniority(in)
		if err != nil || got != want {
			t.Errorf("ParseSeniority(%q) = %v,%v want %v", in, got, err, want)
		}
	}
	if s, err := ParseSeniority(" mid "); err != nil || s != SeniorityMid {
		t.Errorf("ParseSeniority should trim/normalize: %v %v", s, err)
	}
	if _, err := ParseSeniority("staff"); err == nil {
		t.Error("ParseSeniority(staff) should error")
	}
}

func TestCompetencyValidate(t *testing.T) {
	if err := (Competency{Name: "Go", Weight: 0.5}).Validate(); err != nil {
		t.Errorf("valid competency rejected: %v", err)
	}
	if err := (Competency{Name: "", Weight: 0.5}).Validate(); err == nil {
		t.Error("empty name should fail")
	}
	if err := (Competency{Name: "Go", Weight: 1.5}).Validate(); err == nil {
		t.Error("weight > 1 should fail")
	}
	if err := (Competency{Name: "Go", Weight: -0.1}).Validate(); err == nil {
		t.Error("negative weight should fail")
	}
}

func TestRubricValidateAndNormalize(t *testing.T) {
	if err := validRubric().Validate(); err != nil {
		t.Errorf("valid rubric rejected: %v", err)
	}
	if err := (Rubric{}).Validate(); err == nil {
		t.Error("empty rubric should fail")
	}
	if err := (Rubric{Competencies: []Competency{{Name: "", Weight: 1}}}).Validate(); err == nil {
		t.Error("invalid competency should fail")
	}
	if err := (Rubric{Competencies: []Competency{{Name: "Go", Weight: 0.3}}}).Validate(); err == nil {
		t.Error("weights not summing to 1 should fail")
	}

	raw := Rubric{Competencies: []Competency{{Name: "Go", Weight: 2}, {Name: "SQL", Weight: 2}}}
	n := raw.Normalize()
	if got := n.TotalWeight(); got < 0.999 || got > 1.001 {
		t.Errorf("normalized total = %v, want 1.0", got)
	}
	zero := Rubric{Competencies: []Competency{{Name: "Go", Weight: 0}}}
	if got := zero.Normalize().TotalWeight(); got != 0 {
		t.Errorf("zero-weight normalize changed total: %v", got)
	}
}

func TestRoleSpecValidate(t *testing.T) {
	if err := validSpec().Validate(); err != nil {
		t.Errorf("valid spec rejected: %v", err)
	}
	bad := validSpec()
	bad.Title = "  "
	if err := bad.Validate(); err == nil {
		t.Error("blank title should fail")
	}
	bad = validSpec()
	bad.Seniority = SeniorityUnspecified
	if err := bad.Validate(); err == nil {
		t.Error("invalid seniority should fail")
	}
	bad = validSpec()
	bad.SalaryBand = kernel.SalaryBand{Currency: "GHS", Low: 2000, High: 1000}
	if err := bad.Validate(); err == nil {
		t.Error("invalid salary band should fail")
	}
}

func TestRoleStatusValid(t *testing.T) {
	if RoleStatusUnspecified.Valid() {
		t.Error("unspecified should be invalid")
	}
	if !RoleDraft.Valid() || !RoleOpen.Valid() || !RoleClosed.Valid() {
		t.Error("draft/open/closed should be valid")
	}
}

func TestNewRoleAndTransitions(t *testing.T) {
	emp := kernel.NewID()
	now := time.Unix(1700000000, 0)
	r, err := NewRole(emp, validSpec(), validRubric(), now)
	if err != nil {
		t.Fatalf("NewRole: %v", err)
	}
	if r.ID.IsZero() || r.Status != RoleDraft || r.Title != "Backend Engineer" {
		t.Error("unexpected role fields")
	}
	if err := r.Open(); err != nil {
		t.Errorf("Open from draft: %v", err)
	}
	if err := r.Open(); err == nil {
		t.Error("Open from open should fail")
	}
	r.Close()
	if r.Status != RoleClosed {
		t.Error("Close did not set status")
	}
	if err := r.Open(); err != nil {
		t.Errorf("Open from closed: %v", err)
	}

	if _, err := NewRole(kernel.ID(""), validSpec(), validRubric(), now); err == nil {
		t.Error("zero employer should fail")
	}
	if _, err := NewRole(emp, RoleSpec{}, validRubric(), now); err == nil {
		t.Error("invalid spec should fail")
	}
	if _, err := NewRole(emp, validSpec(), Rubric{}, now); err == nil {
		t.Error("invalid rubric should fail")
	}
}
