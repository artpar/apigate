// Package group provides user group value types and pure validation functions.
// This package has NO dependencies on I/O or external packages.
package group

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// Role represents a member's role within a group.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// IsValid returns true if the role is a known valid role.
func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember:
		return true
	}
	return false
}

// CanInvite returns true if this role can invite new members.
func (r Role) CanInvite() bool {
	return r == RoleOwner || r == RoleAdmin
}

// CanRemoveMembers returns true if this role can remove members.
func (r Role) CanRemoveMembers() bool {
	return r == RoleOwner || r == RoleAdmin
}

// CanManageKeys returns true if this role can create/delete group API keys.
func (r Role) CanManageKeys() bool {
	return r == RoleOwner || r == RoleAdmin
}

// CanChangeRoles returns true if this role can change other members' roles.
func (r Role) CanChangeRoles() bool {
	return r == RoleOwner
}

// CanEditGroup returns true if this role can edit group settings.
func (r Role) CanEditGroup() bool {
	return r == RoleOwner
}

// CanDeleteGroup returns true if this role can delete the group.
func (r Role) CanDeleteGroup() bool {
	return r == RoleOwner
}

// CanManageBilling returns true if this role can manage group billing.
func (r Role) CanManageBilling() bool {
	return r == RoleOwner
}

// Status represents the group's current state.
type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
)

// IsValid returns true if the status is a known valid status.
func (s Status) IsValid() bool {
	return s == StatusActive || s == StatusSuspended
}

// Group represents a user group (immutable value type).
type Group struct {
	ID               string
	Name             string
	Slug             string // URL-friendly identifier
	Description      string
	OwnerID          string
	PlanID           string
	BillingEmail     string
	StripeCustomerID string
	Status           Status
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// GenerateID creates a new group ID.
func GenerateID() string {
	idBytes := make([]byte, 8)
	rand.Read(idBytes)
	return "grp_" + hex.EncodeToString(idBytes)
}

// WithID returns a copy of the group with the ID set.
func (g Group) WithID(id string) Group {
	g.ID = id
	return g
}

// WithOwnerID returns a copy of the group with the OwnerID set.
func (g Group) WithOwnerID(ownerID string) Group {
	g.OwnerID = ownerID
	return g
}

// WithPlanID returns a copy of the group with the PlanID set.
func (g Group) WithPlanID(planID string) Group {
	g.PlanID = planID
	return g
}

// WithStatus returns a copy of the group with the Status set.
func (g Group) WithStatus(status Status) Group {
	g.Status = status
	return g
}

// IsActive returns true if the group is active.
func (g Group) IsActive() bool {
	return g.Status == StatusActive
}

// Member represents a group membership (immutable value type).
type Member struct {
	ID        string
	GroupID   string
	UserID    string
	Role      Role
	InvitedBy string
	InvitedAt *time.Time
	JoinedAt  time.Time
}

// GenerateMemberID creates a new member ID.
func GenerateMemberID() string {
	idBytes := make([]byte, 8)
	rand.Read(idBytes)
	return "gm_" + hex.EncodeToString(idBytes)
}

// WithRole returns a copy of the member with the Role set.
func (m Member) WithRole(role Role) Member {
	m.Role = role
	return m
}

// Invite represents a pending group invitation (immutable value type).
type Invite struct {
	ID        string
	GroupID   string
	Email     string
	Role      Role
	InvitedBy string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// GenerateInviteID creates a new invite ID.
func GenerateInviteID() string {
	idBytes := make([]byte, 8)
	rand.Read(idBytes)
	return "gi_" + hex.EncodeToString(idBytes)
}

// GenerateInviteToken creates a secure random invite token.
func GenerateInviteToken() string {
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	return hex.EncodeToString(tokenBytes)
}

// IsExpired returns true if the invite has expired.
func (i Invite) IsExpired() bool {
	return time.Now().UTC().After(i.ExpiresAt)
}

// IsValid returns true if the invite is not expired.
func (i Invite) IsValid() bool {
	return !i.IsExpired()
}

// CreateGroupRequest represents a group creation request (value type).
type CreateGroupRequest struct {
	Name         string
	Slug         string
	Description  string
	BillingEmail string
}

// CreateGroupResult represents the outcome of group creation validation.
type CreateGroupResult struct {
	Valid  bool
	Errors map[string]string
}

// slugRegex matches valid URL-friendly slugs.
var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ValidateCreateGroup validates a group creation request (pure function).
func ValidateCreateGroup(req CreateGroupRequest) CreateGroupResult {
	errors := make(map[string]string)

	// Validate name
	name := strings.TrimSpace(req.Name)
	if name == "" {
		errors["name"] = "Name is required"
	} else if len(name) < 2 {
		errors["name"] = "Name must be at least 2 characters"
	} else if len(name) > 100 {
		errors["name"] = "Name must be less than 100 characters"
	}

	// Validate slug
	slug := strings.TrimSpace(req.Slug)
	if slug == "" {
		errors["slug"] = "Slug is required"
	} else if len(slug) < 2 {
		errors["slug"] = "Slug must be at least 2 characters"
	} else if len(slug) > 50 {
		errors["slug"] = "Slug must be less than 50 characters"
	} else if !slugRegex.MatchString(slug) {
		errors["slug"] = "Slug must contain only lowercase letters, numbers, and hyphens"
	}

	// Validate billing email if provided
	if req.BillingEmail != "" && !isValidEmail(req.BillingEmail) {
		errors["billing_email"] = "Invalid email format"
	}

	return CreateGroupResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// UpdateGroupRequest represents a group update request (value type).
type UpdateGroupRequest struct {
	Name         *string
	Description  *string
	BillingEmail *string
}

// ValidateUpdateGroup validates a group update request (pure function).
func ValidateUpdateGroup(req UpdateGroupRequest) CreateGroupResult {
	errors := make(map[string]string)

	// Validate name if provided
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			errors["name"] = "Name cannot be empty"
		} else if len(name) < 2 {
			errors["name"] = "Name must be at least 2 characters"
		} else if len(name) > 100 {
			errors["name"] = "Name must be less than 100 characters"
		}
	}

	// Validate billing email if provided
	if req.BillingEmail != nil && *req.BillingEmail != "" && !isValidEmail(*req.BillingEmail) {
		errors["billing_email"] = "Invalid email format"
	}

	return CreateGroupResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// InviteRequest represents a member invite request (value type).
type InviteRequest struct {
	Email string
	Role  Role
}

// InviteResult represents the outcome of invite validation.
type InviteResult struct {
	Valid  bool
	Errors map[string]string
}

// ValidateInvite validates an invite request (pure function).
func ValidateInvite(req InviteRequest) InviteResult {
	errors := make(map[string]string)

	// Validate email
	email := strings.TrimSpace(req.Email)
	if email == "" {
		errors["email"] = "Email is required"
	} else if !isValidEmail(email) {
		errors["email"] = "Invalid email format"
	}

	// Validate role
	if !req.Role.IsValid() {
		errors["role"] = "Invalid role"
	} else if req.Role == RoleOwner {
		errors["role"] = "Cannot invite as owner"
	}

	return InviteResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// GenerateSlug creates a URL-friendly slug from a name.
func GenerateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces and underscores with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range slug {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			result.WriteRune(r)
		}
	}
	slug = result.String()

	// Remove consecutive hyphens
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	// Limit length
	if len(slug) > 50 {
		slug = slug[:50]
		// Ensure we don't end with a hyphen
		slug = strings.TrimRight(slug, "-")
	}

	return slug
}

// Helper functions (pure)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	return emailRegex.MatchString(email)
}
