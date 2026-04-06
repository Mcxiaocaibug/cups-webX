package main

import (
	"context"
	"database/sql"
	"os"
	"strings"

	"cups-web/internal/store"
)

const defaultAdminUsername = "admin"
const defaultCUPSHost = "localhost:631"

type setupState struct {
	SetupComplete bool
	CUPSHost      string
	AdminCount    int
}

func bootstrapAppState(ctx context.Context) error {
	return appStore.WithTx(ctx, false, func(tx *sql.Tx) error {
		adminCount, err := store.CountAdmins(ctx, tx)
		if err != nil {
			return err
		}
		setupComplete, err := store.GetSettingInt(ctx, tx, store.SettingSetupComplete, 0)
		if err != nil {
			return err
		}
		currentHost, err := store.GetSettingString(ctx, tx, store.SettingCUPSHost, "")
		if err != nil {
			return err
		}
		if strings.TrimSpace(currentHost) == "" {
			if err := store.SetSettingString(ctx, tx, store.SettingCUPSHost, defaultConfiguredCUPSHost()); err != nil {
				return err
			}
		}
		if adminCount > 0 && setupComplete == 0 {
			if err := store.SetSettingInt(ctx, tx, store.SettingSetupComplete, 1); err != nil {
				return err
			}
		}
		return nil
	})
}

func loadSetupState(ctx context.Context) (setupState, error) {
	var state setupState
	err := appStore.WithTx(ctx, true, func(tx *sql.Tx) error {
		adminCount, err := store.CountAdmins(ctx, tx)
		if err != nil {
			return err
		}
		setupComplete, err := store.GetSettingInt(ctx, tx, store.SettingSetupComplete, 0)
		if err != nil {
			return err
		}
		cupsHost, err := store.GetSettingString(ctx, tx, store.SettingCUPSHost, "")
		if err != nil {
			return err
		}
		state = setupState{
			SetupComplete: setupComplete > 0 && adminCount > 0,
			CUPSHost:      normalizeCUPSHost(cupsHost),
			AdminCount:    adminCount,
		}
		return nil
	})
	if err != nil {
		return setupState{}, err
	}
	if state.CUPSHost == "" {
		state.CUPSHost = defaultConfiguredCUPSHost()
	}
	return state, nil
}

func currentCUPSHost(ctx context.Context) (string, error) {
	state, err := loadSetupState(ctx)
	if err != nil {
		return "", err
	}
	if state.CUPSHost != "" {
		return state.CUPSHost, nil
	}
	return defaultConfiguredCUPSHost(), nil
}

func defaultConfiguredCUPSHost() string {
	host := normalizeCUPSHost(os.Getenv("CUPS_HOST"))
	if host != "" {
		return host
	}
	return defaultCUPSHost
}

func normalizeCUPSHost(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}
