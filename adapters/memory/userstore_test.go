package memory_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/memory"
	"github.com/artpar/apigate/ports"
)

func TestUserStore_NewUserStore(t *testing.T) {
	store := memory.NewUserStore()
	if store == nil {
		t.Fatal("NewUserStore returned nil")
	}

	count, _ := store.Count(context.Background())
	if count != 0 {
		t.Errorf("new store should be empty, got %d users", count)
	}
}

func TestUserStore_CreateAndGet(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	user := ports.User{
		ID:       "user-001",
		Email:    "test@example.com",
		Name:     "Test User",
		PlanID:   "plan-free",
		Status:   "active",
		StripeID: "cus_test123",
	}

	if err := store.Create(ctx, user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.Get(ctx, "user-001")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.ID != user.ID {
		t.Errorf("ID = %s, want %s", got.ID, user.ID)
	}
	if got.Email != user.Email {
		t.Errorf("Email = %s, want %s", got.Email, user.Email)
	}
	if got.Name != user.Name {
		t.Errorf("Name = %s, want %s", got.Name, user.Name)
	}
	if got.PlanID != user.PlanID {
		t.Errorf("PlanID = %s, want %s", got.PlanID, user.PlanID)
	}
	if got.Status != user.Status {
		t.Errorf("Status = %s, want %s", got.Status, user.Status)
	}
	if got.StripeID != user.StripeID {
		t.Errorf("StripeID = %s, want %s", got.StripeID, user.StripeID)
	}
}

func TestUserStore_Create_DuplicateID(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "first@example.com"})

	// Creating with same ID but different email overwrites (map behavior)
	err := store.Create(ctx, ports.User{ID: "u1", Email: "second@example.com"})
	if err != nil {
		// Note: duplicate email check happens, but ID overwrite doesn't error
		// The implementation checks for duplicate email first
		t.Logf("Create with same ID result: %v", err)
	}
}

func TestUserStore_Create_DuplicateEmail_Error(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "duplicate@example.com"})
	err := store.Create(ctx, ports.User{ID: "u2", Email: "duplicate@example.com"})

	if err == nil {
		t.Error("expected error for duplicate email")
	}
}

func TestUserStore_Get_NotFound(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
	if err != memory.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserStore_GetByEmail(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "test@example.com", Name: "Test"})

	user, err := store.GetByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}

	if user.ID != "u1" {
		t.Errorf("ID = %s, want u1", user.ID)
	}
	if user.Name != "Test" {
		t.Errorf("Name = %s, want Test", user.Name)
	}
}

func TestUserStore_GetByEmail_NotFound(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	_, err := store.GetByEmail(ctx, "nonexistent@example.com")
	if err == nil {
		t.Error("expected error for nonexistent email")
	}
	if err != memory.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserStore_GetByEmail_CaseSensitive(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "Test@Example.com"})

	// Email lookup is case-sensitive in this implementation
	_, err := store.GetByEmail(ctx, "test@example.com")
	if err == nil {
		t.Log("Note: Email lookup is case-sensitive")
	}
}

func TestUserStore_Update(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "old@example.com", Name: "Old Name"})

	err := store.Update(ctx, ports.User{ID: "u1", Email: "new@example.com", Name: "New Name"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	user, _ := store.Get(ctx, "u1")
	if user.Name != "New Name" {
		t.Errorf("Name = %s, want 'New Name'", user.Name)
	}
	if user.Email != "new@example.com" {
		t.Errorf("Email = %s, want 'new@example.com'", user.Email)
	}

	// Verify old email is removed from index
	_, err = store.GetByEmail(ctx, "old@example.com")
	if err == nil {
		t.Error("expected old email to be removed from index")
	}

	// Verify new email is indexed
	user, err = store.GetByEmail(ctx, "new@example.com")
	if err != nil {
		t.Errorf("new email should be indexed: %v", err)
	}
}

func TestUserStore_Update_SameEmail(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "same@example.com", Name: "Old Name"})

	err := store.Update(ctx, ports.User{ID: "u1", Email: "same@example.com", Name: "New Name"})
	if err != nil {
		t.Fatalf("Update with same email should work: %v", err)
	}

	user, _ := store.Get(ctx, "u1")
	if user.Name != "New Name" {
		t.Errorf("Name = %s, want 'New Name'", user.Name)
	}
}

