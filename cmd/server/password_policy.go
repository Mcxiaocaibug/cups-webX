package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"cups-web/internal/auth"
	"cups-web/internal/store"
)

func requirePasswordChangeCleared(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := auth.GetSession(r)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		mustChangePassword, err := loadMustChangePassword(r.Context(), sess.UserID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to validate session")
			return
		}
		if mustChangePassword {
			writeJSONStatus(w, http.StatusForbidden, map[string]interface{}{
				"error":              "password change required",
				"mustChangePassword": true,
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loadMustChangePassword(ctx context.Context, userID int64) (bool, error) {
	var mustChangePassword bool
	err := appStore.WithTx(ctx, true, func(tx *sql.Tx) error {
		user, err := store.GetUserByID(ctx, tx, userID)
		if err != nil {
			return err
		}
		mustChangePassword = user.MustChangePassword
		return nil
	})
	return mustChangePassword, err
}
