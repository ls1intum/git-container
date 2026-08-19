# Hades Git Container

The Hades Git Container is a specialized component of the Hades CI system, designed to handle git repository operations in a secure and isolated environment. This container is responsible for cloning repositories and managing git operations required for CI/CD workflows.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- Secure git repository cloning
- Isolated environment for git operations
- Support for various authentication methods

## Configuration

The container can be configured using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `GIT_CLONE_PATH` | Base path for cloning repositories | `/repos` |
| `CACHE_PATH` | Path for git cache storage | `/cache` |

### Per-repository variables

Each repository to clone is described by a group of `HADES_<group>_<type>` environment variables (for example `HADES_ASSIGNMENT_URL`, `HADES_TEST_BRANCH`). The supported `<type>` values are:

| Type | Description |
|------|-------------|
| `URL` | Repository URL to clone (mandatory) |
| `USERNAME` | Username for HTTP basic authentication (optional) |
| `PASSWORD` | Password / token for HTTP basic authentication (optional) |
| `BRANCH` | Branch to clone (optional, defaults to the remote HEAD) |
| `COMMIT` | Full 40-character SHA-1 commit hash to check out after cloning (optional) |
| `PATH` | Sub-directory to clone into, relative to the base path (optional) |
| `ORDER` | Clone order for this repository (optional) |

When `COMMIT` is set, the container checks out that exact commit after cloning, instead of testing the branch HEAD. This prevents a race condition where a newer commit pushed after a job was queued would otherwise be tested while the result is labelled with the originally scheduled commit. When `COMMIT` is empty, the branch HEAD is used as before.

## Usage

### Running the Container

```fish
docker compose up -d
```

## Development

### Prerequisites

- Go 1.24 or later
- Docker

### Building

```fish
# Build the container
docker build -t hades-git-container .

# Run tests
go test ./...
```

