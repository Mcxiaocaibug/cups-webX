package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"cups-web/internal/auth"
	"cups-web/internal/store"
	"golang.org/x/crypto/bcrypt"
)

func TestSetupBootstrapHandlerCreatesAdminAndSettings(t *testing.T) {
	ctx := context.Background()
	s := openSetupTestStore(t)

	prevStore := appStore
	appStore = s
	t.Cleanup(func() { appStore = prevStore })

	auth.SetupSecureCookie("", "")

	body := bytes.NewBufferString(`{"cupsHost":"cups:631","adminPassword":"secret123","confirmPassword":"secret123","retentionDays":14}`)
	req := httptest.NewRequest(http.MethodPost, "/api/setup/bootstrap", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	setupBootstrapHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		OK       bool       `json:"ok"`
		CUPSHost string     `json:"cupsHost"`
		Session  meResponse `json:"session"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected ok=true")
	}
	if resp.CUPSHost != "cups:631" {
		t.Fatalf("expected cupsHost to persist, got %q", resp.CUPSHost)
	}
	if resp.Session.Username != defaultAdminUsername || resp.Session.Role != store.RoleAdmin {
		t.Fatalf("unexpected session payload: %+v", resp.Session)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected session cookies to be set")
	}

	err := s.WithTx(ctx, true, func(tx *sql.Tx) error {
		user, err := store.GetUserByUsername(ctx, tx, defaultAdminUsername)
		if err != nil {
			return err
		}
		if user.Role != store.RoleAdmin {
			t.Fatalf("expected admin role, got %q", user.Role)
		}
		if user.MustChangePassword {
			t.Fatalf("expected setup-created admin to skip forced password change")
		}

		setupComplete, err := store.GetSettingInt(ctx, tx, store.SettingSetupComplete, 0)
		if err != nil {
			return err
		}
		if setupComplete != 1 {
			t.Fatalf("expected setup_complete=1, got %d", setupComplete)
		}

		retention, err := store.GetSettingInt(ctx, tx, store.SettingRetentionDays, 0)
		if err != nil {
			return err
		}
		if retention != 14 {
			t.Fatalf("expected retention_days=14, got %d", retention)
		}

		cupsHost, err := store.GetSettingString(ctx, tx, store.SettingCUPSHost, "")
		if err != nil {
			return err
		}
		if cupsHost != "cups:631" {
			t.Fatalf("expected cups_host to persist, got %q", cupsHost)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("verify persisted setup: %v", err)
	}
}

func TestBootstrapAppStateMarksLegacyInstallComplete(t *testing.T) {
	ctx := context.Background()
	s := openSetupTestStore(t)

	prevStore := appStore
	appStore = s
	t.Cleanup(func() { appStore = prevStore })

	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	err = s.WithTx(ctx, false, func(tx *sql.Tx) error {
		_, err := store.CreateUser(ctx, tx, store.CreateUserInput{
			Username:           defaultAdminUsername,
			PasswordHash:       string(hash),
			Role:               store.RoleAdmin,
			Protected:          true,
			MustChangePassword: false,
		})
		return err
	})
	if err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	if err := bootstrapAppState(ctx); err != nil {
		t.Fatalf("bootstrap app state: %v", err)
	}

	state, err := loadSetupState(ctx)
	if err != nil {
		t.Fatalf("load setup state: %v", err)
	}
	if !state.SetupComplete {
		t.Fatalf("expected legacy install to be marked as setup complete")
	}
	if state.AdminCount != 1 {
		t.Fatalf("expected one admin, got %d", state.AdminCount)
	}
	if state.CUPSHost == "" {
		t.Fatalf("expected cups host fallback to be present")
	}
}

func openSetupTestStore(t *testing.T) *store.Store {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "setup.db")
	s, err := store.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})
	return s
}
