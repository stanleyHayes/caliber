package role

import (
	"time"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// RoleStatus is the lifecycle state of a role.
type RoleStatus int //nolint:revive // domain name fixed by the role context spec

// Role statuses.
const (
	RoleStatusUnspecified RoleStatus = iota
	RoleDraft
	RoleOpen
	RoleClosed
)

// Valid reports whether the status is known and non-zero.
func (s RoleStatus) Valid() bool { return s >= RoleDraft && s <= RoleClosed }

// Role is an open position with its generated spec and rubric.
type Role struct {
	ID         kernel.ID
	EmployerID kernel.ID
	Title      string
	Status     RoleStatus
	Spec       RoleSpec
	Rubric     Rubric
	CreatedAt  time.Time
}

// NewRole builds a validated draft role from a spec and rubric.
func NewRole(employerID kernel.ID, spec RoleSpec, rubric Rubric, createdAt time.Time) (*Role, error) {
	if employerID.IsZero() {
		return nil, kernel.Invalid("role: employer id is required")
	}
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	if err := rubric.Validate(); err != nil {
		return nil, err
	}
	return &Role{
		ID:         kernel.NewID(),
		EmployerID: employerID,
		Title:      spec.Title,
		Status:     RoleDraft,
		Spec:       spec,
		Rubric:     rubric,
		CreatedAt:  createdAt,
	}, nil
}

// Open transitions the role to open (allowed from draft or closed).
func (r *Role) Open() error {
	if r.Status != RoleDraft && r.Status != RoleClosed {
		return kernel.Invalidf("role: cannot open a role in status %d", r.Status)
	}
	r.Status = RoleOpen
	return nil
}

// Close transitions the role to closed.
func (r *Role) Close() { r.Status = RoleClosed }
