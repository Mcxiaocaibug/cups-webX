package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"cups-web/internal/auth"
	"cups-web/internal/store"
)

func TestRequirePasswordChangeClearedBlocksProtectedRoutes(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "server.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	prevStore := appStore
	appStore = s
	defer func() { appStore = prevStore }()

	var user store.User
	err = s.WithTx(ctx, false, func(tx *sql.Tx) error {
		created, err := store.CreateUser(ctx, tx, store.CreateUserInput{
			Username:           "admin",
			PasswordHash:       "hash",
			Role:               store.RoleAdmin,
			Protected:          true,
			MustChangePassword: true,
		})
		if err != nil {
			return err
		}
		user = created
		return nil
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	auth.SetupSecureCookie("", "")
	cookieWriter := httptest.NewRecorder()
	if err := auth.SetSession(cookieWriter, auth.Session{UserID: user.ID, Username: user.Username, Role: user.Role}); err != nil {
		t.Fatalf("SetSession: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/printers", nil)
	for _, cookie := range cookieWriter.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()

	handler := requirePasswordChangeCleared(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "mustChangePassword") {
		t.Fatalf("expected mustChangePassword in response body, got %q", rr.Body.String())
	}
}

func TestRequirePasswordChangeClearedAllowsUpdatedUsers(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "server.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	prevStore := appStore
	appStore = s
	defer func() { appStore = prevStore }()

	var user store.User
	err = s.WithTx(ctx, false, func(tx *sql.Tx) error {
		created, err := store.CreateUser(ctx, tx, store.CreateUserInput{
			Username:     "user1",
			PasswordHash: "hash",
			Role:         store.RoleUser,
		})
		if err != nil {
			return err
		}
		user = created
		return nil
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	auth.SetupSecureCookie("", "")
	cookieWriter := httptest.NewRecorder()
	if err := auth.SetSession(cookieWriter, auth.Session{UserID: user.ID, Username: user.Username, Role: user.Role}); err != nil {
		t.Fatalf("SetSession: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/printers", nil)
	for _, cookie := range cookieWriter.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()

	handler := requirePasswordChangeCleared(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}
