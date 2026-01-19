package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/artpar/apigate/domain/group"
	"github.com/artpar/apigate/ports"
)

// GroupStore implements ports.GroupStore using SQLite.
type GroupStore struct {
	db *DB
}

// NewGroupStore creates a new SQLite group store.
func NewGroupStore(db *DB) *GroupStore {
	return &GroupStore{db: db}
}

// Get retrieves a group by ID.
func (s *GroupStore) Get(ctx context.Context, id string) (group.Group, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, description, owner_id, plan_id, billing_email,
		       status, created_at, updated_at
		FROM groups
		WHERE id = ?
	`, id)
	return scanGroup(row)
}

// GetBySlug retrieves a group by slug.
func (s *GroupStore) GetBySlug(ctx context.Context, slug string) (group.Group, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, description, owner_id, plan_id, billing_email,
		       status, created_at, updated_at
		FROM groups
		WHERE slug = ?
	`, slug)
	return scanGroup(row)
}

// Create stores a new group.
func (s *GroupStore) Create(ctx context.Context, g group.Group) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO groups (id, name, slug, description, owner_id, plan_id, billing_email,
		                    status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, g.ID, g.Name, g.Slug, g.Description, g.OwnerID, nullStringVal(g.PlanID),
		nullStringVal(g.BillingEmail), g.Status, g.CreatedAt, g.UpdatedAt)
	return err
}

// Update modifies an existing group.
func (s *GroupStore) Update(ctx context.Context, g group.Group) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE groups
		SET name = ?, slug = ?, description = ?, plan_id = ?, billing_email = ?,
		    status = ?, updated_at = ?
		WHERE id = ?
	`, g.Name, g.Slug, g.Description, nullStringVal(g.PlanID), nullStringVal(g.BillingEmail),
		g.Status, time.Now().UTC(), g.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a group.
func (s *GroupStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM groups WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ListByUser returns all groups a user is a member of.
func (s *GroupStore) ListByUser(ctx context.Context, userID string) ([]group.Group, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT g.id, g.name, g.slug, g.description, g.owner_id, g.plan_id, g.billing_email,
		       g.status, g.created_at, g.updated_at
		FROM groups g
		INNER JOIN group_members gm ON g.id = gm.group_id
		WHERE gm.user_id = ?
		ORDER BY g.name ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanGroups(rows)
}

// ListOwned returns all groups owned by a user.
func (s *GroupStore) ListOwned(ctx context.Context, ownerID string) ([]group.Group, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, slug, description, owner_id, plan_id, billing_email,
		       status, created_at, updated_at
		FROM groups
		WHERE owner_id = ?
		ORDER BY name ASC
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanGroups(rows)
}

func scanGroup(row *sql.Row) (group.Group, error) {
	var g group.Group
	var description, planID, billingEmail sql.NullString

	err := row.Scan(
		&g.ID, &g.Name, &g.Slug, &description, &g.OwnerID, &planID, &billingEmail,
		&g.Status, &g.CreatedAt, &g.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return group.Group{}, ErrNotFound
	}
	if err != nil {
		return group.Group{}, err
	}

	g.Description = description.String
	g.PlanID = planID.String
	g.BillingEmail = billingEmail.String

	return g, nil
}

func scanGroups(rows *sql.Rows) ([]group.Group, error) {
	var groups []group.Group
	for rows.Next() {
		var g group.Group
		var description, planID, billingEmail sql.NullString

		err := rows.Scan(
			&g.ID, &g.Name, &g.Slug, &description, &g.OwnerID, &planID, &billingEmail,
			&g.Status, &g.CreatedAt, &g.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		g.Description = description.String
		g.PlanID = planID.String
		g.BillingEmail = billingEmail.String

		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// nullStringVal returns sql.NullString for optional strings.
// Note: This function exists in multiple store files - consolidate if needed.
func nullStringVal(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// Ensure interface compliance.
var _ ports.GroupStore = (*GroupStore)(nil)

// GroupMemberStore implements ports.GroupMemberStore using SQLite.
type GroupMemberStore struct {
	db *DB
}

// NewGroupMemberStore creates a new SQLite group member store.
func NewGroupMemberStore(db *DB) *GroupMemberStore {
	return &GroupMemberStore{db: db}
}

// Get retrieves a membership by ID.
func (s *GroupMemberStore) Get(ctx context.Context, id string) (group.Member, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, group_id, user_id, role, invited_by, invited_at, joined_at
		FROM group_members
		WHERE id = ?
	`, id)
	return scanMember(row)
}

