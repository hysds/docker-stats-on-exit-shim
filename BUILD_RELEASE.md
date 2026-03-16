# Build and Release Guide for v2.0

## Prerequisites

- Go 1.22 or later
- Git
- Access to GitHub repository for releases

## Step 1: Clean Up Old Build System

Remove the git submodule and old vendor directory:

```bash
# Remove git submodule reference
git rm vendor/github.com/opencontainers/runc
rm -rf .gitmodules

# Commit the cleanup
git add .
git commit -m "Remove git submodule, migrate to Go modules"
```

## Step 2: Initialize Go Modules

The `go.mod` file has already been created. Download dependencies:

```bash
go mod download
go mod tidy
```

This will create `go.sum` with checksums for all dependencies.

## Step 3: Build and Test

### Run Unit Tests

```bash
go test -v ./...
```

Expected output:
- TestCgroupDetection: Validates cgroup version detection
- TestShimEndToEnd: Tests full workflow with sleep command
- TestShimExitCodeForwarding: Verifies exit codes 0, 1, 42
- TestJsonOutputCompatibility: Validates JSON structure

### Build for Linux AMD64

```bash
GOOS=linux GOARCH=amd64 go build -o docker-stats-on-exit-shim .
```

### Test the Binary

```bash
# Test on cgroup v1 host
./docker-stats-on-exit-shim /tmp/test_v1.json sleep 1
cat /tmp/test_v1.json

# Test on cgroup v2 host (if available)
./docker-stats-on-exit-shim /tmp/test_v2.json sleep 1
cat /tmp/test_v2.json
```

## Step 4: Integration Testing

### Test in Docker Container (cgroup v1)

```bash
docker run --rm -v $(pwd)/docker-stats-on-exit-shim:/shim:ro \
    -v /tmp/test-stats:/out ubuntu:20.04 \
    /shim /out/stats.json /bin/sleep 2

# Verify output
python3 -c "
import json
stats = json.load(open('/tmp/test-stats/stats.json'))
assert stats['wall_time'] > 1_500_000_000
assert 'cpu_stats' in stats['cgroups']
print('PASS: cgroup v1 stats collection works')
"
```

### Test in Rootless Container (cgroup v2)

If you have rootless Docker/Podman configured:

```bash
podman run --rm -v $(pwd)/docker-stats-on-exit-shim:/shim:ro \
    -v /tmp/test-stats:/out ubuntu:22.04 \
    /shim /out/stats_v2.json /bin/sleep 2

# Verify output
python3 -c "
import json
stats = json.load(open('/tmp/test-stats/stats_v2.json'))
assert stats['wall_time'] > 1_500_000_000
assert 'cpu_stats' in stats['cgroups']
print('PASS: cgroup v2 stats collection works')
"
```

## Step 5: Create GitHub Release

### Tag the Release

```bash
git tag -a v2.0.0 -m "v2.0.0 - Add cgroup v2 support

- Automatic cgroup v1/v2 detection at runtime
- Backward compatible with cgroup v1
- Support for rootless Docker/Podman
- Migrated to Go modules
- Added comprehensive unit tests
"

git push origin v2.0.0
```

### Create Release on GitHub

```bash
gh release create v2.0.0 docker-stats-on-exit-shim \
    --title "v2.0.0 - cgroup v2 Support" \
    --notes "## What's New

- **Automatic cgroup v1/v2 detection**: Runtime detection with appropriate API selection
- **Rootless container support**: Works correctly with rootless Docker/Podman (cgroup v2)
- **Backward compatible**: Identical JSON output structure on both cgroup versions
- **Go modules migration**: Modern dependency management with Go 1.22+
- **Comprehensive tests**: Unit tests for detection, end-to-end, exit codes, and JSON schema

## Breaking Changes

None - this is a drop-in replacement for v1.0.

## Installation

Download the binary and make it executable:

\`\`\`bash
wget https://github.com/hysds/docker-stats-on-exit-shim/releases/download/v2.0.0/docker-stats-on-exit-shim
chmod +x docker-stats-on-exit-shim
\`\`\`

## Verification

The shim will print to stderr which cgroup version it detected:
- \`docker-stats-on-exit-shim: using cgroup v1\`
- \`docker-stats-on-exit-shim: using cgroup v2 (path: /)\`

## Known Differences (Acceptable)

On cgroup v2 hosts:
- \`percpu_usage\` may be an empty array
- \`memory_stats.failcnt\` may be 0

These differences do not affect downstream consumers that extract \`wall_time\`, \`user_cpu_time\`, or \`sys_cpu_time\`.
"
```

## Step 6: Validation with HySDS

After release, validate with the HySDS framework:

1. Update test job to use v2.0 binary
2. Run on cgroup v1 host - verify identical output to v1.0
3. Run on cgroup v2 host - verify stats are collected
4. Check OpenSearch indexing works correctly
5. Verify downstream metrics extraction (nisar-pcm-pge-metrics)

## Step 7: Coordinate Downstream Updates

Create NISAR PGE Jira ticket for updating Dockerfiles:
- `.ci/docker/Dockerfile_l1_l2` (line 61)
- `.ci/docker/Dockerfile_l0b` (line 72)
- `.ci/docker/Dockerfile_dc_radar` (line 59)
- `.ci/docker/Dockerfile_l3_sm` (line 61)

Change example:
```dockerfile
# Old
RUN wget https://github.com/hysds/docker-stats-on-exit-shim/releases/download/v1.0/docker-stats-on-exit-shim

# New
RUN wget https://github.com/hysds/docker-stats-on-exit-shim/releases/download/v2.0.0/docker-stats-on-exit-shim
```

## Rollback Plan

If issues are discovered:

1. Revert to v1.0 binary in affected Dockerfiles
2. Investigate issue with v2.0
3. Create patch release v2.0.1 with fix
4. Re-test and re-deploy

## Acceptance Criteria Checklist

- [ ] AC-1: Shim detects cgroup version and prints to stderr
- [ ] AC-2: JSON output has identical structure on v1 and v2
- [ ] AC-3: go.mod and go.sum exist, vendor/ removed
- [ ] AC-4: v2.0 produces identical output to v1.0 on cgroup v1 hosts
- [ ] AC-5: `go test -v ./...` passes on both cgroup v1 and v2
- [ ] AC-6: NISAR PGE Jira ticket created for Dockerfile updates
- [ ] AC-7: GitHub release v2.0.0 published with binary
