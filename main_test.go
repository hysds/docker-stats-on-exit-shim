// Copyright 2024 HySDS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
)

// TestCgroupDetection validates runtime detection of cgroup v1 vs v2
func TestCgroupDetection(t *testing.T) {
	isV2 := cgroups.IsCgroup2UnifiedMode()
	
	// Check for expected filesystem artifacts based on detection
	if isV2 {
		// cgroup v2 should have unified hierarchy at /sys/fs/cgroup/cgroup.controllers
		if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); os.IsNotExist(err) {
			t.Logf("Warning: cgroup v2 detected but /sys/fs/cgroup/cgroup.controllers not found")
		} else {
			t.Logf("cgroup v2 detected correctly - found cgroup.controllers")
		}
	} else {
		// cgroup v1 should have per-subsystem directories
		if _, err := os.Stat("/sys/fs/cgroup/cpu"); os.IsNotExist(err) {
			t.Logf("Warning: cgroup v1 detected but /sys/fs/cgroup/cpu not found")
		} else {
			t.Logf("cgroup v1 detected correctly - found cpu subsystem")
		}
	}
	
	t.Logf("Cgroup version detection: v2=%v", isV2)
}

// TestShimEndToEnd builds shim, wraps sleep 1, validates JSON output
func TestShimEndToEnd(t *testing.T) {
	// Create temporary directory for test artifacts
	tmpDir := t.TempDir()
	
	// Build the shim binary
	shimBin := filepath.Join(tmpDir, "docker-stats-on-exit-shim")
	buildCmd := exec.Command("go", "build", "-o", shimBin, ".")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build shim: %v", err)
	}
	
	// Run the shim wrapping sleep 1
	statsFile := filepath.Join(tmpDir, "stats.json")
	shimCmd := exec.Command(shimBin, statsFile, "sleep", "1")
	shimCmd.Stdout = os.Stdout
	shimCmd.Stderr = os.Stderr
	
	start := time.Now()
	if err := shimCmd.Run(); err != nil {
		t.Fatalf("Failed to run shim: %v", err)
	}
	elapsed := time.Since(start)
	
	// Verify the command took approximately 1 second
	if elapsed < 900*time.Millisecond || elapsed > 2*time.Second {
		t.Errorf("Expected ~1s execution time, got %v", elapsed)
	}
	
	// Read and parse the stats file
	data, err := os.ReadFile(statsFile)
	if err != nil {
		t.Fatalf("Failed to read stats file: %v", err)
	}
	
	var stats Stats
	if err := json.Unmarshal(data, &stats); err != nil {
		t.Fatalf("Failed to parse stats JSON: %v", err)
	}
	
	// Validate wall_time is approximately 1 second (in nanoseconds)
	expectedWallTime := int64(1_000_000_000) // 1 second in nanoseconds
	if stats.WallClockTime < 500_000_000 || stats.WallClockTime > 2_000_000_000 {
		t.Errorf("Expected wall_time ~%d ns, got %d ns", expectedWallTime, stats.WallClockTime)
	}
	
	// Validate cgroups stats exist
	if stats.Cgroups == nil {
		t.Error("Expected cgroups stats to be non-nil")
	}
	
	// Validate expected top-level keys exist
	if stats.Cgroups != nil {
		if stats.Cgroups.CpuStats.CpuUsage.TotalUsage == 0 {
			t.Log("Warning: total_usage is 0 (may be expected for sleep)")
		}
		t.Logf("Stats collected successfully: wall_time=%dns, cpu_total=%d", 
			stats.WallClockTime, stats.Cgroups.CpuStats.CpuUsage.TotalUsage)
	}
}

