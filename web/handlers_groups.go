package web

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/artpar/apigate/domain/group"
	"github.com/artpar/apigate/ports"
	"github.com/go-chi/chi/v5"
)

// groupPageData extends PageData for group pages.
type groupPageData struct {
	PageData
	Groups       []group.Group
	Group        group.Group
	Members      []memberWithUser
	Invites      []group.Invite
	UserRole     group.Role
	Plans        []planOption
	FormErrors   map[string]string
	FormValues   map[string]string
}

// memberWithUser combines member data with user info for display.
type memberWithUser struct {
	Member group.Member
	Email  string
	Name   string
}

// planOption represents a plan for selection.
type planOption struct {
	ID   string
	Name string
}

// GroupsPage shows the list of groups the user belongs to.
func (h *Handler) GroupsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		h.redirectToLogin(w, r)
		return
	}

	// Get groups for current user
	var groups []group.Group
	if h.groups != nil {
		var err error
		groups, err = h.groups.ListByUser(ctx, claims.UserID)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to list groups")
			// Continue with empty list
		}
	}

	data := h.newPageData(ctx, "Groups")
	h.render(w, "groups", groupPageData{
		PageData: data,
		Groups:   groups,
	})
}

// GroupNewPage shows the form to create a new group.
func (h *Handler) GroupNewPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		h.redirectToLogin(w, r)
		return
	}

	// Get available plans
	plans := h.getPlansForSelect(ctx)

	data := h.newPageData(ctx, "New Group")
	h.render(w, "group_new", groupPageData{
		PageData:   data,
		Plans:      plans,
		FormErrors: make(map[string]string),
		FormValues: make(map[string]string),
	})
}

// GroupCreate handles creating a new group.
func (h *Handler) GroupCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		h.redirectToLogin(w, r)
		return
	}

	if h.groups == nil {
		http.Error(w, "Groups not enabled", http.StatusServiceUnavailable)
		return
	}

	// Parse form values
	name := strings.TrimSpace(r.FormValue("name"))
	slug := strings.TrimSpace(r.FormValue("slug"))
	description := strings.TrimSpace(r.FormValue("description"))
	billingEmail := strings.TrimSpace(r.FormValue("billing_email"))
	planID := strings.TrimSpace(r.FormValue("plan_id"))

	// Auto-generate slug if not provided
	if slug == "" {
		slug = group.GenerateSlug(name)
	}

	// Validate
	req := group.CreateGroupRequest{
		Name:         name,
		Slug:         slug,
		Description:  description,
		BillingEmail: billingEmail,
	}
	result := group.ValidateCreateGroup(req)
	if !result.Valid {
		plans := h.getPlansForSelect(ctx)
		data := h.newPageData(ctx, "New Group")
		h.render(w, "group_new", groupPageData{
			PageData:   data,
			Plans:      plans,
			FormErrors: result.Errors,
			FormValues: map[string]string{
				"name":          name,
				"slug":          slug,
				"description":   description,
				"billing_email": billingEmail,
				"plan_id":       planID,
			},
		})
		return
	}

	// Check if slug is already taken
	_, err := h.groups.GetBySlug(ctx, slug)
	if err == nil {
		plans := h.getPlansForSelect(ctx)
		data := h.newPageData(ctx, "New Group")
		h.render(w, "group_new", groupPageData{
			PageData: data,
			Plans:    plans,
			FormErrors: map[string]string{
				"slug": "This slug is already taken",
			},
			FormValues: map[string]string{
				"name":          name,
				"slug":          slug,
				"description":   description,
				"billing_email": billingEmail,
				"plan_id":       planID,
			},
		})
		return
	}

	now := time.Now().UTC()

	// Create the group
	g := group.Group{
		ID:           group.GenerateID(),
		Name:         name,
		Slug:         slug,
		Description:  description,
		OwnerID:      claims.UserID,
		PlanID:       planID,
		BillingEmail: billingEmail,
		Status:       group.StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.groups.Create(ctx, g); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create group")
		http.Error(w, "Failed to create group", http.StatusInternalServerError)
		return
	}

	// Add creator as owner member
	if h.groupMembers != nil {
		member := group.Member{
			ID:       group.GenerateMemberID(),
			GroupID:  g.ID,
			UserID:   claims.UserID,
			Role:     group.RoleOwner,
			JoinedAt: now,
		}
		if err := h.groupMembers.Create(ctx, member); err != nil {
			h.logger.Error().Err(err).Msg("Failed to add owner as member")
			// Continue - group was created, membership failed
		}
	}

	http.Redirect(w, r, "/groups/"+g.ID+"?success=Group+created", http.StatusFound)
}

