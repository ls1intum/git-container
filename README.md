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

