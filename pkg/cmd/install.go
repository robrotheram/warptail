package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"warptail/pkg/utils"

	"github.com/urfave/cli/v3"
)

//go:embed config.sample.yaml
var configFile []byte

const serviceFile = `[Unit]
Description=Warptail Service
After=network.target

[Service]
Type=simple
ExecStart=%s
Restart=on-failure
RestartSec=5
Environment="CONFIG_PATH=%s"
Environment="HOME=%s"

[Install]
WantedBy=multi-user.target
`

const servicePath = "/etc/systemd/system/warptail.service"
const serviceName = "warptail.service"

func InstallService(ctx context.Context, cmd *cli.Command) error {

	configPath, err := filepath.Abs(utils.ConfigPath)

	if err != nil {
		return fmt.Errorf("failed path to config file: %w", err)
	}
	configBaseDir := filepath.Dir(configPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("Installing config file to", configPath)
		if err := os.WriteFile(configPath, configFile, 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
	}

	targetPath, _ := CurrentExecutable()

	// Write the embedded service file to /etc/systemd/system/
	fmt.Println("Installing service file to", servicePath)

	if err := os.WriteFile(servicePath, []byte(fmt.Sprintf(serviceFile, targetPath, configPath, configBaseDir)), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Reload systemd manager configuration
	fmt.Println("Reloading systemd daemon...")
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Enable the service to start on boot
	fmt.Println("Enabling the service...")
	if err := exec.Command("systemctl", "enable", serviceName).Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	fmt.Println("Starting the service...")
	if err := exec.Command("systemctl", "start", serviceName).Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	fmt.Println("Service installed and enabled successfully.")
	return nil
}

func UninstallService(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("Stopping the service...")
	if err := exec.Command("systemctl", "stop", serviceName).Run(); err != nil {
		fmt.Printf("Warning: failed to stop service: %v\n", err)
	}

	fmt.Println("Disabling the service...")
	if err := exec.Command("systemctl", "disable", serviceName).Run(); err != nil {
		fmt.Printf("Warning: failed to disable service: %v\n", err)
	}

	fmt.Println("Removing service file from", servicePath)
	if err := os.Remove(servicePath); err != nil {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	fmt.Println("Reloading systemd daemon...")
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	fmt.Println("Service uninstalled successfully.")
	return nil
}
