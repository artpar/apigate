package web

import (
	"context"
	"testing"

	"github.com/artpar/apigate/adapters/auth"
)

func TestWithClaims_GetClaims(t *testing.T) {
	claims := &auth.Claims{
		UserID: "user123",
		Email:  "test@example.com",
		Role:   "admin",
	}

	ctx := withClaims(context.Background(), claims)
	got := getClaims(ctx)

	if got == nil {
		t.Fatal("getClaims() returned nil")
	}
	if got.UserID != claims.UserID {
		t.Errorf("UserID = %s, want %s", got.UserID, claims.UserID)
	}
	if got.Email != claims.Email {
		t.Errorf("Email = %s, want %s", got.Email, claims.Email)
	}
	if got.Role != claims.Role {
		t.Errorf("Role = %s, want %s", got.Role, claims.Role)
	}
}

func TestGetClaims_Empty(t *testing.T) {
	ctx := context.Background()
	got := getClaims(ctx)

	if got != nil {
		t.Error("getClaims() should return nil for empty context")
	}
}

func TestGetClaims_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), claimsKey, "not claims")
	got := getClaims(ctx)

	if got != nil {
		t.Error("getClaims() should return nil for wrong type")
	}
}

func TestPageData(t *testing.T) {
	pd := PageData{
		Title:       "Test Page",
		CurrentPath: "/test",
		User: &UserInfo{
			ID:    "user1",
			Email: "test@example.com",
			Role:  "user",
		},
		Flash: &FlashMessage{
			Type:    "success",
			Message: "Test message",
		},
		Config: &ConfigInfo{
			UpstreamURL: "http://localhost:8000",
			Version:     "1.0.0",
		},
	}

	if pd.Title != "Test Page" {
		t.Error("Title mismatch")
	}
	if pd.User.Email != "test@example.com" {
		t.Error("User email mismatch")
	}
	if pd.Flash.Type != "success" {
		t.Error("Flash type mismatch")
	}
	if pd.Config.Version != "1.0.0" {
		t.Error("Config version mismatch")
	}
}

func TestNewPageData(t *testing.T) {
	h, _, _, _ := newTestHandler()

	// Test without claims
	ctx := context.Background()
	pd := h.newPageData(ctx, "Test Title")

	if pd.Title != "Test Title" {
		t.Errorf("Title = %s, want Test Title", pd.Title)
	}
	if pd.User != nil {
		t.Error("User should be nil without claims")
	}
	if pd.Config == nil {
		t.Error("Config should not be nil")
	}
	if pd.Config.UpstreamURL != h.appSettings.UpstreamURL {
		t.Error("Config UpstreamURL mismatch")
	}

	// Test with claims
	claims := &auth.Claims{
		UserID: "user123",
		Email:  "test@example.com",
		Role:   "admin",
	}
	ctx = withClaims(ctx, claims)
	pd = h.newPageData(ctx, "With User")

	if pd.User == nil {
		t.Fatal("User should not be nil with claims")
	}
	if pd.User.ID != claims.UserID {
		t.Errorf("User.ID = %s, want %s", pd.User.ID, claims.UserID)
	}
	if pd.User.Email != claims.Email {
		t.Errorf("User.Email = %s, want %s", pd.User.Email, claims.Email)
	}
}

func TestWithPortalUser_GetPortalUser(t *testing.T) {
	user := &PortalUser{
		ID:    "user123",
		Email: "test@example.com",
		Name:  "Test User",
	}

	ctx := withPortalUser(context.Background(), user)
	got := getPortalUser(ctx)

	if got == nil {
		t.Fatal("getPortalUser() returned nil")
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
}

func TestGetPortalUser_Empty(t *testing.T) {
	ctx := context.Background()
	got := getPortalUser(ctx)

	if got != nil {
		t.Error("getPortalUser() should return nil for empty context")
	}
}

func TestPortalPageData(t *testing.T) {
	ppd := PortalPageData{
		Title:   "Portal Page",
		AppName: "TestApp",
		User: &PortalUser{
			ID:    "user1",
			Email: "test@example.com",
			Name:  "Test",
		},
		Flash: &FlashMessage{
			Type:    "info",
			Message: "Welcome",
		},
		Data: map[string]string{"key": "value"},
	}

	if ppd.Title != "Portal Page" {
		t.Error("Title mismatch")
	}
	if ppd.AppName != "TestApp" {
		t.Error("AppName mismatch")
	}
	if ppd.User.Name != "Test" {
		t.Error("User name mismatch")
	}
}
