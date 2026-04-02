package systemd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	serviceName = "streamchop.service"
	systemdDir  = "/etc/systemd/system"
	localDir    = ".streamchop"
)

// Install writes the watchdog script and systemd service file, then enables
// and starts the service. Requires sudo for systemctl and writing to /etc.
func Install(workDir string) error {
	distFile := filepath.Join(workDir, "docker-compose.dist.yml")
	if _, err := os.Stat(distFile); os.IsNotExist(err) {
		return fmt.Errorf("docker-compose.dist.yml not found in %s — run 'streamchop setup' first", workDir)
	}

	servicePath := filepath.Join(systemdDir, serviceName)
	if _, err := os.Stat(servicePath); err == nil {
		return fmt.Errorf("service %s is already installed — run 'streamchop uninstall' first", serviceName)
	}

	// Create .streamchop/ dir for watchdog
	scriptDir := filepath.Join(workDir, localDir)
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return fmt.Errorf("create %s: %w", scriptDir, err)
	}

	// Write watchdog script
	watchdogPath := filepath.Join(scriptDir, "watchdog.sh")
	if err := os.WriteFile(watchdogPath, WatchdogScript, 0755); err != nil {
		return fmt.Errorf("write watchdog: %w", err)
	}

	// Render service template
	rendered := strings.ReplaceAll(ServiceTemplate, "__PWD__", workDir)

	// Write service file to temp location, then sudo mv
	tmpService := filepath.Join(scriptDir, serviceName)
	if err := os.WriteFile(tmpService, []byte(rendered), 0644); err != nil {
		return fmt.Errorf("write service file: %w", err)
	}

	// Move to systemd dir with sudo
	if err := sudoRun("mv", tmpService, servicePath); err != nil {
		return fmt.Errorf("install service file: %w", err)
	}

	// Enable and start
	if err := sudoRun("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}
	if err := sudoRun("systemctl", "enable", serviceName); err != nil {
		return fmt.Errorf("enable service: %w", err)
	}
	if err := sudoRun("systemctl", "start", serviceName); err != nil {
		return fmt.Errorf("start service: %w", err)
	}

	fmt.Println("Service installed and started.")
	return nil
}

// Uninstall stops, disables, and removes the systemd service.
func Uninstall() error {
	servicePath := filepath.Join(systemdDir, serviceName)
	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		return fmt.Errorf("service %s is not installed", serviceName)
	}

	fmt.Println("Stopping and removing service...")

	_ = sudoRun("systemctl", "stop", serviceName)
	_ = sudoRun("systemctl", "disable", serviceName)

	if err := sudoRun("rm", servicePath); err != nil {
		return fmt.Errorf("remove service file: %w", err)
	}
	if err := sudoRun("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	fmt.Println("Service uninstalled.")
	return nil
}

// Status prints the current status of the systemd service.
func Status() error {
	cmd := exec.Command("sudo", "systemctl", "status", serviceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	_ = cmd.Run()
	return nil
}

func sudoRun(name string, args ...string) error {
	cmdArgs := append([]string{name}, args...)
	cmd := exec.Command("sudo", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// RenderTemplate replaces __PWD__ in the service template with the given path.
func RenderTemplate(workDir string) string {
	return strings.ReplaceAll(ServiceTemplate, "__PWD__", workDir)
}
