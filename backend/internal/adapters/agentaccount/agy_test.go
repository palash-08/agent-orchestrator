package agentaccount

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

func TestAgyLaunchEnv(t *testing.T) {
	a := &Agy{DeviceHome: "/tmp/fake-gemini"}
	if env := a.LaunchEnv(""); env != nil {
		t.Fatalf("expected nil env for empty home, got: %v", env)
	}

	t.Setenv("GEMINI_API_KEY", "secret-key")
	t.Setenv("GOOGLE_API_KEY", "other-secret")

	env := a.LaunchEnv("/custom/account/home")
	if env["GEMINI_HOME"] != "/custom/account/home" {
		t.Errorf("expected GEMINI_HOME=/custom/account/home, got: %q", env["GEMINI_HOME"])
	}
	if env["GEMINI_API_KEY"] != "" {
		t.Errorf("expected GEMINI_API_KEY neutralized, got: %q", env["GEMINI_API_KEY"])
	}
	if env["GOOGLE_API_KEY"] != "" {
		t.Errorf("expected GOOGLE_API_KEY neutralized, got: %q", env["GOOGLE_API_KEY"])
	}
}

func TestAgyLoginArgv(t *testing.T) {
	a := &Agy{
		Binary: func(_ context.Context) (string, error) {
			return "/usr/bin/agy", nil
		},
	}
	cmd, err := a.LoginArgv(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmd) != 1 || cmd[0] != "/usr/bin/agy" {
		t.Errorf("unexpected LoginArgv: %v", cmd)
	}
}

func TestAgyIdentity(t *testing.T) {
	home := t.TempDir()
	a := &Agy{DeviceHome: home}

	// Case 1: unauthenticated
	id, err := a.Identity(context.Background(), home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.SignedIn || id.LoggedIn {
		t.Errorf("expected unauthenticated, got SignedIn=%v, LoggedIn=%v", id.SignedIn, id.LoggedIn)
	}

	// Case 2: signed in via antigravity-oauth-token
	tokenPath := filepath.Join(home, "antigravity-cli", "antigravity-oauth-token")
	write(t, tokenPath, `{"token":"xyz"}`)
	id, err = a.Identity(context.Background(), home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !id.SignedIn || !id.LoggedIn {
		t.Errorf("expected authenticated, got: %+v", id)
	}

	// Case 3: google_accounts.json with email
	accountsPath := filepath.Join(home, "google_accounts.json")
	write(t, accountsPath, `{"active":"testuser@gmail.com"}`)
	id, err = a.Identity(context.Background(), home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Email != "testuser@gmail.com" {
		t.Errorf("expected email testuser@gmail.com, got %q", id.Email)
	}

	// Case 4: Logout
	if err := a.Logout(context.Background(), home); err != nil {
		t.Fatalf("logout error: %v", err)
	}
	id, err = a.Identity(context.Background(), home)
	if err != nil {
		t.Fatalf("unexpected error after logout: %v", err)
	}
	if id.SignedIn {
		t.Errorf("expected SignedIn=false after logout, got true")
	}
}

func TestAgyPrepareSharesConfigNotTokens(t *testing.T) {
	device := t.TempDir()
	home := filepath.Join(t.TempDir(), "account-home")

	// Set up device home files
	write(t, filepath.Join(device, "config", "settings.json"), `{"shared":true}`)
	write(t, filepath.Join(device, "antigravity-cli", "mcp", "servers.json"), `{}`)
	write(t, filepath.Join(device, "antigravity-cli", "antigravity-oauth-token"), `{"token":"device-token"}`)
	write(t, filepath.Join(device, "google_accounts.json"), `{"active":"device@gmail.com"}`)

	a := &Agy{DeviceHome: device}
	if err := a.Prepare(home); err != nil {
		t.Fatalf("prepare error: %v", err)
	}

	// Verify shared directories are symlinked
	linksTo(t, filepath.Join(home, "config"), filepath.Join(device, "config"))
	linksTo(t, filepath.Join(home, "antigravity-cli", "mcp"), filepath.Join(device, "antigravity-cli", "mcp"))

	// Verify auth tokens were NOT copied or linked to the account home
	absent(t, filepath.Join(home, "google_accounts.json"))
	absent(t, filepath.Join(home, "antigravity-cli", "antigravity-oauth-token"))
}

func TestAgyProvider(t *testing.T) {
	a := &Agy{}
	if a.Provider() != domain.AgentAccountProviderAgy {
		t.Errorf("expected provider 'agy', got %q", a.Provider())
	}
}