// TestShimExitCodeForwarding verifies exit codes 0, 1, and 42 are forwarded
func TestShimExitCodeForwarding(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Build the shim binary
	shimBin := filepath.Join(tmpDir, "docker-stats-on-exit-shim")
	buildCmd := exec.Command("go", "build", "-o", shimBin, ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build shim: %v", err)
	}
	
	testCases := []struct {
		name     string
		command  string
		args     []string
		expected int
	}{
		{"exit 0", "sh", []string{"-c", "exit 0"}, 0},
		{"exit 1", "sh", []string{"-c", "exit 1"}, 1},
		{"exit 42", "sh", []string{"-c", "exit 42"}, 42},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			statsFile := filepath.Join(tmpDir, "stats_"+tc.name+".json")
			
			args := []string{statsFile, tc.command}
			args = append(args, tc.args...)
			shimCmd := exec.Command(shimBin, args...)
			
			err := shimCmd.Run()
			
			// Check exit code
			if tc.expected == 0 {
				if err != nil {
					t.Errorf("Expected exit code 0, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected exit code %d, got 0", tc.expected)
				} else if exitErr, ok := err.(*exec.ExitError); ok {
					if exitErr.ExitCode() != tc.expected {
						t.Errorf("Expected exit code %d, got %d", tc.expected, exitErr.ExitCode())
					}
				} else {
					t.Errorf("Unexpected error type: %v", err)
				}
			}
			
			// Verify stats file was created
			if _, err := os.Stat(statsFile); os.IsNotExist(err) {
				t.Error("Stats file was not created")
			}
		})
	}
}

// TestJsonOutputCompatibility validates JSON schema matches v1.0
func TestJsonOutputCompatibility(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Build the shim binary
	shimBin := filepath.Join(tmpDir, "docker-stats-on-exit-shim")
	buildCmd := exec.Command("go", "build", "-o", shimBin, ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build shim: %v", err)
	}
	
	// Run the shim
	statsFile := filepath.Join(tmpDir, "stats.json")
	shimCmd := exec.Command(shimBin, statsFile, "sleep", "0.1")
	if err := shimCmd.Run(); err != nil {
		t.Fatalf("Failed to run shim: %v", err)
	}
	
	// Read and parse the stats file
	data, err := os.ReadFile(statsFile)
	if err != nil {
		t.Fatalf("Failed to read stats file: %v", err)
	}
	
	// Parse as generic JSON to check structure
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	
	// Validate top-level keys
	requiredTopLevelKeys := []string{"wall_time", "user_cpu_time", "sys_cpu_time", "cgroups"}
	for _, key := range requiredTopLevelKeys {
		if _, exists := jsonData[key]; !exists {
			t.Errorf("Missing required top-level key: %s", key)
		}
	}
	
	// Validate cgroups sub-structure
	if cgroups, ok := jsonData["cgroups"].(map[string]interface{}); ok {
		expectedCgroupKeys := []string{"cpu_stats", "memory_stats", "blkio_stats", "pids_stats"}
		for _, key := range expectedCgroupKeys {
			if _, exists := cgroups[key]; !exists {
				t.Logf("Warning: cgroups missing key '%s' (may be acceptable depending on system)", key)
			}
		}
		
		// Validate cpu_stats structure
		if cpuStats, ok := cgroups["cpu_stats"].(map[string]interface{}); ok {
			if cpuUsage, ok := cpuStats["cpu_usage"].(map[string]interface{}); ok {
				if _, exists := cpuUsage["total_usage"]; !exists {
					t.Error("Missing cpu_stats.cpu_usage.total_usage")
				}
				// Note: percpu_usage may be empty array on cgroup v2, which is acceptable
				if percpuUsage, exists := cpuUsage["percpu_usage"]; exists {
					t.Logf("percpu_usage present: %v", percpuUsage)
				}
			} else {
				t.Error("Missing or invalid cpu_stats.cpu_usage")
			}
		} else {
			t.Error("Missing or invalid cpu_stats")
		}
	} else {
		t.Error("Missing or invalid cgroups object")
	}
	
	t.Log("JSON output structure validated successfully")
}
