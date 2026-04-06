package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"cups-web/internal/auth"
	"cups-web/internal/store"
	"golang.org/x/crypto/bcrypt"
)

type meResponse struct {
	ID                 int64  `json:"id"`
	Username           string `json:"username"`
	Role               string `json:"role"`
	MustChangePassword bool   `json:"mustChangePassword"`
}

type changePasswordPayload struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func MeHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.GetSession(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp, err := loadMeResponse(r.Context(), sess)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		} else {
			writeJSONError(w, http.StatusInternalServerError, "failed to load profile")
		}
		return
	}
	writeJSON(w, resp)
}

func changeMyPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var payload changePasswordPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	payload.CurrentPassword = strings.TrimSpace(payload.CurrentPassword)
	payload.NewPassword = strings.TrimSpace(payload.NewPassword)
	if payload.CurrentPassword == "" || payload.NewPassword == "" {
		writeJSONError(w, http.StatusBadRequest, "currentPassword and newPassword required")
		return
	}
	if len(payload.NewPassword) < 6 {
		writeJSONError(w, http.StatusBadRequest, "new password must be at least 6 characters")
		return
	}

	sess, err := auth.GetSession(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var user store.User
	err = appStore.WithTx(r.Context(), false, func(tx *sql.Tx) error {
		found, err := store.GetUserByID(r.Context(), tx, sess.UserID)
		if err != nil {
			return err
		}
		if bcrypt.CompareHashAndPassword([]byte(found.PasswordHash), []byte(payload.CurrentPassword)) != nil {
			return errInvalidCredentials
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(payload.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if err := store.UpdateUserPassword(r.Context(), tx, sess.UserID, string(hash)); err != nil {
			return err
		}

		user = found
		return nil
	})
	if err != nil {
		if errors.Is(err, errInvalidCredentials) {
			writeJSONError(w, http.StatusUnauthorized, "current password is incorrect")
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	writeJSON(w, map[string]interface{}{
		"ok":                 true,
		"username":           user.Username,
		"mustChangePassword": false,
	})
}

func loadMeResponse(ctx context.Context, sess auth.Session) (meResponse, error) {
	var resp meResponse
	err := appStore.WithTx(ctx, true, func(tx *sql.Tx) error {
		user, err := store.GetUserByID(ctx, tx, sess.UserID)
		if err != nil {
			return err
		}
		resp = meResponse{
			ID:                 user.ID,
			Username:           user.Username,
			Role:               user.Role,
			MustChangePassword: user.MustChangePassword,
		}
		return nil
	})
	return resp, err
}