// GroupDetailPage shows the group detail/management page.
func (h *Handler) GroupDetailPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		h.redirectToLogin(w, r)
		return
	}

	groupID := chi.URLParam(r, "id")
	if groupID == "" {
		http.Error(w, "Group ID required", http.StatusBadRequest)
		return
	}

	if h.groups == nil {
		http.Error(w, "Groups not enabled", http.StatusServiceUnavailable)
		return
	}

	// Get the group
	g, err := h.groups.Get(ctx, groupID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}
		h.logger.Error().Err(err).Msg("Failed to get group")
		http.Error(w, "Failed to get group", http.StatusInternalServerError)
		return
	}

	// Check if user is a member
	var userRole group.Role
	if h.groupMembers != nil {
		member, err := h.groupMembers.GetByGroupAndUser(ctx, groupID, claims.UserID)
		if err != nil {
			http.Error(w, "You are not a member of this group", http.StatusForbidden)
			return
		}
		userRole = member.Role
	}

	// Get members
	var members []memberWithUser
	if h.groupMembers != nil {
		memberList, err := h.groupMembers.ListByGroup(ctx, groupID)
		if err == nil {
			for _, m := range memberList {
				email := ""
				name := ""
				if h.users != nil {
					user, err := h.users.Get(ctx, m.UserID)
					if err == nil {
						email = user.Email
						name = user.Name
					}
				}
				members = append(members, memberWithUser{
					Member: m,
					Email:  email,
					Name:   name,
				})
			}
		}
	}

	// Get pending invites
	var invites []group.Invite
	if h.groupInvites != nil && userRole.CanInvite() {
		invites, _ = h.groupInvites.ListByGroup(ctx, groupID)
	}

	data := h.newPageData(ctx, g.Name)
	h.render(w, "group_detail", groupPageData{
		PageData: data,
		Group:    g,
		Members:  members,
		Invites:  invites,
		UserRole: userRole,
	})
}

// GroupEditPage shows the form to edit a group.
func (h *Handler) GroupEditPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		h.redirectToLogin(w, r)
		return
	}

	groupID := chi.URLParam(r, "id")
	if groupID == "" {
		http.Error(w, "Group ID required", http.StatusBadRequest)
		return
	}

	if h.groups == nil {
		http.Error(w, "Groups not enabled", http.StatusServiceUnavailable)
		return
	}

	// Get the group
	g, err := h.groups.Get(ctx, groupID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}
		h.logger.Error().Err(err).Msg("Failed to get group")
		http.Error(w, "Failed to get group", http.StatusInternalServerError)
		return
	}

	// Check if user can edit
	var userRole group.Role
	if h.groupMembers != nil {
		member, err := h.groupMembers.GetByGroupAndUser(ctx, groupID, claims.UserID)
		if err != nil || !member.Role.CanEditGroup() {
			http.Error(w, "You don't have permission to edit this group", http.StatusForbidden)
			return
		}
		userRole = member.Role
	}

	plans := h.getPlansForSelect(ctx)

	data := h.newPageData(ctx, "Edit "+g.Name)
	h.render(w, "group_edit", groupPageData{
		PageData:   data,
		Group:      g,
		UserRole:   userRole,
		Plans:      plans,
		FormErrors: make(map[string]string),
	})
}

