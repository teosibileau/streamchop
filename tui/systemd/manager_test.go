package systemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	rendered := RenderTemplate("/opt/streamchop")

	if strings.Contains(rendered, "__PWD__") {
		t.Error("template still contains __PWD__ placeholder")
	}
	if !strings.Contains(rendered, "WorkingDirectory=/opt/streamchop") {
		t.Error("expected WorkingDirectory=/opt/streamchop")
	}
	if !strings.Contains(rendered, "ExecStart=/opt/streamchop/.streamchop/watchdog.sh") {
		t.Error("expected ExecStart with correct watchdog path")
	}
}

func TestServiceTemplateIsValid(t *testing.T) {
	if !strings.Contains(ServiceTemplate, "[Unit]") {
		t.Error("service template missing [Unit] section")
	}
	if !strings.Contains(ServiceTemplate, "[Service]") {
		t.Error("service template missing [Service] section")
	}
	if !strings.Contains(ServiceTemplate, "[Install]") {
		t.Error("service template missing [Install] section")
	}
}

func TestWatchdogScriptIsValid(t *testing.T) {
	script := string(WatchdogScript)
	if !strings.HasPrefix(script, "#!/bin/bash") {
		t.Error("watchdog script missing shebang")
	}
	if !strings.Contains(script, "docker compose -f docker-compose.dist.yml") {
		t.Error("watchdog script missing docker compose command")
	}
	if !strings.Contains(script, "systemd-notify READY=1") {
		t.Error("watchdog script missing READY notification")
	}
}

func TestInstallFailsWithoutDistFile(t *testing.T) {
	tmpDir := t.TempDir()
	err := Install(tmpDir)
	if err == nil {
		t.Fatal("expected error when docker-compose.dist.yml is missing")
	}
	if !strings.Contains(err.Error(), "docker-compose.dist.yml not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInstallWritesWatchdog(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping on root — would actually install")
	}

	tmpDir := t.TempDir()

	distPath := filepath.Join(tmpDir, "docker-compose.dist.yml")
	if err := os.WriteFile(distPath, []byte("services: {}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Install will fail at the sudo step, but should write watchdog first
	_ = Install(tmpDir)

	watchdogPath := filepath.Join(tmpDir, ".streamchop", "watchdog.sh")
	info, err := os.Stat(watchdogPath)
	if err != nil {
		t.Fatalf("watchdog not written: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("watchdog should be executable")
	}
}
