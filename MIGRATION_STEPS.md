# Migration Steps from v1.0 to v2.0

## Quick Start Commands

Run these commands in order to complete the migration:

```bash
cd /Users/mcayanan/git/docker-stats-on-exit-shim

# 1. Remove git submodule
git rm -f vendor/github.com/opencontainers/runc
rm -f .gitmodules

# 2. Initialize Go modules (already done - go.mod exists)
go mod download
go mod tidy

# 3. Verify the build works
go build -o docker-stats-on-exit-shim .

# 4. Run tests
go test -v ./...

# 5. Stage all changes
git add .

# 6. Commit the changes
git commit -m "v2.0: Add cgroup v2 support

- Add automatic cgroup v1/v2 detection at runtime
- Migrate from GOPATH + git submodules to Go modules
- Add comprehensive unit tests (main_test.go)
- Update README with modern build instructions
- Maintain backward compatibility with cgroup v1

Fixes FIXME at main.go:105 - now supports both cgroup versions
"

# 7. Create and push tag
git tag -a v2.0.0 -m "v2.0.0 - cgroup v2 support"
git push origin main
git push origin v2.0.0
```

## What Changed

### Files Modified
- `main.go`: Added cgroup v2 detection and dual-mode manager creation
- `README.md`: Updated with Go modules build instructions and features section

### Files Created
- `go.mod`: Go modules configuration
- `main_test.go`: Comprehensive unit tests
- `BUILD_RELEASE.md`: Build and release documentation
- `MIGRATION_STEPS.md`: This file

### Files to Remove
- `.gitmodules`: Git submodule configuration (no longer needed)
- `vendor/github.com/opencontainers/runc/`: Git submodule directory (replaced by Go modules)

### Files Generated (after go mod tidy)
- `go.sum`: Dependency checksums

## Verification Checklist

After running the commands above, verify:

- [ ] `vendor/` directory is removed
- [ ] `.gitmodules` file is removed
- [ ] `go.sum` file exists
- [ ] `go build` completes successfully
- [ ] `go test -v ./...` passes all tests
- [ ] Binary runs: `./docker-stats-on-exit-shim /tmp/test.json sleep 1`
- [ ] Stats file created: `cat /tmp/test.json`
- [ ] Stderr shows cgroup version: "using cgroup v1" or "using cgroup v2"

## Next Steps

1. **Test on cgroup v1 host**: Verify backward compatibility
2. **Test on cgroup v2 host**: Verify new functionality
3. **Create GitHub release**: Follow BUILD_RELEASE.md
4. **Update downstream**: Create NISAR PGE ticket for Dockerfile updates
5. **Validate HySDS**: Test with actual HySDS jobs

## Troubleshooting

### If go mod tidy fails
```bash
# Clear module cache and retry
go clean -modcache
go mod download
go mod tidy
```

### If tests fail
```bash
# Run tests with verbose output
go test -v -run TestCgroupDetection ./...

# Check cgroup version on your system
cat /proc/self/cgroup
ls -la /sys/fs/cgroup/
```

### If build fails
```bash
# Check Go version (need 1.22+)
go version

# Verify dependencies
go mod verify
```

## Rollback

If you need to rollback to the old GOPATH + submodule approach:

```bash
git checkout v1.0
git submodule init
git submodule update
```