// GroupUpdate handles updating a group.
func (h *Handler) GroupUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		h.redirectToLogin(w, r)
		return
	}

	groupID := chi.URLParam(r, "id")
	if groupID == "" {
		http.Error(w, "Group ID required", http.StatusBadRequest)
		return
	}

	if h.groups == nil {
		http.Error(w, "Groups not enabled", http.StatusServiceUnavailable)
		return
	}

	// Get the group
	g, err := h.groups.Get(ctx, groupID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}
		h.logger.Error().Err(err).Msg("Failed to get group")
		http.Error(w, "Failed to get group", http.StatusInternalServerError)
		return
	}

	// Check if user can edit
	if h.groupMembers != nil {
		member, err := h.groupMembers.GetByGroupAndUser(ctx, groupID, claims.UserID)
		if err != nil || !member.Role.CanEditGroup() {
			http.Error(w, "You don't have permission to edit this group", http.StatusForbidden)
			return
		}
	}

	// Parse form values
	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	billingEmail := strings.TrimSpace(r.FormValue("billing_email"))
	planID := strings.TrimSpace(r.FormValue("plan_id"))

	// Validate
	req := group.UpdateGroupRequest{
		Name:         &name,
		Description:  &description,
		BillingEmail: &billingEmail,
	}
	result := group.ValidateUpdateGroup(req)
	if !result.Valid {
		plans := h.getPlansForSelect(ctx)
		data := h.newPageData(ctx, "Edit "+g.Name)
		h.render(w, "group_edit", groupPageData{
			PageData:   data,
			Group:      g,
			Plans:      plans,
			FormErrors: result.Errors,
		})
		return
	}

	// Update the group
	g.Name = name
	g.Description = description
	g.BillingEmail = billingEmail
	g.PlanID = planID
	g.UpdatedAt = time.Now().UTC()

	if err := h.groups.Update(ctx, g); err != nil {
		h.logger.Error().Err(err).Msg("Failed to update group")
		http.Error(w, "Failed to update group", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/groups/"+groupID+"?success=Group+updated", http.StatusFound)
}

// GroupDelete handles deleting a group.
func (h *Handler) GroupDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		h.redirectToLogin(w, r)
		return
	}

	groupID := chi.URLParam(r, "id")
	if groupID == "" {
		http.Error(w, "Group ID required", http.StatusBadRequest)
		return
	}

	if h.groups == nil {
		http.Error(w, "Groups not enabled", http.StatusServiceUnavailable)
		return
	}

	// Check if user can delete
	if h.groupMembers != nil {
		member, err := h.groupMembers.GetByGroupAndUser(ctx, groupID, claims.UserID)
		if err != nil || !member.Role.CanDeleteGroup() {
			http.Error(w, "You don't have permission to delete this group", http.StatusForbidden)
			return
		}
	}

	// Delete the group
	if err := h.groups.Delete(ctx, groupID); err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete group")
		http.Error(w, "Failed to delete group", http.StatusInternalServerError)
		return
	}

	// For HTMX requests, return redirect header
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/groups?success=Group+deleted")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/groups?success=Group+deleted", http.StatusFound)
}

