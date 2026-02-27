package main

import (
	"testing"
)

func TestNewCommand_ReturnsCommand(t *testing.T) {
	cmd := newCommand()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
	if cmd.Use != AppName {
		t.Errorf("expected Use=%s, got %s", AppName, cmd.Use)
	}
}

func TestNewCommand_HasExpectedFlags(t *testing.T) {
	expectedFlags := []string{
		"debug",
		"image-filter",
		"kube-config",
		"kube-context",
		"master-url",
		"owners",
		"notifications",
		"storage",
		"filename",
		"s3-bucket",
		"s3-endpoint",
		"s3-region",
		"s3-insecure",
		"git-password",
		"git-url",
		"git-private-key-file",
		"git-directory",
		"github-app-id",
		"github-installation-id",
		"api-key",
		"api-signature",
		"api-endpoint",
		// Don't test "http-header", -> this will result in "flag redefined" errors
		"annotation-name-base",
		"annotation-name-scans",
		"annotation-name-contact",
		"annotation-name-defect-dojo",
		"environment-name",
		"is-scan-dependency-check",
		"is-scan-dependency-track",
		"is-scan-lifetime",
		"is-scan-baseimage-lifetime",
		"is-scan-distroless",
		"is-scan-malware",
		"is-scan-new-version",
		"is-scan-runasroot",
		"is-scan-run-as-privileged",
		"is-scan-potentially-running-as-root",
		"is-scan-potentially-running-as-privileged",
		"ScanLifetimeMaxDays",
		"skip",
		"engagement-tags",
		"container-type",
		"team",
		"product",
		"owners",
		"notifications",
		"namespace-filter",
		"negated_namespace_filter",
	}
	cmd := newCommand()
	for _, flag := range expectedFlags {
		t.Run(flag, func(t *testing.T) {
			if cmd.Flag(flag) == nil && cmd.PersistentFlags().Lookup(flag) == nil {
				t.Errorf("expected flag --%s to be registered", flag)
			}
		})
	}
}

func TestNewCommand_DefaultFlagValues(t *testing.T) {
	cmd := newCommand()

	debugFlag := cmd.PersistentFlags().Lookup("debug")
	if debugFlag.DefValue != "false" {
		t.Errorf("expected debug default=false, got %s", debugFlag.DefValue)
	}

	kubeConfigFlag := cmd.PersistentFlags().Lookup("kube-config")
	if kubeConfigFlag.DefValue != "" {
		t.Errorf("expected kube-config default='', got %s", kubeConfigFlag.DefValue)
	}

	kubeContextFlag := cmd.PersistentFlags().Lookup("kube-context")
	if kubeContextFlag.DefValue != "" {
		t.Errorf("expected kube-context default='', got %s", kubeContextFlag.DefValue)
	}

	masterUrlFlag := cmd.PersistentFlags().Lookup("master-url")
	if masterUrlFlag.DefValue != "" {
		t.Errorf("expected master-url default='', got %s", masterUrlFlag.DefValue)
	}
}

func TestNewCommand_ValidOwnersFlag(t *testing.T) {
	cmd := newCommand()
	// PersistentPreRunE is not triggered by ExecuteC without a Run, so call it directly
	preRunE, _ := cmd.PersistentPreRunE, cmd.PersistentFlags().Set("owners", `[{"role": "admin", "uuid": "1234", "name": "Alice"}]`)
	if err := preRunE(cmd, []string{}); err != nil {
		t.Errorf("unexpected error with valid owners flag: %v", err)
	}
}

func TestNewCommand_InvalidOwnersFlag(t *testing.T) {
	cmd := newCommand()
	if err := cmd.PersistentFlags().Set("owners", "not-valid-json"); err != nil {
		t.Fatalf("failed to set notifications flag: %v", err)
	}

	preRunE := cmd.PersistentPreRunE
	if err := preRunE(cmd, []string{}); err == nil {
		t.Error("expected error for invalid owners JSON, got nil")
	}
}

func TestNewCommand_ValidNotificationsFlag(t *testing.T) {
	cmd := newCommand()
	if err := cmd.PersistentFlags().Set("notifications", `{"slack": ["#channel"], "emails": ["test@example.com"], "msteams": ["team-a"]}`); err != nil {
		t.Fatalf("failed to set notifications flag: %v", err)
	}
	preRunE := cmd.PersistentPreRunE
	if err := preRunE(cmd, []string{}); err != nil {
		t.Errorf("unexpected error with valid notifications flag: %v", err)
	}
}

func TestNewCommand_InvalidNotificationsFlag(t *testing.T) {
	cmd := newCommand()
	if err := cmd.PersistentFlags().Set("notifications", "not-valid-json"); err != nil {
		t.Fatalf("failed to set notifications flag: %v", err)
	}

	preRunE := cmd.PersistentPreRunE
	if err := preRunE(cmd, []string{}); err == nil {
		t.Error("expected error for invalid notifications JSON, got nil")
	}
}

func TestNewCommand_EmptyOwnersFlag_NoError(t *testing.T) {
	cmd := newCommand()
	// owners flag left empty — should not attempt unmarshal
	preRunE := cmd.PersistentPreRunE
	if err := preRunE(cmd, []string{}); err != nil {
		t.Errorf("unexpected error with empty owners flag: %v", err)
	}
}

func TestNewCommand_EmptyNotificationsFlag_NoError(t *testing.T) {
	cmd := newCommand()
	// notifications flag left empty — should not attempt unmarshal
	preRunE := cmd.PersistentPreRunE
	if err := preRunE(cmd, []string{}); err != nil {
		t.Errorf("unexpected error with empty notifications flag: %v", err)
	}
}

func TestNewCommand_ImageFilterFlag(t *testing.T) {
	cmd := newCommand()
	if err := cmd.Flags().Set("image-filter", "mock-service,mongo"); err != nil {
		t.Errorf("unexpected error setting image-filter: %v", err)
	}
}
