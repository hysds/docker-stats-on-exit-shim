# Testing Guide for docker-stats-on-exit-shim v2.0

## Overview

This guide shows you how to test the v2.0 changes and verify compatibility with your HySDS codebase.

## Prerequisites

- Go 1.22+ installed
- Python 3.6+ installed
- Access to both cgroup v1 and v2 hosts (optional but recommended)

## Test 1: Build and Unit Tests

### Build the shim

```bash
cd /Users/mcayanan/git/docker-stats-on-exit-shim

# Download dependencies
go mod download
go mod tidy

# Build
go build -o docker-stats-on-exit-shim .
```

### Run Go unit tests

```bash
# Run all tests
go test -v ./...

# Run specific test
go test -v -run TestCgroupDetection
go test -v -run TestShimEndToEnd
go test -v -run TestShimExitCodeForwarding
go test -v -run TestJsonOutputCompatibility
```

**Expected output:**
- All 4 tests should pass
- You'll see which cgroup version was detected
- JSON structure validation confirms compatibility

## Test 2: Manual Functional Test

### Basic execution test

```bash
cd /Users/mcayanan/git/docker-stats-on-exit-shim

# Run the shim wrapping a simple command
./docker-stats-on-exit-shim /tmp/test_stats.json sleep 2

# Check stderr output - should show cgroup version detected
# Example: "docker-stats-on-exit-shim: using cgroup v1"
# or: "docker-stats-on-exit-shim: using cgroup v2 (path: /)"

# Examine the output
cat /tmp/test_stats.json | python3 -m json.tool
```

**What to verify:**
- File `/tmp/test_stats.json` is created
- JSON is valid and well-formed
- Contains keys: `wall_time`, `user_cpu_time`, `sys_cpu_time`, `cgroups`
- `wall_time` is approximately 2,000,000,000 nanoseconds (2 seconds)

## Test 3: HySDS Compatibility Test

This validates that v2.0 output is compatible with HySDS job_worker.py:

```bash
cd /Users/mcayanan/git/docker-stats-on-exit-shim

# Run the compatibility test script
python3 test_hysds_compatibility.py
```

**What this tests:**
1. Builds the shim
2. Runs it with `sleep 1`
3. Validates JSON structure matches HySDS expectations
4. Compares against `@/Users/mcayanan/git/hysds/test/examples/_docker_stats.json`
5. Simulates how `job_worker.py` loads the stats

**Expected output:**
```
======================================================================
docker-stats-on-exit-shim v2.0 - HySDS Compatibility Test
======================================================================
Building docker-stats-on-exit-shim...
✓ Build successful

Running shim with sleep 1...
Shim output: docker-stats-on-exit-shim: using cgroup v1

✓ Loaded HySDS example stats for reference

======================================================================
Validation Tests
======================================================================

Top-level keys:
✓ All required top-level keys present: ['wall_time', 'user_cpu_time', 'sys_cpu_time', 'cgroups']

Cgroups structure:
✓ All cgroups sections present: ['cpu_stats', 'memory_stats', 'blkio_stats', 'pids_stats']
✓ cpu_stats structure valid

Data types:
✓ All data types correct

Value ranges:
✓ wall_time is reasonable: 1.00 seconds
✓ CPU times are non-negative

HySDS loading:
✓ Successfully loaded stats into usage_stats list

======================================================================
Summary
======================================================================
Tests passed: 5/5

✓ ALL TESTS PASSED - v2.0 is compatible with HySDS!
```

## Test 4: Exit Code Forwarding

Verify that exit codes are properly forwarded:

```bash
cd /Users/mcayanan/git/docker-stats-on-exit-shim

# Test exit code 0 (success)
./docker-stats-on-exit-shim /tmp/exit0.json sh -c "exit 0"
echo "Exit code: $?"  # Should be 0

# Test exit code 1 (failure)
./docker-stats-on-exit-shim /tmp/exit1.json sh -c "exit 1"
echo "Exit code: $?"  # Should be 1

# Test exit code 42 (custom)
./docker-stats-on-exit-shim /tmp/exit42.json sh -c "exit 42"
echo "Exit code: $?"  # Should be 42
```

## Test 5: Integration with HySDS (Manual)

### Simulate HySDS job_worker.py behavior

```python
#!/usr/bin/env python3
"""Simulate how HySDS job_worker.py processes docker stats."""

import json
import os
import subprocess

# Run the shim
stats_file = "/tmp/hysds_test_stats.json"
subprocess.run([
    "./docker-stats-on-exit-shim",
    stats_file,
    "sleep", "1"
])

# Simulate find_usage_stats() - it searches for _docker_stats.json files
# For this test, we'll just load the file directly
def find_usage_stats(work_dir):
    """Simulate HySDS find_usage_stats function."""
    for root, dirs, files in os.walk(work_dir):
        for file in files:
            if file.endswith("_docker_stats.json"):
                yield os.path.join(root, file)

# Simulate job_worker.py stats collection (lines 1554-1565)
job = {"job_info": {"metrics": {"usage_stats": []}}}

# This is exactly what job_worker.py does:
usage_stats = {}
with open(stats_file) as f:
    try:
        usage_stats = json.load(f)
    except Exception as e:
        print(f"Failed to load usage stats: {e}")
        exit(1)

if len(usage_stats) > 0:
    job["job_info"]["metrics"]["usage_stats"].append(usage_stats)

# Verify
print("✓ Successfully loaded stats into job metrics")
print(f"  Number of stats entries: {len(job['job_info']['metrics']['usage_stats'])}")
print(f"  Wall time: {usage_stats['wall_time'] / 1_000_000_000:.2f} seconds")
print(f"  User CPU: {usage_stats['user_cpu_time'] / 1_000_000_000:.6f} seconds")
print(f"  Sys CPU: {usage_stats['sys_cpu_time'] / 1_000_000_000:.6f} seconds")
print(f"  Cgroups sections: {list(usage_stats['cgroups'].keys())}")
```

