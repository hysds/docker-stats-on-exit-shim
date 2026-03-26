#!/usr/bin/env python3
"""
Test docker-stats-on-exit-shim v2.0 compatibility with HySDS.

This script validates that the v2.0 shim output is compatible with
the HySDS job_worker.py stats collection logic.
"""

import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path


def load_hysds_example_stats():
    """Load the example stats from HySDS test data."""
    hysds_example = Path(__file__).parent.parent / "hysds" / "test" / "examples" / "_docker_stats.json"
    
    if not hysds_example.exists():
        print(f"Warning: HySDS example not found at {hysds_example}")
        return None
    
    with open(hysds_example) as f:
        return json.load(f)


def build_shim():
    """Build the docker-stats-on-exit-shim binary."""
    print("Building docker-stats-on-exit-shim...")
    result = subprocess.run(
        ["go", "build", "-o", "docker-stats-on-exit-shim", "."],
        capture_output=True,
        text=True
    )
    
    if result.returncode != 0:
        print(f"Build failed:\n{result.stderr}")
        return False
    
    print("✓ Build successful")
    return True


def run_shim_test():
    """Run the shim and capture output."""
    with tempfile.TemporaryDirectory() as tmpdir:
        stats_file = os.path.join(tmpdir, "_docker_stats.json")
        
        print("\nRunning shim with sleep 1...")
        result = subprocess.run(
            ["./docker-stats-on-exit-shim", stats_file, "sleep", "1"],
            capture_output=True,
            text=True
        )
        
        if result.returncode != 0:
            print(f"Shim execution failed:\n{result.stderr}")
            return None
        
        # Print cgroup detection message
        if result.stderr:
            print(f"Shim output: {result.stderr.strip()}")
        
        # Load the generated stats
        with open(stats_file) as f:
            return json.load(f)


def validate_top_level_keys(stats, expected_keys):
    """Validate top-level keys match expected structure."""
    missing = set(expected_keys) - set(stats.keys())
    extra = set(stats.keys()) - set(expected_keys)
    
    if missing:
        print(f"✗ Missing top-level keys: {missing}")
        return False
    
    if extra:
        print(f"  Note: Extra top-level keys (acceptable): {extra}")
    
    print(f"✓ All required top-level keys present: {expected_keys}")
    return True


def validate_cgroups_structure(stats):
    """Validate cgroups sub-structure."""
    if "cgroups" not in stats:
        print("✗ Missing 'cgroups' key")
        return False
    
    cgroups = stats["cgroups"]
    expected_sections = ["cpu_stats", "memory_stats", "blkio_stats", "pids_stats"]
    
    missing = [s for s in expected_sections if s not in cgroups]
    if missing:
        print(f"✗ Missing cgroups sections: {missing}")
        return False
    
    print(f"✓ All cgroups sections present: {expected_sections}")
    
    # Validate cpu_stats structure
    if "cpu_usage" not in cgroups["cpu_stats"]:
        print("✗ Missing cpu_stats.cpu_usage")
        return False
    
    if "total_usage" not in cgroups["cpu_stats"]["cpu_usage"]:
        print("✗ Missing cpu_stats.cpu_usage.total_usage")
        return False
    
    print("✓ cpu_stats structure valid")
    
    # Check percpu_usage (may be empty on cgroup v2)
    percpu = cgroups["cpu_stats"]["cpu_usage"].get("percpu_usage", [])
    if isinstance(percpu, list):
        if len(percpu) == 0:
            print("  Note: percpu_usage is empty (acceptable on cgroup v2)")
        else:
            print(f"  Note: percpu_usage has {len(percpu)} entries")
    
    return True


def validate_data_types(stats):
    """Validate data types match expected types."""
    checks = [
        ("wall_time", int),
        ("user_cpu_time", int),
        ("sys_cpu_time", int),
    ]
    
    for key, expected_type in checks:
        if key not in stats:
            print(f"✗ Missing key: {key}")
            return False
        
        if not isinstance(stats[key], expected_type):
            print(f"✗ {key} has wrong type: {type(stats[key])} (expected {expected_type})")
            return False
    
    print("✓ All data types correct")
    return True


def validate_values(stats):
    """Validate values are reasonable."""
    # wall_time should be approximately 1 second (in nanoseconds)
    wall_time = stats["wall_time"]
    if wall_time < 500_000_000 or wall_time > 2_000_000_000:
        print(f"✗ wall_time out of range: {wall_time} ns (expected ~1,000,000,000 ns)")
        return False
    
    print(f"✓ wall_time is reasonable: {wall_time / 1_000_000_000:.2f} seconds")
    
    # CPU times should be non-negative
    if stats["user_cpu_time"] < 0 or stats["sys_cpu_time"] < 0:
        print("✗ CPU times are negative")
        return False
    
    print("✓ CPU times are non-negative")
    return True


def simulate_hysds_loading(stats):
    """Simulate how HySDS job_worker.py loads the stats."""
    print("\nSimulating HySDS job_worker.py loading...")
    
    try:
        # This is what job_worker.py does:
        # usage_stats = json.load(f)
        # job["job_info"]["metrics"]["usage_stats"].append(usage_stats)
        
        usage_stats_list = []
        usage_stats_list.append(stats)
        
        print(f"✓ Successfully loaded stats into usage_stats list")
        print(f"  Stats keys: {list(stats.keys())}")
        print(f"  Cgroups keys: {list(stats['cgroups'].keys())}")
        
        return True
    except Exception as e:
        print(f"✗ Failed to simulate HySDS loading: {e}")
        return False


def main():
    """Run all validation tests."""
    print("=" * 70)
    print("docker-stats-on-exit-shim v2.0 - HySDS Compatibility Test")
    print("=" * 70)
    
    # Build the shim
    if not build_shim():
        sys.exit(1)
    
    # Load HySDS example for reference
    hysds_example = load_hysds_example_stats()
    if hysds_example:
        print(f"\n✓ Loaded HySDS example stats for reference")
        expected_keys = list(hysds_example.keys())
    else:
        print("\nUsing default expected keys")
        expected_keys = ["wall_time", "user_cpu_time", "sys_cpu_time", "cgroups"]
    
    # Run the shim and get output
    stats = run_shim_test()
    if stats is None:
        sys.exit(1)
    
    print("\n" + "=" * 70)
    print("Validation Tests")
    print("=" * 70)
    
    # Run validation tests
    tests = [
        ("Top-level keys", lambda: validate_top_level_keys(stats, expected_keys)),
        ("Cgroups structure", lambda: validate_cgroups_structure(stats)),
        ("Data types", lambda: validate_data_types(stats)),
        ("Value ranges", lambda: validate_values(stats)),
        ("HySDS loading", lambda: simulate_hysds_loading(stats)),
    ]
    
    results = []
    for name, test_func in tests:
        print(f"\n{name}:")
        results.append(test_func())
    
    # Summary
    print("\n" + "=" * 70)
    print("Summary")
    print("=" * 70)
    
    passed = sum(results)
    total = len(results)
    
    print(f"Tests passed: {passed}/{total}")
    
    if passed == total:
        print("\n✓ ALL TESTS PASSED - v2.0 is compatible with HySDS!")
        sys.exit(0)
    else:
        print(f"\n✗ {total - passed} test(s) failed")
        sys.exit(1)


if __name__ == "__main__":
    main()