// GroupMemberAdd handles adding a member to a group by user ID.
func (h *Handler) GroupMemberAdd(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		h.redirectToLogin(w, r)
		return
	}

	groupID := chi.URLParam(r, "id")
	if groupID == "" {
		http.Error(w, "Group ID required", http.StatusBadRequest)
		return
	}

	if h.groups == nil || h.groupMembers == nil {
		http.Error(w, "Groups not enabled", http.StatusServiceUnavailable)
		return
	}

	// Check if user can add members
	member, err := h.groupMembers.GetByGroupAndUser(ctx, groupID, claims.UserID)
	if err != nil || !member.Role.CanInvite() {
		http.Error(w, "You don't have permission to add members", http.StatusForbidden)
		return
	}

	userID := strings.TrimSpace(r.FormValue("user_id"))
	roleStr := strings.TrimSpace(r.FormValue("role"))

	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	role := group.Role(roleStr)
	if !role.IsValid() {
		role = group.RoleMember
	}

	// Prevent adding as owner
	if role == group.RoleOwner {
		http.Error(w, "Cannot add member as owner", http.StatusBadRequest)
		return
	}

	// Check if already a member
	_, err = h.groupMembers.GetByGroupAndUser(ctx, groupID, userID)
	if err == nil {
		http.Error(w, "User is already a member", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	newMember := group.Member{
		ID:        group.GenerateMemberID(),
		GroupID:   groupID,
		UserID:    userID,
		Role:      role,
		InvitedBy: claims.UserID,
		InvitedAt: &now,
		JoinedAt:  now,
	}

	if err := h.groupMembers.Create(ctx, newMember); err != nil {
		h.logger.Error().Err(err).Msg("Failed to add member")
		http.Error(w, "Failed to add member", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/groups/"+groupID+"?success=Member+added", http.StatusFound)
}

// GroupMemberRemove handles removing a member from a group.
func (h *Handler) GroupMemberRemove(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		h.redirectToLogin(w, r)
		return
	}

	groupID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "uid")

	if groupID == "" || userID == "" {
		http.Error(w, "Group ID and User ID required", http.StatusBadRequest)
		return
	}

	if h.groupMembers == nil {
		http.Error(w, "Groups not enabled", http.StatusServiceUnavailable)
		return
	}

	// Check if user can remove members
	currentMember, err := h.groupMembers.GetByGroupAndUser(ctx, groupID, claims.UserID)
	if err != nil {
		http.Error(w, "You are not a member of this group", http.StatusForbidden)
		return
	}

	// Get member being removed
	targetMember, err := h.groupMembers.GetByGroupAndUser(ctx, groupID, userID)
	if err != nil {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	// Cannot remove owner
	if targetMember.Role == group.RoleOwner {
		http.Error(w, "Cannot remove the group owner", http.StatusBadRequest)
		return
	}

	// Users can remove themselves, or admins/owners can remove others
	if userID != claims.UserID && !currentMember.Role.CanRemoveMembers() {
		http.Error(w, "You don't have permission to remove members", http.StatusForbidden)
		return
	}

	if err := h.groupMembers.Delete(ctx, targetMember.ID); err != nil {
		h.logger.Error().Err(err).Msg("Failed to remove member")
		http.Error(w, "Failed to remove member", http.StatusInternalServerError)
		return
	}

	// For HTMX requests, return success
	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// If user removed themselves, redirect to groups list
	if userID == claims.UserID {
		http.Redirect(w, r, "/groups?success=Left+group", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/groups/"+groupID+"?success=Member+removed", http.StatusFound)
}

// GroupMemberUpdateRole handles updating a member's role.
func (h *Handler) GroupMemberUpdateRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		h.redirectToLogin(w, r)
		return
	}

	groupID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "uid")

	if groupID == "" || userID == "" {
		http.Error(w, "Group ID and User ID required", http.StatusBadRequest)
		return
	}

	if h.groupMembers == nil {
		http.Error(w, "Groups not enabled", http.StatusServiceUnavailable)
		return
	}

	// Check if user can change roles
	currentMember, err := h.groupMembers.GetByGroupAndUser(ctx, groupID, claims.UserID)
	if err != nil || !currentMember.Role.CanChangeRoles() {
		http.Error(w, "You don't have permission to change roles", http.StatusForbidden)
		return
	}

	// Get the target member
	targetMember, err := h.groupMembers.GetByGroupAndUser(ctx, groupID, userID)
	if err != nil {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	// Cannot change owner's role
	if targetMember.Role == group.RoleOwner {
		http.Error(w, "Cannot change the owner's role", http.StatusBadRequest)
		return
	}

	newRole := group.Role(r.FormValue("role"))
	if !newRole.IsValid() || newRole == group.RoleOwner {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}

	// Update the role
	targetMember.Role = newRole
	if err := h.groupMembers.Update(ctx, targetMember); err != nil {
		h.logger.Error().Err(err).Msg("Failed to update role")
		http.Error(w, "Failed to update role", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/groups/"+groupID+"?success=Role+updated", http.StatusFound)
}

// GroupInviteCreate handles creating a group invite.
func (h *Handler) GroupInviteCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		h.redirectToLogin(w, r)
		return
	}

	groupID := chi.URLParam(r, "id")
	if groupID == "" {
		http.Error(w, "Group ID required", http.StatusBadRequest)
		return
	}

	if h.groupInvites == nil || h.groupMembers == nil {
		http.Error(w, "Groups not enabled", http.StatusServiceUnavailable)
		return
	}

	// Check if user can invite
	member, err := h.groupMembers.GetByGroupAndUser(ctx, groupID, claims.UserID)
	if err != nil || !member.Role.CanInvite() {
		http.Error(w, "You don't have permission to invite members", http.StatusForbidden)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	roleStr := strings.TrimSpace(r.FormValue("role"))

	role := group.Role(roleStr)
	if !role.IsValid() {
		role = group.RoleMember
	}

	// Validate invite
	req := group.InviteRequest{
		Email: email,
		Role:  role,
	}
	result := group.ValidateInvite(req)
	if !result.Valid {
		http.Error(w, "Invalid invite: "+getFirstError(result.Errors), http.StatusBadRequest)
		return
	}

	// Check if email is already a member (by looking up user)
	if h.users != nil {
		user, err := h.users.GetByEmail(ctx, email)
		if err == nil {
			_, err := h.groupMembers.GetByGroupAndUser(ctx, groupID, user.ID)
			if err == nil {
				http.Error(w, "User is already a member of this group", http.StatusBadRequest)
				return
			}
		}
	}

	// Check if there's already a pending invite for this email in this group
	existingInvites, _ := h.groupInvites.ListByEmail(ctx, email)
	for _, inv := range existingInvites {
		if inv.GroupID == groupID && inv.IsValid() {
			http.Error(w, "An invite for this email is already pending", http.StatusBadRequest)
			return
		}
	}

	now := time.Now().UTC()
	invite := group.Invite{
		ID:        group.GenerateInviteID(),
		GroupID:   groupID,
		Email:     email,
		Role:      role,
		InvitedBy: claims.UserID,
		Token:     group.GenerateInviteToken(),
		ExpiresAt: now.Add(7 * 24 * time.Hour), // 7 days
		CreatedAt: now,
	}

	if err := h.groupInvites.Create(ctx, invite); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create invite")
		http.Error(w, "Failed to create invite", http.StatusInternalServerError)
		return
	}

	// TODO: Send invite email
	// For now, the invite token can be shared manually

	http.Redirect(w, r, "/groups/"+groupID+"?success=Invite+sent", http.StatusFound)
}

// GroupInviteRevoke handles revoking a group invite.
func (h *Handler) GroupInviteRevoke(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		h.redirectToLogin(w, r)
		return
	}

	groupID := chi.URLParam(r, "id")
	inviteID := chi.URLParam(r, "iid")

	if groupID == "" || inviteID == "" {
		http.Error(w, "Group ID and Invite ID required", http.StatusBadRequest)
		return
	}

	if h.groupInvites == nil || h.groupMembers == nil {
		http.Error(w, "Groups not enabled", http.StatusServiceUnavailable)
		return
	}

	// Check if user can revoke invites
	member, err := h.groupMembers.GetByGroupAndUser(ctx, groupID, claims.UserID)
	if err != nil || !member.Role.CanInvite() {
		http.Error(w, "You don't have permission to revoke invites", http.StatusForbidden)
		return
	}

	if err := h.groupInvites.Delete(ctx, inviteID); err != nil {
		h.logger.Error().Err(err).Msg("Failed to revoke invite")
		http.Error(w, "Failed to revoke invite", http.StatusInternalServerError)
		return
	}

	// For HTMX requests, return success
	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/groups/"+groupID+"?success=Invite+revoked", http.StatusFound)
}

// GroupInviteAcceptPage shows the page to accept a group invite.
func (h *Handler) GroupInviteAcceptPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := chi.URLParam(r, "token")

	if token == "" {
		http.Error(w, "Invite token required", http.StatusBadRequest)
		return
	}

	if h.groupInvites == nil || h.groups == nil {
		http.Error(w, "Groups not enabled", http.StatusServiceUnavailable)
		return
	}

	// Get the invite
	invite, err := h.groupInvites.GetByToken(ctx, token)
	if err != nil {
		http.Error(w, "Invalid or expired invite", http.StatusNotFound)
		return
	}

	if invite.IsExpired() {
		http.Error(w, "This invite has expired", http.StatusGone)
		return
	}

	// Get the group
	g, err := h.groups.Get(ctx, invite.GroupID)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	// Check if user is logged in
	claims := getClaims(ctx)

	data := h.newPageData(ctx, "Accept Invite")
	h.render(w, "group_invite_accept", struct {
		PageData
		Invite     group.Invite
		Group      group.Group
		IsLoggedIn bool
	}{
		PageData:   data,
		Invite:     invite,
		Group:      g,
		IsLoggedIn: claims != nil,
	})
}

