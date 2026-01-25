package group

import (
	"strings"
	"testing"
	"time"
)

func TestRole_IsValid(t *testing.T) {
	tests := []struct {
		role Role
		want bool
	}{
		{RoleOwner, true},
		{RoleAdmin, true},
		{RoleMember, true},
		{Role("invalid"), false},
		{Role(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			got := tt.role.IsValid()
			if got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRole_CanInvite(t *testing.T) {
	tests := []struct {
		role Role
		want bool
	}{
		{RoleOwner, true},
		{RoleAdmin, true},
		{RoleMember, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			got := tt.role.CanInvite()
			if got != tt.want {
				t.Errorf("CanInvite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRole_CanRemoveMembers(t *testing.T) {
	tests := []struct {
		role Role
		want bool
	}{
		{RoleOwner, true},
		{RoleAdmin, true},
		{RoleMember, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			got := tt.role.CanRemoveMembers()
			if got != tt.want {
				t.Errorf("CanRemoveMembers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRole_CanManageKeys(t *testing.T) {
	tests := []struct {
		role Role
		want bool
	}{
		{RoleOwner, true},
		{RoleAdmin, true},
		{RoleMember, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			got := tt.role.CanManageKeys()
			if got != tt.want {
				t.Errorf("CanManageKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRole_CanChangeRoles(t *testing.T) {
	tests := []struct {
		role Role
		want bool
	}{
		{RoleOwner, true},
		{RoleAdmin, false},
		{RoleMember, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			got := tt.role.CanChangeRoles()
			if got != tt.want {
				t.Errorf("CanChangeRoles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRole_CanEditGroup(t *testing.T) {
	tests := []struct {
		role Role
		want bool
	}{
		{RoleOwner, true},
		{RoleAdmin, false},
		{RoleMember, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			got := tt.role.CanEditGroup()
			if got != tt.want {
				t.Errorf("CanEditGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRole_CanDeleteGroup(t *testing.T) {
	tests := []struct {
		role Role
		want bool
	}{
		{RoleOwner, true},
		{RoleAdmin, false},
		{RoleMember, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			got := tt.role.CanDeleteGroup()
			if got != tt.want {
				t.Errorf("CanDeleteGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRole_CanManageBilling(t *testing.T) {
	tests := []struct {
		role Role
		want bool
	}{
		{RoleOwner, true},
		{RoleAdmin, false},
		{RoleMember, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			got := tt.role.CanManageBilling()
			if got != tt.want {
				t.Errorf("CanManageBilling() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatus_IsValid(t *testing.T) {
	tests := []struct {
		status Status
		want   bool
	}{
		{StatusActive, true},
		{StatusSuspended, true},
		{Status("invalid"), false},
		{Status(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.IsValid()
			if got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	id := GenerateID()
	if !strings.HasPrefix(id, "grp_") {
		t.Errorf("GenerateID() = %v, want prefix grp_", id)
	}
	if len(id) != 20 { // grp_ (4) + 16 hex chars
		t.Errorf("GenerateID() length = %d, want 20", len(id))
	}

	// Ensure unique
	id2 := GenerateID()
	if id == id2 {
		t.Error("GenerateID() should generate unique IDs")
	}
}

func TestGroup_WithID(t *testing.T) {
	g := Group{Name: "Test"}
	g2 := g.WithID("grp_123")

	if g2.ID != "grp_123" {
		t.Errorf("WithID() ID = %v, want grp_123", g2.ID)
	}
	if g2.Name != "Test" {
		t.Errorf("WithID() should preserve Name")
	}
	// Original should be unchanged
	if g.ID != "" {
		t.Error("WithID() should not modify original")
	}
}

func TestGroup_WithOwnerID(t *testing.T) {
	g := Group{Name: "Test"}
	g2 := g.WithOwnerID("usr_123")

	if g2.OwnerID != "usr_123" {
		t.Errorf("WithOwnerID() OwnerID = %v, want usr_123", g2.OwnerID)
	}
	if g.OwnerID != "" {
		t.Error("WithOwnerID() should not modify original")
	}
}

func TestGroup_WithPlanID(t *testing.T) {
	g := Group{Name: "Test"}
	g2 := g.WithPlanID("plan_123")

	if g2.PlanID != "plan_123" {
		t.Errorf("WithPlanID() PlanID = %v, want plan_123", g2.PlanID)
	}
	if g.PlanID != "" {
		t.Error("WithPlanID() should not modify original")
	}
}

func TestGroup_WithStatus(t *testing.T) {
	g := Group{Name: "Test"}
	g2 := g.WithStatus(StatusSuspended)

	if g2.Status != StatusSuspended {
		t.Errorf("WithStatus() Status = %v, want suspended", g2.Status)
	}
	if g.Status != "" {
		t.Error("WithStatus() should not modify original")
	}
}

func TestGroup_IsActive(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{"active status", StatusActive, true},
		{"suspended status", StatusSuspended, false},
		{"empty status", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := Group{Status: tt.status}
			got := g.IsActive()
			if got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateMemberID(t *testing.T) {
	id := GenerateMemberID()
	if !strings.HasPrefix(id, "gm_") {
		t.Errorf("GenerateMemberID() = %v, want prefix gm_", id)
	}
	if len(id) != 19 { // gm_ (3) + 16 hex chars
		t.Errorf("GenerateMemberID() length = %d, want 19", len(id))
	}
}

func TestMember_WithRole(t *testing.T) {
	m := Member{UserID: "usr_123", Role: RoleMember}
	m2 := m.WithRole(RoleAdmin)

	if m2.Role != RoleAdmin {
		t.Errorf("WithRole() Role = %v, want admin", m2.Role)
	}
	if m.Role != RoleMember {
		t.Error("WithRole() should not modify original")
	}
}

func TestGenerateInviteID(t *testing.T) {
	id := GenerateInviteID()
	if !strings.HasPrefix(id, "gi_") {
		t.Errorf("GenerateInviteID() = %v, want prefix gi_", id)
	}
	if len(id) != 19 { // gi_ (3) + 16 hex chars
		t.Errorf("GenerateInviteID() length = %d, want 19", len(id))
	}
}

func TestGenerateInviteToken(t *testing.T) {
	token := GenerateInviteToken()
	if len(token) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("GenerateInviteToken() length = %d, want 64", len(token))
	}

	// Ensure unique
	token2 := GenerateInviteToken()
	if token == token2 {
		t.Error("GenerateInviteToken() should generate unique tokens")
	}
}

func TestInvite_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"expired yesterday", time.Now().UTC().Add(-24 * time.Hour), true},
		{"expires tomorrow", time.Now().UTC().Add(24 * time.Hour), false},
		{"expires now (edge case)", time.Now().UTC().Add(-1 * time.Second), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := Invite{ExpiresAt: tt.expiresAt}
			got := i.IsExpired()
			if got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvite_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"expired", time.Now().UTC().Add(-1 * time.Hour), false},
		{"not expired", time.Now().UTC().Add(1 * time.Hour), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := Invite{ExpiresAt: tt.expiresAt}
			got := i.IsValid()
			if got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateCreateGroup(t *testing.T) {
	tests := []struct {
		name       string
		req        CreateGroupRequest
		wantValid  bool
		wantErrors []string
	}{
		{
			name:      "valid request",
			req:       CreateGroupRequest{Name: "Test Group", Slug: "test-group"},
			wantValid: true,
		},
		{
			name:       "empty name",
			req:        CreateGroupRequest{Name: "", Slug: "test-group"},
			wantValid:  false,
			wantErrors: []string{"name"},
		},
		{
			name:       "name too short",
			req:        CreateGroupRequest{Name: "A", Slug: "test-group"},
			wantValid:  false,
			wantErrors: []string{"name"},
		},
		{
			name:       "name too long",
			req:        CreateGroupRequest{Name: strings.Repeat("a", 101), Slug: "test"},
			wantValid:  false,
			wantErrors: []string{"name"},
		},
		{
			name:       "empty slug",
			req:        CreateGroupRequest{Name: "Test", Slug: ""},
			wantValid:  false,
			wantErrors: []string{"slug"},
		},
		{
			name:       "slug too short",
			req:        CreateGroupRequest{Name: "Test", Slug: "a"},
			wantValid:  false,
			wantErrors: []string{"slug"},
		},
		{
			name:       "slug too long",
			req:        CreateGroupRequest{Name: "Test", Slug: strings.Repeat("a", 51)},
			wantValid:  false,
			wantErrors: []string{"slug"},
		},
		{
			name:       "invalid slug format",
			req:        CreateGroupRequest{Name: "Test", Slug: "Invalid Slug!"},
			wantValid:  false,
			wantErrors: []string{"slug"},
		},
		{
			name:       "invalid billing email",
			req:        CreateGroupRequest{Name: "Test", Slug: "test", BillingEmail: "invalid"},
			wantValid:  false,
			wantErrors: []string{"billing_email"},
		},
		{
			name:      "valid with billing email",
			req:       CreateGroupRequest{Name: "Test", Slug: "test", BillingEmail: "test@example.com"},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateCreateGroup(tt.req)
			if result.Valid != tt.wantValid {
				t.Errorf("ValidateCreateGroup() Valid = %v, want %v", result.Valid, tt.wantValid)
			}
			for _, errKey := range tt.wantErrors {
				if _, ok := result.Errors[errKey]; !ok {
					t.Errorf("ValidateCreateGroup() missing error for %v", errKey)
				}
			}
		})
	}
}

func TestValidateUpdateGroup(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name       string
		req        UpdateGroupRequest
		wantValid  bool
		wantErrors []string
	}{
		{
			name:      "empty request is valid",
			req:       UpdateGroupRequest{},
			wantValid: true,
		},
		{
			name:      "valid name update",
			req:       UpdateGroupRequest{Name: strPtr("New Name")},
			wantValid: true,
		},
		{
			name:       "empty name",
			req:        UpdateGroupRequest{Name: strPtr("")},
			wantValid:  false,
			wantErrors: []string{"name"},
		},
		{
			name:       "name too short",
			req:        UpdateGroupRequest{Name: strPtr("A")},
			wantValid:  false,
			wantErrors: []string{"name"},
		},
		{
			name:       "name too long",
			req:        UpdateGroupRequest{Name: strPtr(strings.Repeat("a", 101))},
			wantValid:  false,
			wantErrors: []string{"name"},
		},
		{
			name:       "invalid billing email",
			req:        UpdateGroupRequest{BillingEmail: strPtr("invalid")},
			wantValid:  false,
			wantErrors: []string{"billing_email"},
		},
		{
			name:      "valid billing email",
			req:       UpdateGroupRequest{BillingEmail: strPtr("test@example.com")},
			wantValid: true,
		},
		{
			name:      "empty billing email is valid (clears it)",
			req:       UpdateGroupRequest{BillingEmail: strPtr("")},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateUpdateGroup(tt.req)
			if result.Valid != tt.wantValid {
				t.Errorf("ValidateUpdateGroup() Valid = %v, want %v", result.Valid, tt.wantValid)
			}
			for _, errKey := range tt.wantErrors {
				if _, ok := result.Errors[errKey]; !ok {
					t.Errorf("ValidateUpdateGroup() missing error for %v", errKey)
				}
			}
		})
	}
}

func TestValidateInvite(t *testing.T) {
	tests := []struct {
		name       string
		req        InviteRequest
		wantValid  bool
		wantErrors []string
	}{
		{
			name:      "valid invite",
			req:       InviteRequest{Email: "user@example.com", Role: RoleMember},
			wantValid: true,
		},
		{
			name:      "admin role valid",
			req:       InviteRequest{Email: "user@example.com", Role: RoleAdmin},
			wantValid: true,
		},
		{
			name:       "empty email",
			req:        InviteRequest{Email: "", Role: RoleMember},
			wantValid:  false,
			wantErrors: []string{"email"},
		},
		{
			name:       "invalid email",
			req:        InviteRequest{Email: "invalid", Role: RoleMember},
			wantValid:  false,
			wantErrors: []string{"email"},
		},
		{
			name:       "invalid role",
			req:        InviteRequest{Email: "user@example.com", Role: Role("invalid")},
			wantValid:  false,
			wantErrors: []string{"role"},
		},
		{
			name:       "owner role not allowed",
			req:        InviteRequest{Email: "user@example.com", Role: RoleOwner},
			wantValid:  false,
			wantErrors: []string{"role"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateInvite(tt.req)
			if result.Valid != tt.wantValid {
				t.Errorf("ValidateInvite() Valid = %v, want %v", result.Valid, tt.wantValid)
			}
			for _, errKey := range tt.wantErrors {
				if _, ok := result.Errors[errKey]; !ok {
					t.Errorf("ValidateInvite() missing error for %v", errKey)
				}
			}
		})
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Test Group", "test-group"},
		{"test group", "test-group"},
		{"Test_Group", "test-group"},
		{"Test  Group", "test-group"},
		{"Test!@#Group", "testgroup"},
		{"Test---Group", "test-group"},
		{"-Test Group-", "test-group"},
		{"A Very " + strings.Repeat("Long ", 20) + "Name", "a-very-long-long-long-long-long-long-long-long-lon"},
		{"123 Numbers", "123-numbers"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSlug(tt.name)
			if got != tt.want {
				t.Errorf("GenerateSlug(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		want  bool
	}{
		{"user@example.com", true},
		{"user.name@example.com", true},
		{"user+tag@example.com", true},
		{"user@subdomain.example.com", true},
		{"invalid", false},
		{"@example.com", false},
		{"user@", false},
		{"user@.com", false},
		{"", false},
		{"  user@example.com  ", true}, // trimmed
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			got := isValidEmail(tt.email)
			if got != tt.want {
				t.Errorf("isValidEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}
