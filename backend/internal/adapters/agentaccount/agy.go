package agentaccount

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// agySharedDirs are directories in the computer's GEMINI_HOME that accounts share.
var agySharedDirs = []string{
	"config",
}

// agyCliSharedDirs are directories under antigravity-cli shared across accounts.
var agyCliSharedDirs = []string{
	"conversations", "brain", "cache", "mcp", "knowledge", "plugin_data",
}

var agyCredentialEnv = []string{
	"GEMINI_API_KEY",
	"GOOGLE_API_KEY",
	"GOOGLE_APPLICATION_CREDENTIALS",
}

// Agy runs the `agy` CLI under an account home (GEMINI_HOME).
type Agy struct {
	Binary     func(context.Context) (string, error)
	DeviceHome string // the computer's own GEMINI_HOME (~/.gemini)
}

var _ ports.AgentAccountProviderCLI = (*Agy)(nil)

// Provider names Agy.
func (*Agy) Provider() domain.AgentAccountProvider { return domain.AgentAccountProviderAgy }

// LaunchEnv points Agy at home.
func (*Agy) LaunchEnv(home string) map[string]string {
	if home == "" {
		return nil
	}
	env := map[string]string{"GEMINI_HOME": home}
	neutralize(env, agyCredentialEnv...)
	return env
}

// LoginArgv launches interactive Agy to sign in.
func (a *Agy) LoginArgv(ctx context.Context) ([]string, error) {
	bin, err := a.Binary(ctx)
	if err != nil {
		return nil, err
	}
	return []string{bin}, nil
}

// Logout removes authentication tokens from home.
func (a *Agy) Logout(ctx context.Context, home string) error {
	if home == "" {
		home = a.DeviceHome
	}
	_ = os.Remove(filepath.Join(home, "antigravity-cli", "antigravity-oauth-token"))
	_ = os.Remove(filepath.Join(home, "jetski-standalone-oauth-token"))
	_ = os.Remove(filepath.Join(home, "google_accounts.json"))
	return nil
}

// Identity checks if home has an active token.
func (a *Agy) Identity(ctx context.Context, home string) (domain.AgentAccountIdentityCheck, error) {
	if home == "" {
		home = a.DeviceHome
	}

	cliToken := filepath.Join(home, "antigravity-cli", "antigravity-oauth-token")
	jetskiToken := filepath.Join(home, "jetski-standalone-oauth-token")
	googleAccounts := filepath.Join(home, "google_accounts.json")

	hasCliToken := checkFileExists(cliToken)
	hasJetskiToken := checkFileExists(jetskiToken)
	hasGoogleAccounts := checkFileExists(googleAccounts)

	if !hasCliToken && !hasJetskiToken && !hasGoogleAccounts {
		return domain.AgentAccountIdentityCheck{
			SignedIn: false,
			LoggedIn: false,
		}, nil
	}

	var email string
	if hasGoogleAccounts {
		var acc struct {
			Active string `json:"active"`
		}
		if data, err := os.ReadFile(googleAccounts); err == nil {
			_ = json.Unmarshal(data, &acc)
			email = strings.TrimSpace(acc.Active)
		}
	}

	return domain.AgentAccountIdentityCheck{
		SignedIn: true,
		LoggedIn: true,
		Email:    email,
		Plan:     "Antigravity",
	}, nil
}

// Limits returns empty limits windows for Agy.
func (a *Agy) Limits(ctx context.Context, home string) (domain.AgentAccountLimits, error) {
	return domain.AgentAccountLimits{
		Windows:   []domain.AgentAccountLimitWindow{},
		FetchedAt: time.Now().UTC(),
	}, nil
}

// Prepare links home to the computer's own folder.
func (a *Agy) Prepare(home string) error {
	if home == "" {
		return nil
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		return err
	}

	var errs []error
	for _, dir := range agySharedDirs {
		errs = append(errs, linkShared(home, a.DeviceHome, dir, false, false))
	}

	cliHome := filepath.Join(home, "antigravity-cli")
	deviceCliHome := filepath.Join(a.DeviceHome, "antigravity-cli")
	if err := os.MkdirAll(cliHome, 0o700); err != nil {
		errs = append(errs, err)
	} else {
		for _, dir := range agyCliSharedDirs {
			errs = append(errs, linkShared(cliHome, deviceCliHome, dir, false, false))
		}
	}

	acpHome := filepath.Join(home, "antigravity-acp")
	deviceAcpHome := filepath.Join(a.DeviceHome, "antigravity-acp")
	if err := os.MkdirAll(acpHome, 0o700); err != nil {
		errs = append(errs, err)
	} else {
		for _, dir := range []string{"conversations", "brain"} {
			errs = append(errs, linkShared(acpHome, deviceAcpHome, dir, false, false))
		}
	}

	return errors.Join(errs...)
}

func checkFileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
