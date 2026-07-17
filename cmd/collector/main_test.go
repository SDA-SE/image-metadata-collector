package main

import (
	"reflect"
	"testing"

	"github.com/SDA-SE/image-metadata-collector/internal/config"
	"github.com/spf13/cobra"
)

func TestNewCommand_ReturnsCommand(t *testing.T) {
	cmd := newCommand()
	if cmd == nil {
		t.Fatal("expected non-nil command")
		return
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
		"image-notification-rules",
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
		"project",
		"http-header",
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
		"image-notification-rules",
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

func TestNewCommand_ValidImageNotificationRulesFlag(t *testing.T) {
	cmd := newCommand()
	if err := cmd.PersistentFlags().Set("image-notification-rules", `[{"image":"^ghcr\\.io/acme/private-app:.*$","notifications":{"slack":["#channel"],"emails":["test@example.com"],"ms_teams":["team-a"]}}]`); err != nil {
		t.Fatalf("failed to set image-notification-rules flag: %v", err)
	}
	preRunE := cmd.PersistentPreRunE
	if err := preRunE(cmd, []string{}); err != nil {
		t.Errorf("unexpected error with valid image-notification-rules flag: %v", err)
	}
}

func TestNewCommand_ValidNegatedImageNotificationRulesFlag(t *testing.T) {
	cmd := newCommand()
	if err := cmd.PersistentFlags().Set("image-notification-rules", `[{"image":"!^ghcr\\.io/acme/private-app:.*$","notifications":{"slack":["#channel"],"emails":["test@example.com"],"ms_teams":["team-a"]}}]`); err != nil {
		t.Fatalf("failed to set image-notification-rules flag: %v", err)
	}
	preRunE := cmd.PersistentPreRunE
	if err := preRunE(cmd, []string{}); err != nil {
		t.Errorf("unexpected error with valid negated image-notification-rules flag: %v", err)
	}
}

func TestNewCommand_InvalidImageNotificationRulesFlag(t *testing.T) {
	cmd := newCommand()
	if err := cmd.PersistentFlags().Set("image-notification-rules", "not-valid-json"); err != nil {
		t.Fatalf("failed to set image-notification-rules flag: %v", err)
	}

	preRunE := cmd.PersistentPreRunE
	if err := preRunE(cmd, []string{}); err == nil {
		t.Error("expected error for invalid image-notification-rules JSON, got nil")
	}
}

func TestNewCommand_InvalidImageNotificationRulesRegexFlag(t *testing.T) {
	cmd := newCommand()
	if err := cmd.PersistentFlags().Set("image-notification-rules", `[{"image":"(","notifications":{"slack":["#channel"]}}]`); err != nil {
		t.Fatalf("failed to set image-notification-rules flag: %v", err)
	}

	preRunE := cmd.PersistentPreRunE
	if err := preRunE(cmd, []string{}); err == nil {
		t.Error("expected error for invalid image-notification-rules regex, got nil")
	}
}

func TestNewCommand_InvalidNegatedImageNotificationRulesRegexFlag(t *testing.T) {
	cmd := newCommand()
	if err := cmd.PersistentFlags().Set("image-notification-rules", `[{"image":"!","notifications":{"slack":["#channel"]}}]`); err != nil {
		t.Fatalf("failed to set image-notification-rules flag: %v", err)
	}

	preRunE := cmd.PersistentPreRunE
	if err := preRunE(cmd, []string{}); err == nil {
		t.Error("expected error for invalid negated image-notification-rules regex, got nil")
	}
}

func TestNewCommand_EmptyImageNotificationRulesFlag_NoError(t *testing.T) {
	cmd := newCommand()
	preRunE := cmd.PersistentPreRunE
	if err := preRunE(cmd, []string{}); err != nil {
		t.Errorf("unexpected error with empty image-notification-rules flag: %v", err)
	}
}

func TestNewCommand_ImageFilterFlag(t *testing.T) {
	cmd := newCommand()
	if err := cmd.Flags().Set("image-filter", "mock-service,mongo"); err != nil {
		t.Errorf("unexpected error setting image-filter: %v", err)
	}
}

func TestNewCommand_HTTPHeadersFromEnvSingleValue(t *testing.T) {
	t.Setenv("COLLECTOR_HTTP_HEADER", "Authorization:Bearer token")

	_, cfg := runPersistentPreRun(t)

	expected := []string{"Authorization:Bearer token"}
	if !reflect.DeepEqual(cfg.HTTPHeaders, expected) {
		t.Fatalf("expected HTTPHeaders=%v, got %v", expected, cfg.HTTPHeaders)
	}
}

func TestNewCommand_HTTPHeadersFromEnvMultipleValues(t *testing.T) {
	t.Setenv("COLLECTOR_HTTP_HEADER", "Authorization:Bearer token,Content-Type:application/json")

	_, cfg := runPersistentPreRun(t)

	expected := []string{"Authorization:Bearer token", "Content-Type:application/json"}
	if !reflect.DeepEqual(cfg.HTTPHeaders, expected) {
		t.Fatalf("expected HTTPHeaders=%v, got %v", expected, cfg.HTTPHeaders)
	}
}

func TestNewCommand_HTTPHeadersFromEnvTrimsWhitespace(t *testing.T) {
	t.Setenv("COLLECTOR_HTTP_HEADER", "Authorization:Bearer token, Content-Type:application/json")

	_, cfg := runPersistentPreRun(t)

	expected := []string{"Authorization:Bearer token", "Content-Type:application/json"}
	if !reflect.DeepEqual(cfg.HTTPHeaders, expected) {
		t.Fatalf("expected HTTPHeaders=%v, got %v", expected, cfg.HTTPHeaders)
	}
}

func TestNewCommand_HTTPHeadersFromCLIRepeatedFlags(t *testing.T) {
	cmd, cfg := newCommandWithConfig()

	if err := cmd.PersistentFlags().Set("http-header", "Authorization:Bearer token"); err != nil {
		t.Fatalf("failed to set first http-header flag: %v", err)
	}
	if err := cmd.PersistentFlags().Set("http-header", "Content-Type:application/json"); err != nil {
		t.Fatalf("failed to set second http-header flag: %v", err)
	}

	if err := cmd.PersistentPreRunE(cmd, []string{}); err != nil {
		t.Fatalf("unexpected error running pre-run: %v", err)
	}

	expected := []string{"Authorization:Bearer token", "Content-Type:application/json"}
	if !reflect.DeepEqual(cfg.HTTPHeaders, expected) {
		t.Fatalf("expected HTTPHeaders=%v, got %v", expected, cfg.HTTPHeaders)
	}
}

func runPersistentPreRun(t *testing.T) (*cobra.Command, *config.Config) {
	t.Helper()

	cmd, cfg := newCommandWithConfig()
	if err := cmd.PersistentPreRunE(cmd, []string{}); err != nil {
		t.Fatalf("unexpected error running pre-run: %v", err)
	}

	return cmd, cfg
}