// GetByGroupAndUser retrieves a membership by group and user.
func (s *GroupMemberStore) GetByGroupAndUser(ctx context.Context, groupID, userID string) (group.Member, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, group_id, user_id, role, invited_by, invited_at, joined_at
		FROM group_members
		WHERE group_id = ? AND user_id = ?
	`, groupID, userID)
	return scanMember(row)
}

// Create stores a new membership.
func (s *GroupMemberStore) Create(ctx context.Context, m group.Member) error {
	var invitedAt interface{}
	if m.InvitedAt != nil {
		invitedAt = *m.InvitedAt
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO group_members (id, group_id, user_id, role, invited_by, invited_at, joined_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, m.ID, m.GroupID, m.UserID, m.Role, nullStringVal(m.InvitedBy), invitedAt, m.JoinedAt)
	return err
}

// Update modifies a membership.
func (s *GroupMemberStore) Update(ctx context.Context, m group.Member) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE group_members SET role = ? WHERE id = ?
	`, m.Role, m.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a membership.
func (s *GroupMemberStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM group_members WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ListByGroup returns all members of a group.
func (s *GroupMemberStore) ListByGroup(ctx context.Context, groupID string) ([]group.Member, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, group_id, user_id, role, invited_by, invited_at, joined_at
		FROM group_members
		WHERE group_id = ?
		ORDER BY joined_at ASC
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMembers(rows)
}

// ListByUser returns all group memberships for a user.
func (s *GroupMemberStore) ListByUser(ctx context.Context, userID string) ([]group.Member, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, group_id, user_id, role, invited_by, invited_at, joined_at
		FROM group_members
		WHERE user_id = ?
		ORDER BY joined_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMembers(rows)
}

func scanMember(row *sql.Row) (group.Member, error) {
	var m group.Member
	var invitedBy sql.NullString
	var invitedAt sql.NullTime

	err := row.Scan(
		&m.ID, &m.GroupID, &m.UserID, &m.Role, &invitedBy, &invitedAt, &m.JoinedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return group.Member{}, ErrNotFound
	}
	if err != nil {
		return group.Member{}, err
	}

	m.InvitedBy = invitedBy.String
	if invitedAt.Valid {
		m.InvitedAt = &invitedAt.Time
	}

	return m, nil
}

func scanMembers(rows *sql.Rows) ([]group.Member, error) {
	var members []group.Member
	for rows.Next() {
		var m group.Member
		var invitedBy sql.NullString
		var invitedAt sql.NullTime

		err := rows.Scan(
			&m.ID, &m.GroupID, &m.UserID, &m.Role, &invitedBy, &invitedAt, &m.JoinedAt,
		)
		if err != nil {
			return nil, err
		}

		m.InvitedBy = invitedBy.String
		if invitedAt.Valid {
			m.InvitedAt = &invitedAt.Time
		}

		members = append(members, m)
	}
	return members, rows.Err()
}

// Ensure interface compliance.
var _ ports.GroupMemberStore = (*GroupMemberStore)(nil)

// GroupInviteStore implements ports.GroupInviteStore using SQLite.
type GroupInviteStore struct {
	db *DB
}

// NewGroupInviteStore creates a new SQLite group invite store.
func NewGroupInviteStore(db *DB) *GroupInviteStore {
	return &GroupInviteStore{db: db}
}

// Get retrieves an invite by ID.
func (s *GroupInviteStore) Get(ctx context.Context, id string) (group.Invite, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, group_id, email, role, invited_by, token, expires_at, created_at
		FROM group_invites
		WHERE id = ?
	`, id)
	return scanInvite(row)
}

// GetByToken retrieves an invite by token.
func (s *GroupInviteStore) GetByToken(ctx context.Context, token string) (group.Invite, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, group_id, email, role, invited_by, token, expires_at, created_at
		FROM group_invites
		WHERE token = ?
	`, token)
	return scanInvite(row)
}

// Create stores a new invite.
func (s *GroupInviteStore) Create(ctx context.Context, inv group.Invite) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO group_invites (id, group_id, email, role, invited_by, token, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, inv.ID, inv.GroupID, inv.Email, inv.Role, inv.InvitedBy, inv.Token, inv.ExpiresAt, inv.CreatedAt)
	return err
}

// Delete removes an invite.
func (s *GroupInviteStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM group_invites WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ListByGroup returns all pending invites for a group.
func (s *GroupInviteStore) ListByGroup(ctx context.Context, groupID string) ([]group.Invite, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, group_id, email, role, invited_by, token, expires_at, created_at
		FROM group_invites
		WHERE group_id = ? AND expires_at > ?
		ORDER BY created_at DESC
	`, groupID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanInvites(rows)
}

// ListByEmail returns all pending invites for an email address.
func (s *GroupInviteStore) ListByEmail(ctx context.Context, email string) ([]group.Invite, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, group_id, email, role, invited_by, token, expires_at, created_at
		FROM group_invites
		WHERE email = ? AND expires_at > ?
		ORDER BY created_at DESC
	`, email, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanInvites(rows)
}

// DeleteExpired removes all expired invites.
func (s *GroupInviteStore) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM group_invites WHERE expires_at < ?
	`, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func scanInvite(row *sql.Row) (group.Invite, error) {
	var inv group.Invite

	err := row.Scan(
		&inv.ID, &inv.GroupID, &inv.Email, &inv.Role, &inv.InvitedBy,
		&inv.Token, &inv.ExpiresAt, &inv.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return group.Invite{}, ErrNotFound
	}
	if err != nil {
		return group.Invite{}, err
	}

	return inv, nil
}

func scanInvites(rows *sql.Rows) ([]group.Invite, error) {
	var invites []group.Invite
	for rows.Next() {
		var inv group.Invite

		err := rows.Scan(
			&inv.ID, &inv.GroupID, &inv.Email, &inv.Role, &inv.InvitedBy,
			&inv.Token, &inv.ExpiresAt, &inv.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		invites = append(invites, inv)
	}
	return invites, rows.Err()
}

// Ensure interface compliance.
var _ ports.GroupInviteStore = (*GroupInviteStore)(nil)
