# Hades Git Container

The Hades Git Container is a specialized component of the Hades CI system, designed to handle git repository operations in a secure and isolated environment. This container is responsible for cloning repositories and managing git operations required for CI/CD workflows.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- Secure git repository cloning
- Isolated environment for git operations
- Support for various authentication methods

## Configuration

The container is configured entirely through environment variables. There are two groups: a small set of global variables that control the container itself, and any number of per-repository variable groups that describe the repositories to clone.

### Global variables

| Variable | Description | Default |
|----------|-------------|---------|
| `REPOSITORY_DIR` | Base directory the repositories are cloned into. It is only applied when the path already exists inside the container; otherwise the default is used. | `/opt/repositories/` |
| `DEBUG` | When set to the string `true`, raises the log level to `debug` so every cloned repository is logged in detail. | unset (info level) |

### Per-repository variables

Every repository to clone is described by a *group* of environment variables that share the naming scheme:

```
HADES_<group>_<type>
```

`<group>` is an arbitrary label you choose to tie the variables of one repository together (for example `ASSIGNMENT`, `TEST`, `group1`). `<type>` selects which field of that repository the variable sets. All variables that share the same `<group>` describe a single repository.

The supported `<type>` values are:

| Type | Description |
|------|-------------|
| `URL` | Repository URL to clone (mandatory) |
| `USERNAME` | Username for HTTP basic authentication (optional) |
| `PASSWORD` | Password / token for HTTP basic authentication (optional) |
| `BRANCH` | Branch to clone (optional, defaults to the remote HEAD) |
| `COMMIT` | Full 40-character SHA-1 commit hash to check out after cloning (optional) |
| `PATH` | Sub-directory under the base directory to clone into (optional) |
| `ORDER` | Integer that controls the clone order relative to the other repositories (optional, defaults to `0`) |

When `COMMIT` is set, the container checks out that exact commit after cloning, instead of testing the branch HEAD. This prevents a race condition where a newer commit pushed after a job was queued would otherwise be tested while the result is labelled with the originally scheduled commit. When `COMMIT` is empty, the branch HEAD is used as before.

#### How the parsing works internally

Understanding how the container reads these variables explains the naming rules:

1. On start-up the container scans **all** environment variables and keeps only those whose name begins with the exact prefix `HADES_`.
2. Each matching name is split into at most three parts on the underscore: `HADES`, `<group>`, and `<type>`. The split happens only on the **first two** underscores, so `<type>` is everything after the second underscore.
3. Variables that share the same `<group>` are collected into one repository, and `<type>` decides which field it fills.

This has a few practical consequences:

- **`<group>` must not contain an underscore.** Because the split stops after the second underscore, the first underscore-delimited token becomes the group and the rest becomes the type. `HADES_MY_REPO_URL` is parsed as group `MY` with type `REPO_URL` (an unknown type that is ignored), not group `MY_REPO`. Use a single token such as `MYREPO` instead.
- **`<type>` is case-sensitive and must be upper-case.** The fields are matched against the exact keys `URL`, `USERNAME`, `PASSWORD`, `BRANCH`, `COMMIT`, `PATH`, and `ORDER`. A variable like `HADES_group1_url` is silently ignored.
- **`<group>` is case-sensitive** but otherwise free-form. `HADES_Group1_URL` and `HADES_group1_URL` describe two different repositories.
- A group **without a `URL`** is skipped with a warning; every other field is optional.
- Any `HADES_...` name that has fewer than two underscores after the prefix is ignored.

### Cloning multiple repositories

Cloning several repositories in one container run is simply a matter of defining several groups. Give each repository its own `<group>` label and the container clones all of them.

By default all repositories share clone order `0` and the order in which they are cloned is unspecified. Set `HADES_<group>_ORDER` to an integer to make the order deterministic: repositories are cloned from the **lowest** `ORDER` value to the highest. This matters when one repository has to exist before another, for example an assignment repository that a test repository is placed inside.

Give each repository a distinct `PATH` (or rely on the default, described below) so they do not clone into the same directory.

#### Where each repository ends up

The target directory for a repository is derived as follows:

- If `PATH` is set, the repository is cloned into `<REPOSITORY_DIR>/<PATH>`.
- If `PATH` is not set, the container derives the directory name from the last segment of the `URL` with a trailing `.git` removed. For example `https://github.com/group1/repo1.git` clones into `<REPOSITORY_DIR>/repo1`.

#### Example: two repositories

The following clones a test repository first (order `1`) into `example`, then an assignment repository (order `2`) nested inside it at `example/assignment`. This mirrors the setup used by the Hades scheduler, where the assignment is checked out inside the test project so the tests can run against it:

```yaml
services:
  git-container:
    image: ghcr.io/hades-scheduler/git-container:latest
    environment:
      - REPOSITORY_DIR=/opt/repositories

      # First repository: the tests, cloned into /opt/repositories/example
      - HADES_TEST_URL=https://github.com/Mtze/Artemis-Java-Test.git
      - HADES_TEST_PATH=./example
      - HADES_TEST_ORDER=1

      # Second repository: the assignment, cloned into
      # /opt/repositories/example/assignment
      - HADES_ASSIGNMENT_URL=https://github.com/Mtze/Artemis-Java-Solution.git
      - HADES_ASSIGNMENT_PATH=./example/assignment
      - HADES_ASSIGNMENT_ORDER=2
    volumes:
      - ./repo:/opt/repositories
```

For a private repository, add `HADES_<group>_USERNAME` and `HADES_<group>_PASSWORD`, and pin an exact revision with `HADES_<group>_COMMIT` when needed. A runnable version of this configuration is available in [`compose.yml`](compose.yml), which also contains a second service (`git-container-pinned`) that clones the same two repositories pinned to explicit commit hashes.

## Usage

### Running the Container

```fish
docker compose up -d
```

## Development

### Prerequisites

- Go 1.25 or later
- Docker

### Building

```fish
# Build the container
docker build -t hades-git-container .

# Run tests
go test ./...
```

