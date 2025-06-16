package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

const (
	missionScriptingPath = "Scripts\\MissionScripting.lua"
	dcsServerExe         = "DCS_server.exe"
	serviceName          = "dcsdog"
)

var elog debug.Log

type dcsdogService struct{}

func (s *dcsdogService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	_ = elog.Info(1, "DCS Dog service started.")

	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := run(false) // Not interactive
			if err != nil {
				_ = elog.Error(1, fmt.Sprintf("Error in run loop: %v", err))
			}
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				_ = elog.Info(1, "DCS Dog service stopping.")
				return
			default:
				_ = elog.Error(1, fmt.Sprintf("Unexpected control request #%d", c.Cmd))
			}
		}
	}
}

// findDCSInstallPath searches for DCS install path, returning the path, the method used ("registry" or "process"), and an error
func findDCSInstallPath() (path string, method string, err error) {
	// Try to find from registry first
	path, err = findDCSInUninstallKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Uninstall`)
	if err == nil && path != "" {
		return path, "registry", nil
	}

	path, err = findDCSInUninstallKey(registry.LOCAL_MACHINE, `Software\Microsoft\Windows\CurrentVersion\Uninstall`)
	if err == nil && path != "" {
		return path, "registry", nil
	}

	// Fallback to finding from a running process
	path, err = findDCSFromProcess()
	if err == nil && path != "" {
		return path, "process", nil
	}

	return "", "", fmt.Errorf("DCS World installation not found")
}

// findDCSInUninstallKey searches for DCS World in the Uninstall registry key
func findDCSInUninstallKey(rootKey registry.Key, baseKey string) (string, error) {
	k, err := registry.OpenKey(rootKey, baseKey, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := k.Close(); err != nil {
			// Log or handle error, for now we can just print it
			fmt.Printf("warning: failed to close registry key: %v", err)
		}
	}()

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
			if err := appKey.Close(); err != nil {
				// Log or handle error, for now we can just print it
				fmt.Printf("warning: failed to close registry key: %v", err)
			}
			if err == nil && installLocation != "" {
				// Verify the path exists and contains DCS_server.exe, which could be in root or a bin/ subdir
				if _, err := os.Stat(filepath.Join(installLocation, "bin", dcsServerExe)); err == nil {
					return installLocation, nil
				}
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
			exeDir := filepath.Dir(exe)
			// The executable is often in a 'bin' subdirectory of the install root.
			if filepath.Base(exeDir) == "bin" {
				return filepath.Dir(exeDir), nil
			}
			return exeDir, nil
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

// updateMissionScripting updates the MissionScripting.lua file, returning true if a change was made.
func updateMissionScripting(dcsPath string) (bool, error) {
	scriptPath := filepath.Join(dcsPath, missionScriptingPath)
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return false, fmt.Errorf("failed to read MissionScripting.lua: %w", err)
	}

	if strings.Contains(string(content), "-- Modified by dcsdog") {
		return false, nil
	}

	lines := strings.Split(string(content), "\n")
	linesToComment := make(map[int]bool)

	// --- Pass 1: Find and mark the function definition block for commenting ---
	inFunctionBlock := false
	functionIndent := 0
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if !inFunctionBlock {
			if strings.Contains(trimmedLine, "local function sanitizeModule") {
				inFunctionBlock = true
				functionIndent = len(line) - len(strings.TrimLeft(line, " \t"))
				linesToComment[i] = true
			}
		} else {
			linesToComment[i] = true
			currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))
			if trimmedLine == "end" && currentIndent == functionIndent {
				inFunctionBlock = false
			}
		}
	}

	// --- Pass 2: Find and mark the sanitization do...end block for commenting ---
	inPotentialBlock := false
	isSanitizeBlock := false
	blockStart := -1
	doIndent := 0
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if !inPotentialBlock {
			if trimmedLine == "do" {
				inPotentialBlock = true
				blockStart = i
				doIndent = len(line) - len(strings.TrimLeft(line, " \t"))
			}
		} else {
			if strings.Contains(trimmedLine, "sanitizeModule") || strings.Contains(trimmedLine, "_G") {
				isSanitizeBlock = true
			}
			currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))
			if trimmedLine == "end" && currentIndent == doIndent {
				if isSanitizeBlock {
					// Mark the entire block for commenting
					for j := blockStart; j <= i; j++ {
						linesToComment[j] = true
					}
				}
				// Reset for next potential block
				inPotentialBlock = false
				isSanitizeBlock = false
			}
		}
	}

	// --- Final Pass: Apply the comments ---
	for i, line := range lines {
		if linesToComment[i] {
			lines[i] = "-- " + line + " -- Modified by dcsdog"
		}
	}

	finalContent := strings.Join(lines, "\n")
	if err := os.WriteFile(scriptPath, []byte(finalContent), 0644); err != nil {
		return true, fmt.Errorf("failed to write updated MissionScripting.lua: %w", err)
	}

	return true, nil
}

// run is the main application logic
func run(isInteractive bool) error {
	// Find DCS installation path
	dcsPath, method, err := findDCSInstallPath()
	if err != nil {
		return err
	}
	if isInteractive {
		if method == "process" {
			fmt.Println("Found DCS World installation via running process (fallback).")
		} else {
			fmt.Println("Found DCS World installation via registry.")
		}
		fmt.Printf("DCS Path: %s\n", dcsPath)
	}

	// Update MissionScripting.lua
	scriptUpdated, err := updateMissionScripting(dcsPath)
	if err != nil {
		return err
	}

	// If the script was just updated, we may need to restart the server to apply it.
	if scriptUpdated {
		if isInteractive {
			fmt.Println("MissionScripting.lua was updated.")
		}
		// Check if DCS server is running
		isRunning, err := isProcessRunning(dcsServerExe)
		if err != nil {
			return err
		}

		// Restart DCS server if it's running to apply changes
		if isRunning {
			if isInteractive {
				fmt.Println("DCS server is running, restarting to apply changes...")
			}
			if err := restartDCServer(dcsPath); err != nil {
				return err
			}
			_ = elog.Info(1, "DCS Server restarted to apply script changes.")
		}
	}

	return nil
}

func main() {
	isWindowsService, err := svc.IsWindowsService()
	if err != nil {
		fmt.Printf("failed to determine if session is a Windows service: %v", err)
		os.Exit(1)
	}

	if isWindowsService {
		elog, err = eventlog.Open(serviceName)
		if err != nil {
			return
		}
		defer elog.Close()

		err = svc.Run(serviceName, &dcsdogService{})
		if err != nil {
			_ = elog.Error(1, fmt.Sprintf("%s service failed: %v", serviceName, err))
			return
		}
		return
	}

	if err := run(true); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	fmt.Println("DCS Dog check complete.")
}
