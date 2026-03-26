# Docker Stats On Exit Shim

This is a small utility designed to capture the statistics for the run of a Docker
container before its destruction.

It is designed to be used as the main process of a Docker container that wraps the
real command by waiting for it to exit and then querying the active Cgroup subsystems
to gather their statistics. It dumps these statistics to a file as JSON and then exits
with the exit code of the real command.

## Features

- **Automatic cgroup v1/v2 detection**: Detects cgroup version at runtime and uses the appropriate API
- **Backward compatible**: Produces identical JSON output structure on both cgroup v1 and v2 hosts
- **Rootless container support**: Works correctly with rootless Docker/Podman (requires cgroup v2)
- **Drop-in replacement**: No changes needed to downstream consumers

## Example

```bash
$ docker run --rm -ti -v`pwd`:/tmp/:rw ubuntu /tmp/docker-stats-on-exit-shim /tmp/output.json /bin/sleep 1
$ cat output.json
```
```json
{
  "wall_time": 1000765975,
  "user_cpu_time": 0,
  "sys_cpu_time": 0,
  "cgroups": {
    "cpu_stats": {
      "cpu_usage": {
        "total_usage": 21326399,
        "percpu_usage": [
          14721062,
          1512284,
          1730836,
          3362217,
          0,
          0,
          0,
          0
        ],
        "usage_in_kernelmode": 0,
        "usage_in_usermode": 10000000
      },
      "throttling_data": {}
    },
    "memory_stats": {
    ...
    }
  }
}
```

## Building

This project uses Go modules (requires Go 1.22+).

### Local Build

```bash
git clone https://github.com/hysds/docker-stats-on-exit-shim.git
cd docker-stats-on-exit-shim
go mod download
go build -o docker-stats-on-exit-shim .
```

### Cross-Compile for Linux

```bash
GOOS=linux GOARCH=amd64 go build -o docker-stats-on-exit-shim .
```

### Run Tests

```bash
go test -v ./...
```

### Build in Docker Container

```bash
docker run --rm -v $(pwd):/workspace -w /workspace golang:1.22 go build -o docker-stats-on-exit-shim .
```

## Caveats

* The recorded statistics won't quite be before container destruction but it's probably close enough.
* The recorded statistics will contain the run of the tool (i.e. it will contribute to CPU usage). It should
  be a very small contribution though.
