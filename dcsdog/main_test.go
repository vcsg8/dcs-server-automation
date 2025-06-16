package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindDCSInUninstallKey(t *testing.T) {
	// Create a temporary directory for test
	tempDir, err := os.MkdirTemp("", "dcs-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Fatalf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create a mock DCS server exe
	serverExe := filepath.Join(tempDir, dcsServerExe)
	if err := os.WriteFile(serverExe, []byte("mock"), 0644); err != nil {
		t.Fatalf("Failed to create mock server exe: %v", err)
	}

	// Test cases
	tests := []struct {
		name     string
		keyName  string
		location string
		wantErr  bool
	}{
		{
			name:     "DCS World Server",
			keyName:  "DCS World Server_is1",
			location: tempDir,
			wantErr:  false,
		},
		{
			name:     "DCS World OpenBeta Server",
			keyName:  "DCS World OpenBeta Server_is1",
			location: tempDir,
			wantErr:  false,
		},
		{
			name:     "Invalid Key",
			keyName:  "Invalid Key",
			location: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock registry key would go here
			// Since we can't easily mock the Windows registry,
			// this is more of a placeholder for the test structure
			// In a real implementation, we'd use a mock registry
		})
	}
}

func TestUpdateMissionScripting(t *testing.T) {
	// Create a temporary directory for test
	tempDir, err := os.MkdirTemp("", "dcs-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Fatalf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create test cases
	tests := []struct {
		name     string
		content  string
		expected string
		wantErr  bool
	}{
		{
			name: "Basic sanitizeModule",
			content: `local function sanitizeModule(name)
	_G[name] = nil
	package.loaded[name] = nil
end

do
	sanitizeModule('os')
	sanitizeModule('io')
	sanitizeModule('lfs')
	_G['require'] = nil
	_G['loadlib'] = nil
	_G['package'] = nil
end`,
			expected: `-- local function sanitizeModule(name) -- Modified by dcsdog
-- 	_G[name] = nil -- Modified by dcsdog
-- 	package.loaded[name] = nil -- Modified by dcsdog
-- end -- Modified by dcsdog

-- do -- Modified by dcsdog
-- 	sanitizeModule('os') -- Modified by dcsdog
-- 	sanitizeModule('io') -- Modified by dcsdog
-- 	sanitizeModule('lfs') -- Modified by dcsdog
-- 	_G['require'] = nil -- Modified by dcsdog
-- 	_G['loadlib'] = nil -- Modified by dcsdog
-- 	_G['package'] = nil -- Modified by dcsdog
-- end -- Modified by dcsdog`,
			wantErr: false,
		},
		{
			name: "Already Modified",
			content: `-- local function sanitizeModule(name) -- Modified by dcsdog
	_G[name] = nil
	package.loaded[name] = nil
end`,
			expected: `-- local function sanitizeModule(name) -- Modified by dcsdog
	_G[name] = nil
	package.loaded[name] = nil
end`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			scriptPath := filepath.Join(tempDir, missionScriptingPath)
			if err := os.MkdirAll(filepath.Dir(scriptPath), 0755); err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}
			if err := os.WriteFile(scriptPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Run test
			_, err := updateMissionScripting(tempDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("updateMissionScripting() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check result
			if !tt.wantErr {
				content, err := os.ReadFile(scriptPath)
				if err != nil {
					t.Fatalf("Failed to read test file: %v", err)
				}
				actual := strings.ReplaceAll(string(content), "\r\n", "\n")
				expected := strings.ReplaceAll(tt.expected, "\r\n", "\n")
				if actual != expected {
					t.Errorf("updateMissionScripting() content = %q, want %q", actual, expected)
				}
			}
		})
	}
}

func TestIsProcessRunning(t *testing.T) {
	// Since we can't easily mock process information,
	// this is more of a placeholder for the test structure
	tests := []struct {
		name        string
		processName string
		want        bool
		wantErr     bool
	}{
		{
			name:        "Non-existent process",
			processName: "nonexistent_process.exe",
			want:        false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isProcessRunning(tt.processName)
			if (err != nil) != tt.wantErr {
				t.Errorf("isProcessRunning() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("isProcessRunning() = %v, want %v", got, tt.want)
			}
		})
	}
}