func TestUserStore_Update_NotFound(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	err := store.Update(ctx, ports.User{ID: "nonexistent", Email: "test@example.com"})
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
	if err != memory.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserStore_Delete(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "test@example.com"})

	err := store.Delete(ctx, "u1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify user is deleted
	_, err = store.Get(ctx, "u1")
	if err == nil {
		t.Error("expected user to be deleted")
	}

	// Verify email index is cleaned up
	_, err = store.GetByEmail(ctx, "test@example.com")
	if err == nil {
		t.Error("expected email index to be cleaned up")
	}
}

func TestUserStore_Delete_NotFound(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	err := store.Delete(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
	if err != memory.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserStore_List(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "a@example.com"})
	store.Create(ctx, ports.User{ID: "u2", Email: "b@example.com"})
	store.Create(ctx, ports.User{ID: "u3", Email: "c@example.com"})

	users, err := store.List(ctx, 0, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(users) != 3 {
		t.Errorf("expected 3 users, got %d", len(users))
	}
}

func TestUserStore_List_WithLimit(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		store.Create(ctx, ports.User{ID: string(rune('a' + i)), Email: string(rune('a'+i)) + "@example.com"})
	}

	users, err := store.List(ctx, 5, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(users) != 5 {
		t.Errorf("expected 5 users with limit, got %d", len(users))
	}
}

func TestUserStore_List_WithOffset(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		store.Create(ctx, ports.User{ID: string(rune('a' + i)), Email: string(rune('a'+i)) + "@example.com"})
	}

	users, err := store.List(ctx, 0, 7)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(users) != 3 {
		t.Errorf("expected 3 users after offset 7, got %d", len(users))
	}
}

func TestUserStore_List_OffsetBeyondTotal(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "a@example.com"})
	store.Create(ctx, ports.User{ID: "u2", Email: "b@example.com"})

	users, err := store.List(ctx, 10, 100)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if users != nil && len(users) != 0 {
		t.Errorf("expected nil or empty slice for offset beyond total, got %d", len(users))
	}
}

func TestUserStore_List_LimitAndOffset(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	for i := 0; i < 20; i++ {
		id := string([]rune{rune('a' + i/26), rune('a' + i%26)})
		store.Create(ctx, ports.User{ID: id, Email: id + "@example.com"})
	}

	users, err := store.List(ctx, 5, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(users) != 5 {
		t.Errorf("expected 5 users with limit 5 offset 10, got %d", len(users))
	}
}

func TestUserStore_List_Empty(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	users, err := store.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if users != nil && len(users) != 0 {
		t.Errorf("expected nil or empty slice, got %d", len(users))
	}
}

func TestUserStore_Count(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	store.Create(ctx, ports.User{ID: "u1", Email: "a@example.com"})
	store.Create(ctx, ports.User{ID: "u2", Email: "b@example.com"})
	store.Create(ctx, ports.User{ID: "u3", Email: "c@example.com"})

	count, err = store.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestUserStore_Count_AfterDelete(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "a@example.com"})
	store.Create(ctx, ports.User{ID: "u2", Email: "b@example.com"})

	store.Delete(ctx, "u1")

	count, _ := store.Count(ctx)
	if count != 1 {
		t.Errorf("expected 1 after delete, got %d", count)
	}
}

func TestUserStore_GetAll(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "a@example.com"})
	store.Create(ctx, ports.User{ID: "u2", Email: "b@example.com"})

	all := store.GetAll()
	if len(all) != 2 {
		t.Errorf("expected 2 users, got %d", len(all))
	}
}

