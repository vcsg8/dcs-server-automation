package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/v3/process"
	"golang.org/x/sys/windows/registry"
)

const (
	missionScriptingPath = "Scripts\\MissionScripting.lua"
	dcsServerExe         = "DCS_server.exe"
)

// findDCSInstallPath finds the DCS World installation path using the Uninstall registry keys
func findDCSInstallPath() (string, error) {
	// Try both 32-bit and 64-bit uninstall keys
	uninstallKeys := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}

	for _, baseKey := range uninstallKeys {
		// Try HKLM first
		if path, err := findDCSInUninstallKey(registry.LOCAL_MACHINE, baseKey); err == nil {
			return path, nil
		}

		// Then try HKCU
		if path, err := findDCSInUninstallKey(registry.CURRENT_USER, baseKey); err == nil {
			return path, nil
		}
	}

	// Fallback to process check if DCS is running
	if path, err := findDCSFromProcess(); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("could not find DCS World installation. Please ensure it is installed")
}

// findDCSInUninstallKey searches for DCS World in the Uninstall registry key
func findDCSInUninstallKey(rootKey registry.Key, baseKey string) (string, error) {
	k, err := registry.OpenKey(rootKey, baseKey, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return "", err
	}
	defer k.Close()

	// Get all subkeys
	subKeys, err := k.ReadSubKeyNames(0)
	if err != nil {
		return "", err
	}

	// Look for either DCS World Server key pattern
	for _, subKey := range subKeys {
		lowerKey := strings.ToLower(subKey)
		if strings.Contains(lowerKey, "dcs world server_is1") ||
			strings.Contains(lowerKey, "dcs world openbeta server_is1") {
			appKey, err := registry.OpenKey(rootKey, baseKey+"\\"+subKey, registry.QUERY_VALUE)
			if err != nil {
				continue
			}

			// Get the installation path
			installLocation, _, err := appKey.GetStringValue("InstallLocation")
			appKey.Close()
			if err == nil && installLocation != "" {
				// Verify the path exists and contains DCS_server.exe
				if _, err := os.Stat(filepath.Join(installLocation, dcsServerExe)); err == nil {
					return installLocation, nil
				}
			}
		}
	}

	return "", fmt.Errorf("DCS World not found in uninstall registry")
}

// findDCSFromProcess attempts to find DCS World path from running process
func findDCSFromProcess() (string, error) {
	processes, err := process.Processes()
	if err != nil {
		return "", err
	}

	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			continue
		}
		if strings.EqualFold(name, dcsServerExe) {
			exe, err := p.Exe()
			if err != nil {
				continue
			}
			return filepath.Dir(exe), nil
		}
	}

	return "", fmt.Errorf("DCS World process not found")
}

// isProcessRunning checks if a process with the given name is running
func isProcessRunning(processName string) (bool, error) {
	processes, err := process.Processes()
	if err != nil {
		return false, fmt.Errorf("failed to get processes: %v", err)
	}

	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			continue
		}
		if strings.EqualFold(name, processName) {
			return true, nil
		}
	}
	return false, nil
}

// restartDCServer restarts the DCS server process
func restartDCServer(dcsPath string) error {
	// Kill existing process
	processes, err := process.Processes()
	if err != nil {
		return fmt.Errorf("failed to get processes: %v", err)
	}

	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			continue
		}
		if strings.EqualFold(name, dcsServerExe) {
			if err := p.Kill(); err != nil {
				return fmt.Errorf("failed to kill DCS server: %v", err)
			}
		}
	}

	// Start new process
	serverPath := filepath.Join(dcsPath, dcsServerExe)
	cmd := exec.Command(serverPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start DCS server: %v", err)
	}

	return nil
}

// updateMissionScripting updates the MissionScripting.lua file
func updateMissionScripting(dcsPath string) error {
	scriptPath := filepath.Join(dcsPath, missionScriptingPath)

	// Read the file
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read MissionScripting.lua: %v", err)
	}

	// If file was modified by dcsdog, skip
	if strings.Contains(string(content), "-- Modified by dcsdog") {
		return nil
	}

	// Update content
	lines := strings.Split(string(content), "\n")
	var newLines []string
	inDoBlock := false
	doBlockIndent := 0

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		// Check for sanitizeModule function definition
		if strings.Contains(trimmedLine, "local function sanitizeModule") {
			newLines = append(newLines, "-- "+line+" -- Modified by dcsdog")
			continue
		}

		// Check for do block start
		if trimmedLine == "do" {
			inDoBlock = true
			doBlockIndent = indent
			newLines = append(newLines, "-- "+line+" -- Modified by dcsdog")
			continue
		}

		// If we're in a do block, check for sanitizeModule calls or _G modifications
		if inDoBlock {
			if indent <= doBlockIndent && trimmedLine != "" {
				// End of do block
				inDoBlock = false
				newLines = append(newLines, line)
			} else if strings.Contains(trimmedLine, "sanitizeModule") ||
				strings.Contains(trimmedLine, "_G[") ||
				strings.Contains(trimmedLine, "_G.") {
				newLines = append(newLines, "-- "+line+" -- Modified by dcsdog")
			} else {
				newLines = append(newLines, line)
			}
		} else {
			newLines = append(newLines, line)
		}
	}

	// Write updated content
	if err := os.WriteFile(scriptPath, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
		return fmt.Errorf("failed to write updated MissionScripting.lua: %v", err)
	}

	return nil
}

func main() {
	// Find DCS installation path
	dcsPath, err := findDCSInstallPath()
	if err != nil {
		fmt.Printf("Error finding DCS installation: %v\n", err)
		os.Exit(1)
	}

	// Update MissionScripting.lua
	if err := updateMissionScripting(dcsPath); err != nil {
		fmt.Printf("Error updating MissionScripting.lua: %v\n", err)
		os.Exit(1)
	}

	// Check if DCS server is running
	isRunning, err := isProcessRunning(dcsServerExe)
	if err != nil {
		fmt.Printf("Error checking DCS server status: %v\n", err)
		os.Exit(1)
	}

	// Restart DCS server if it's running
	if isRunning {
		if err := restartDCServer(dcsPath); err != nil {
			fmt.Printf("Error restarting DCS server: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("DCS server has been restarted")
	} else {
		fmt.Println("DCS server is not running")
	}
}