// GroupInviteAccept handles accepting a group invite.
func (h *Handler) GroupInviteAccept(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		// Redirect to login with return URL
		token := chi.URLParam(r, "token")
		http.Redirect(w, r, "/login?redirect=/groups/invite/"+token, http.StatusFound)
		return
	}

	token := chi.URLParam(r, "token")
	if token == "" {
		http.Error(w, "Invite token required", http.StatusBadRequest)
		return
	}

	if h.groupInvites == nil || h.groupMembers == nil {
		http.Error(w, "Groups not enabled", http.StatusServiceUnavailable)
		return
	}

	// Get the invite
	invite, err := h.groupInvites.GetByToken(ctx, token)
	if err != nil {
		http.Error(w, "Invalid or expired invite", http.StatusNotFound)
		return
	}

	if invite.IsExpired() {
		http.Error(w, "This invite has expired", http.StatusGone)
		return
	}

	// Check if already a member
	_, err = h.groupMembers.GetByGroupAndUser(ctx, invite.GroupID, claims.UserID)
	if err == nil {
		// Already a member - just delete invite and redirect
		h.groupInvites.Delete(ctx, invite.ID)
		http.Redirect(w, r, "/groups/"+invite.GroupID, http.StatusFound)
		return
	}

	// Add as member
	now := time.Now().UTC()
	member := group.Member{
		ID:        group.GenerateMemberID(),
		GroupID:   invite.GroupID,
		UserID:    claims.UserID,
		Role:      invite.Role,
		InvitedBy: invite.InvitedBy,
		InvitedAt: &invite.CreatedAt,
		JoinedAt:  now,
	}

	if err := h.groupMembers.Create(ctx, member); err != nil {
		h.logger.Error().Err(err).Msg("Failed to add member")
		http.Error(w, "Failed to join group", http.StatusInternalServerError)
		return
	}

	// Delete the invite
	h.groupInvites.Delete(ctx, invite.ID)

	http.Redirect(w, r, "/groups/"+invite.GroupID+"?success=Welcome+to+the+group", http.StatusFound)
}