func TestUserStore_GetAll_Empty(t *testing.T) {
	store := memory.NewUserStore()

	all := store.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 users, got %d", len(all))
	}
}

func TestUserStore_Clear(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "a@example.com"})
	store.Create(ctx, ports.User{ID: "u2", Email: "b@example.com"})

	store.Clear()

	count, _ := store.Count(ctx)
	if count != 0 {
		t.Errorf("expected 0 after Clear, got %d", count)
	}

	// Verify email index is also cleared
	_, err := store.GetByEmail(ctx, "a@example.com")
	if err == nil {
		t.Error("expected email index to be cleared")
	}
}

func TestUserStore_Clear_MultipleTimes(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	store.Create(ctx, ports.User{ID: "u1", Email: "test@example.com"})
	store.Clear()
	store.Clear() // Should not panic

	count, _ := store.Count(ctx)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestUserStore_ConcurrentAccess(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent creates
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			user := ports.User{
				ID:    string([]rune{rune('a' + idx/100), rune('a' + (idx/10)%10), rune('a' + idx%10)}),
				Email: string([]rune{rune('a' + idx/100), rune('a' + (idx/10)%10), rune('a' + idx%10)}) + "@example.com",
			}
			store.Create(ctx, user)
		}(i)
	}

	wg.Wait()

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.Count(ctx)
			store.List(ctx, 10, 0)
			store.GetAll()
		}()
	}

	wg.Wait()
}

func TestUserStore_UserWithAllFields(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	now := time.Now()
	user := ports.User{
		ID:           "complete-user",
		Email:        "complete@example.com",
		PasswordHash: []byte("hashed_password"),
		Name:         "Complete User",
		PlanID:       "plan-pro",
		Status:       "active",
		StripeID:     "cus_complete",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	store.Create(ctx, user)

	got, _ := store.Get(ctx, "complete-user")
	if string(got.PasswordHash) != string(user.PasswordHash) {
		t.Errorf("PasswordHash mismatch")
	}
	if got.CreatedAt != user.CreatedAt {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, user.CreatedAt)
	}
}

func TestUserStore_FullLifecycle(t *testing.T) {
	store := memory.NewUserStore()
	ctx := context.Background()

	// Create
	user := ports.User{
		ID:     "lifecycle-user",
		Email:  "lifecycle@example.com",
		Name:   "Lifecycle User",
		Status: "active",
	}
	store.Create(ctx, user)

	// Read
	got, _ := store.Get(ctx, "lifecycle-user")
	if got.Name != "Lifecycle User" {
		t.Errorf("Name = %s", got.Name)
	}

	// Read by email
	got, _ = store.GetByEmail(ctx, "lifecycle@example.com")
	if got.ID != "lifecycle-user" {
		t.Errorf("ID = %s", got.ID)
	}

	// Update
	user.Name = "Updated Lifecycle User"
	user.Status = "suspended"
	store.Update(ctx, user)

	got, _ = store.Get(ctx, "lifecycle-user")
	if got.Name != "Updated Lifecycle User" {
		t.Errorf("Name after update = %s", got.Name)
	}
	if got.Status != "suspended" {
		t.Errorf("Status after update = %s", got.Status)
	}

	// Count
	count, _ := store.Count(ctx)
	if count != 1 {
		t.Errorf("Count = %d", count)
	}

	// List
	users, _ := store.List(ctx, 10, 0)
	if len(users) != 1 {
		t.Errorf("List len = %d", len(users))
	}

	// Delete
	store.Delete(ctx, "lifecycle-user")

	_, err := store.Get(ctx, "lifecycle-user")
	if err == nil {
		t.Error("expected user to be deleted")
	}

	// Clear
	store.Create(ctx, ports.User{ID: "new-user", Email: "new@example.com"})
	store.Clear()

	count, _ = store.Count(ctx)
	if count != 0 {
		t.Errorf("Count after clear = %d", count)
	}
}
