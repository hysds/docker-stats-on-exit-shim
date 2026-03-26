#!/bin/bash
# Quick test script for docker-stats-on-exit-shim v2.0
# Run this to verify the implementation works correctly

set -e  # Exit on error

echo "========================================================================"
echo "docker-stats-on-exit-shim v2.0 - Quick Test"
echo "========================================================================"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.22+ first."
    echo "   On Mac: brew install go"
    exit 1
fi

echo "✓ Go is installed: $(go version)"

# Navigate to the correct directory
cd "$(dirname "$0")"

echo ""
echo "Step 1: Download dependencies"
echo "------------------------------------------------------------------------"
go mod download
go mod tidy
echo "✓ Dependencies downloaded"

echo ""
echo "Step 2: Build the shim"
echo "------------------------------------------------------------------------"
go build -o docker-stats-on-exit-shim .
echo "✓ Build successful"

echo ""
echo "Step 3: Run Go unit tests"
echo "------------------------------------------------------------------------"
go test -v ./...
echo "✓ All Go tests passed"

echo ""
echo "Step 4: Manual functional test"
echo "------------------------------------------------------------------------"
STATS_FILE="/tmp/docker_stats_test_$$.json"
./docker-stats-on-exit-shim "$STATS_FILE" sleep 1

if [ ! -f "$STATS_FILE" ]; then
    echo "❌ Stats file was not created"
    exit 1
fi

echo "✓ Stats file created: $STATS_FILE"

# Validate JSON
if command -v python3 &> /dev/null; then
    echo ""
    echo "Step 5: Validate JSON structure"
    echo "------------------------------------------------------------------------"
    python3 << EOF
import json
import sys

with open('$STATS_FILE') as f:
    stats = json.load(f)

required_keys = ['wall_time', 'user_cpu_time', 'sys_cpu_time', 'cgroups']
missing = [k for k in required_keys if k not in stats]

if missing:
    print(f"❌ Missing keys: {missing}")
    sys.exit(1)

print("✓ All required top-level keys present")

if 'cgroups' in stats:
    cgroups_keys = list(stats['cgroups'].keys())
    print(f"✓ Cgroups sections: {cgroups_keys}")

wall_time_sec = stats['wall_time'] / 1_000_000_000
print(f"✓ Wall time: {wall_time_sec:.2f} seconds")

print("\n✓ JSON structure is valid")
EOF
else
    echo "⚠ Python3 not found, skipping JSON validation"
fi

echo ""
echo "Step 6: Test exit code forwarding"
echo "------------------------------------------------------------------------"
./docker-stats-on-exit-shim /tmp/exit0_$$.json sh -c "exit 0"
if [ $? -eq 0 ]; then
    echo "✓ Exit code 0 forwarded correctly"
else
    echo "❌ Exit code 0 not forwarded correctly"
    exit 1
fi

./docker-stats-on-exit-shim /tmp/exit1_$$.json sh -c "exit 1" || true
if [ $? -eq 1 ]; then
    echo "✓ Exit code 1 forwarded correctly"
else
    echo "❌ Exit code 1 not forwarded correctly"
    exit 1
fi

echo ""
echo "========================================================================"
echo "Summary"
echo "========================================================================"
echo "✓ Build successful"
echo "✓ Unit tests passed"
echo "✓ Functional test passed"
echo "✓ JSON structure valid"
echo "✓ Exit codes forwarded correctly"
echo ""
echo "🎉 All tests passed! v2.0 is working correctly."
echo ""
echo "Next steps:"
echo "  1. Run HySDS compatibility test: python3 test_hysds_compatibility.py"
echo "  2. Test in Docker container (see TESTING_GUIDE.md)"
echo "  3. Follow BUILD_RELEASE.md to create v2.0.0 release"
echo ""
