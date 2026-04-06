package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"cups-web/internal/auth"
	"cups-web/internal/ipp"
	"cups-web/internal/store"
	"golang.org/x/crypto/bcrypt"
)

var errSetupAlreadyComplete = errors.New("setup already completed")
var errSetupNeedsAdmin = errors.New("admin session required")

type setupStatusResponse struct {
	SetupComplete bool   `json:"setupComplete"`
	CUPSHost      string `json:"cupsHost"`
	AdminCount    int    `json:"adminCount"`
	AdminUsername string `json:"adminUsername"`
}

type setupTestCUPSPayload struct {
	CUPSHost string `json:"cupsHost"`
}

type setupBootstrapPayload struct {
	CUPSHost        string `json:"cupsHost"`
	AdminPassword   string `json:"adminPassword"`
	ConfirmPassword string `json:"confirmPassword"`
	RetentionDays   *int64 `json:"retentionDays"`
}

func setupStatusHandler(w http.ResponseWriter, r *http.Request) {
	state, err := loadSetupState(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load setup state")
		return
	}
	writeJSON(w, setupStatusResponse{
		SetupComplete: state.SetupComplete,
		CUPSHost:      state.CUPSHost,
		AdminCount:    state.AdminCount,
		AdminUsername: defaultAdminUsername,
	})
}

func setupTestCUPSHandler(w http.ResponseWriter, r *http.Request) {
	allowed, err := setupWriteAllowed(r)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to validate setup access")
		return
	}
	if !allowed {
		writeJSONError(w, http.StatusForbidden, errSetupNeedsAdmin.Error())
		return
	}

	var payload setupTestCUPSPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	host := normalizeCUPSHost(payload.CUPSHost)
	if host == "" {
		host = defaultConfiguredCUPSHost()
	}

	printers, err := ipp.ListPrinters(host)
	if err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]interface{}{
			"ok":       false,
			"cupsHost": host,
			"error":    err.Error(),
		})
		return
	}
	writeJSON(w, map[string]interface{}{
		"ok":           true,
		"cupsHost":     host,
		"printerCount": len(printers),
		"printers":     printers,
	})
}

func setupBootstrapHandler(w http.ResponseWriter, r *http.Request) {
	state, err := loadSetupState(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load setup state")
		return
	}
	if state.SetupComplete || state.AdminCount > 0 {
		writeJSONError(w, http.StatusConflict, errSetupAlreadyComplete.Error())
		return
	}

	var payload setupBootstrapPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	password := strings.TrimSpace(payload.AdminPassword)
	confirm := strings.TrimSpace(payload.ConfirmPassword)
	if len(password) < 6 {
		writeJSONError(w, http.StatusBadRequest, "admin password must be at least 6 characters")
		return
	}
	if password != confirm {
		writeJSONError(w, http.StatusBadRequest, "passwords do not match")
		return
	}
	cupsHost := normalizeCUPSHost(payload.CUPSHost)
	if cupsHost == "" {
		cupsHost = state.CUPSHost
	}
	if cupsHost == "" {
		cupsHost = defaultConfiguredCUPSHost()
	}
	if payload.RetentionDays != nil && *payload.RetentionDays < 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid retentionDays")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	var created store.User
	err = appStore.WithTx(r.Context(), false, func(tx *sql.Tx) error {
		adminCount, err := store.CountAdmins(r.Context(), tx)
		if err != nil {
			return err
		}
		if adminCount > 0 {
			return errSetupAlreadyComplete
		}

		user, err := store.CreateUser(r.Context(), tx, store.CreateUserInput{
			Username:           defaultAdminUsername,
			PasswordHash:       string(hash),
			Role:               store.RoleAdmin,
			Protected:          true,
			MustChangePassword: false,
		})
		if err != nil {
			return err
		}
		created = user
		if err := store.SetSettingString(r.Context(), tx, store.SettingCUPSHost, cupsHost); err != nil {
			return err
		}
		if payload.RetentionDays != nil {
			if err := store.SetSettingInt(r.Context(), tx, store.SettingRetentionDays, *payload.RetentionDays); err != nil {
				return err
			}
		}
		if err := store.SetSettingInt(r.Context(), tx, store.SettingSetupComplete, 1); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errSetupAlreadyComplete) {
			writeJSONError(w, http.StatusConflict, errSetupAlreadyComplete.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to complete setup")
		return
	}

	sess := auth.Session{UserID: created.ID, Username: created.Username, Role: created.Role}
	if err := auth.SetSession(w, sess); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "session error")
		return
	}
	token := randomToken()
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   os.Getenv("SESSION_SECURE") == "true",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	writeJSON(w, map[string]interface{}{
		"ok": true,
		"session": meResponse{
			ID:                 created.ID,
			Username:           created.Username,
			Role:               created.Role,
			MustChangePassword: false,
		},
		"cupsHost": cupsHost,
	})
}

func setupWriteAllowed(r *http.Request) (bool, error) {
	state, err := loadSetupState(r.Context())
	if err != nil {
		return false, err
	}
	if !state.SetupComplete {
		return true, nil
	}

	sess, err := auth.GetSession(r)
	if err != nil {
		return false, nil
	}
	profile, err := loadMeResponse(r.Context(), sess)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return profile.Role == store.RoleAdmin && !profile.MustChangePassword, nil
}
