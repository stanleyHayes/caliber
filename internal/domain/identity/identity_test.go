package identity

import (
	"testing"
	"time"
)

func TestParseRole(t *testing.T) {
	ok := map[string]Role{"employer": RoleEmployer, "Recruiter": RoleRecruiter, " candidate ": RoleCandidate}
	for in, want := range ok {
		got, err := ParseRole(in)
		if err != nil || got != want {
			t.Errorf("ParseRole(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseRole("admin"); err == nil {
		t.Error("ParseRole(admin) should error")
	}
}

func TestRoleValidString(t *testing.T) {
	if RoleUnspecified.Valid() {
		t.Error("unspecified should be invalid")
	}
	if !RoleEmployer.Valid() || !RoleCandidate.Valid() {
		t.Error("employer/candidate should be valid")
	}
	for r, want := range map[Role]string{RoleEmployer: "employer", RoleRecruiter: "recruiter", RoleCandidate: "candidate", RoleUnspecified: "unspecified"} {
		if r.String() != want {
			t.Errorf("String(%d) = %q, want %q", r, r.String(), want)
		}
	}
}

func TestNewEmail(t *testing.T) {
	e, err := NewEmail("  Foo@Bar.COM ")
	if err != nil {
		t.Fatalf("NewEmail err: %v", err)
	}
	if e.String() != "foo@bar.com" {
		t.Errorf("normalized = %q, want foo@bar.com", e.String())
	}
	for _, bad := range []string{"", "nope", "a b@c.com", "@x.com", "x@y"} {
		if _, err := NewEmail(bad); err == nil {
			t.Errorf("NewEmail(%q) should error", bad)
		}
	}
}

func TestAccountStatus(t *testing.T) {
	if StatusUnspecified.Valid() {
		t.Error("unspecified should be invalid")
	}
	if !StatusActive.Valid() || !StatusLocked.Valid() {
		t.Error("active/locked should be valid")
	}
	for s, want := range map[AccountStatus]string{StatusActive: "active", StatusLocked: "locked", StatusUnspecified: "unspecified"} {
		if s.String() != want {
			t.Errorf("String(%d) = %q, want %q", s, s.String(), want)
		}
	}
}

func TestPasswordPolicy(t *testing.T) {
	p := DefaultPasswordPolicy()
	if p.MinLength != DefaultPasswordMinLength {
		t.Errorf("default MinLength = %d", p.MinLength)
	}
	if err := p.Validate("short"); err == nil {
		t.Error("short password should fail")
	}
	if err := p.Validate("            "); err == nil {
		t.Error("blank password should fail")
	}
	if err := p.Validate("a-strong-enough-password"); err != nil {
		t.Errorf("valid password rejected: %v", err)
	}
	if err := (PasswordPolicy{}).Validate("tiny"); err == nil {
		t.Error("zero MinLength should fall back to default and reject short")
	}
}

func TestNewUser(t *testing.T) {
	em, _ := NewEmail("u@example.com")
	now := time.Unix(1700000000, 0)
	u, err := NewUser(em, RoleCandidate, "Ada", "hash", now)
	if err != nil {
		t.Fatalf("NewUser: %v", err)
	}
	if u.ID.IsZero() || !u.IsActive() || !u.CreatedAt.Equal(now) {
		t.Error("unexpected user fields")
	}
	u.Lock()
	if u.IsActive() {
		t.Error("locked user should not be active")
	}

	if _, err := NewUser("", RoleCandidate, "Ada", "h", now); err == nil {
		t.Error("empty email should fail")
	}
	if _, err := NewUser(em, RoleUnspecified, "Ada", "h", now); err == nil {
		t.Error("invalid role should fail")
	}
	if _, err := NewUser(em, RoleCandidate, "  ", "h", now); err == nil {
		t.Error("blank name should fail")
	}
	if _, err := NewUser(em, RoleCandidate, "Ada", " ", now); err == nil {
		t.Error("blank hash should fail")
	}
}