// PartialGroups returns the groups list partial for HTMX.
func (h *Handler) PartialGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var groups []group.Group
	if h.groups != nil {
		var err error
		groups, err = h.groups.ListByUser(ctx, claims.UserID)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to list groups")
		}
	}

	h.render(w, "partials/groups_list", map[string]interface{}{
		"Groups": groups,
	})
}

// PartialGroupMembers returns the group members partial for HTMX.
func (h *Handler) PartialGroupMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	groupID := chi.URLParam(r, "id")
	if groupID == "" {
		http.Error(w, "Group ID required", http.StatusBadRequest)
		return
	}

	if h.groupMembers == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// Check membership
	currentMember, err := h.groupMembers.GetByGroupAndUser(ctx, groupID, claims.UserID)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	memberList, err := h.groupMembers.ListByGroup(ctx, groupID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list members")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var members []memberWithUser
	for _, m := range memberList {
		email := ""
		name := ""
		if h.users != nil {
			user, err := h.users.Get(ctx, m.UserID)
			if err == nil {
				email = user.Email
				name = user.Name
			}
		}
		members = append(members, memberWithUser{
			Member: m,
			Email:  email,
			Name:   name,
		})
	}

	h.render(w, "partials/group_members", map[string]interface{}{
		"Members":  members,
		"UserRole": currentMember.Role,
		"GroupID":  groupID,
	})
}

// PartialGroupInvites returns the group invites partial for HTMX.
func (h *Handler) PartialGroupInvites(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := getClaims(ctx)
	if claims == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	groupID := chi.URLParam(r, "id")
	if groupID == "" {
		http.Error(w, "Group ID required", http.StatusBadRequest)
		return
	}

	if h.groupInvites == nil || h.groupMembers == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// Check if user can view invites
	member, err := h.groupMembers.GetByGroupAndUser(ctx, groupID, claims.UserID)
	if err != nil || !member.Role.CanInvite() {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	invites, err := h.groupInvites.ListByGroup(ctx, groupID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list invites")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.render(w, "partials/group_invites", map[string]interface{}{
		"Invites": invites,
		"GroupID": groupID,
	})
}

// Helper functions

func (h *Handler) getPlansForSelect(ctx context.Context) []planOption {
	var plans []planOption
	if h.plans != nil {
		planList, err := h.plans.List(ctx)
		if err == nil {
			for _, p := range planList {
				plans = append(plans, planOption{
					ID:   p.ID,
					Name: p.Name,
				})
			}
		}
	}
	return plans
}

func getFirstError(errors map[string]string) string {
	for _, v := range errors {
		return v
	}
	return "Unknown error"
}