Save this as `test_hysds_integration.py` and run:

```bash
python3 test_hysds_integration.py
```

## Test 6: Docker Container Test (cgroup v1)

Test the shim inside a Docker container:

```bash
cd /Users/mcayanan/git/docker-stats-on-exit-shim

# Build for Linux if on Mac
GOOS=linux GOARCH=amd64 go build -o docker-stats-on-exit-shim .

# Run in Docker container
docker run --rm \
    -v $(pwd)/docker-stats-on-exit-shim:/shim:ro \
    -v /tmp/docker-test:/out \
    ubuntu:20.04 \
    /shim /out/container_stats.json /bin/sleep 2

# Check the output
cat /tmp/docker-test/container_stats.json | python3 -m json.tool
```

## Test 7: Compare v1 and v2 Output (if you have v1.0)

If you have the old v1.0 binary, compare outputs:

```bash
# Run v1.0
./docker-stats-on-exit-shim-v1.0 /tmp/v1_stats.json sleep 1

# Run v2.0
./docker-stats-on-exit-shim /tmp/v2_stats.json sleep 1

# Compare JSON structures
python3 << 'EOF'
import json

with open('/tmp/v1_stats.json') as f:
    v1 = json.load(f)

with open('/tmp/v2_stats.json') as f:
    v2 = json.load(f)

# Compare top-level keys
v1_keys = set(v1.keys())
v2_keys = set(v2.keys())

print("Top-level keys comparison:")
print(f"  v1.0: {sorted(v1_keys)}")
print(f"  v2.0: {sorted(v2_keys)}")
print(f"  Same: {v1_keys == v2_keys}")

# Compare cgroups keys
v1_cg = set(v1['cgroups'].keys())
v2_cg = set(v2['cgroups'].keys())

print("\nCgroups keys comparison:")
print(f"  v1.0: {sorted(v1_cg)}")
print(f"  v2.0: {sorted(v2_cg)}")
print(f"  Same: {v1_cg == v2_cg}")

print("\n✓ Structure is identical" if v1_keys == v2_keys and v1_cg == v2_cg else "\n✗ Structure differs")
EOF
```

## Where It's Used in Your Codebase

### HySDS Framework

**File**: `@/Users/mcayanan/git/hysds/hysds/job_worker.py`

**Lines 187-196**: `find_usage_stats()` function searches for `_docker_stats.json` files

**Lines 1554-1565**: Loads stats and appends to job metrics:
```python
for usage_stats_file in find_usage_stats(job_dir):
    usage_stats = {}
    with open(usage_stats_file) as f:
        try:
            usage_stats = json.load(f)
        except Exception as e:
            tb = traceback.format_exc()
            err = f"Failed to load usage stats from {usage_stats_file}: {str(e)}\n{tb}"
            logger.error(err)
    if len(usage_stats) > 0:
        job["job_info"]["metrics"]["usage_stats"].append(usage_stats)
```

**Test Data**: `@/Users/mcayanan/git/hysds/test/examples/_docker_stats.json`

This is the reference JSON structure that HySDS expects.

### Validation in HySDS

You can run the existing HySDS test to verify compatibility:

```bash
cd /Users/mcayanan/git/hysds

# Run the job_worker test that uses _docker_stats.json
python -m pytest test/test_job_worker.py::TestJobWorkerFuncs::test_find_usage_stats -v
```

## Troubleshooting

### If tests fail with "go: command not found"

Install Go 1.22+:
```bash
# On Mac
brew install go

# Verify
go version
```

### If cgroup detection seems wrong

Check your system's cgroup version:
```bash
# Check cgroup version
cat /proc/self/cgroup

# Check filesystem
ls -la /sys/fs/cgroup/

# cgroup v1: You'll see subdirectories like cpu, memory, etc.
# cgroup v2: You'll see files like cgroup.controllers, cgroup.procs
```

### If JSON structure differs

Run the compatibility test to see specific differences:
```bash
python3 test_hysds_compatibility.py
```

## Success Criteria

✅ All Go unit tests pass  
✅ Python compatibility test passes  
✅ Manual execution creates valid JSON  
✅ Exit codes are forwarded correctly  
✅ JSON structure matches HySDS expectations  
✅ HySDS job_worker.py can load the stats  

Once all tests pass, you're ready to proceed with the release!
