package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestUpdateUserPasswordClearsMustChangePassword(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "store.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	var userID int64
	err = s.WithTx(ctx, false, func(tx *sql.Tx) error {
		user, err := CreateUser(ctx, tx, CreateUserInput{
			Username:           "alice",
			PasswordHash:       "old-hash",
			Role:               RoleUser,
			MustChangePassword: true,
		})
		if err != nil {
			return err
		}
		userID = user.ID
		return nil
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	err = s.WithTx(ctx, false, func(tx *sql.Tx) error {
		return UpdateUserPassword(ctx, tx, userID, "new-hash")
	})
	if err != nil {
		t.Fatalf("UpdateUserPassword: %v", err)
	}

	err = s.WithTx(ctx, true, func(tx *sql.Tx) error {
		user, err := GetUserByID(ctx, tx, userID)
		if err != nil {
			return err
		}
		if user.MustChangePassword {
			t.Fatalf("expected must_change_password to be cleared")
		}
		if user.PasswordHash != "new-hash" {
			t.Fatalf("expected password hash to be updated, got %q", user.PasswordHash)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
}

func TestUpdateUserCanRequirePasswordChangeOnReset(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "store.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	var userID int64
	err = s.WithTx(ctx, false, func(tx *sql.Tx) error {
		user, err := CreateUser(ctx, tx, CreateUserInput{
			Username:     "bob",
			PasswordHash: "old-hash",
			Role:         RoleUser,
		})
		if err != nil {
			return err
		}
		userID = user.ID
		return nil
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	required := true
	err = s.WithTx(ctx, false, func(tx *sql.Tx) error {
		_, err := UpdateUser(ctx, tx, UpdateUserInput{
			ID:                 userID,
			Username:           "bob",
			PasswordHash:       ptr("reset-hash"),
			MustChangePassword: &required,
			Role:               RoleUser,
		})
		return err
	})
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	err = s.WithTx(ctx, true, func(tx *sql.Tx) error {
		user, err := GetUserByID(ctx, tx, userID)
		if err != nil {
			return err
		}
		if !user.MustChangePassword {
			t.Fatalf("expected must_change_password to be set")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
}

func ptr[T any](v T) *T {
	return &v
}
