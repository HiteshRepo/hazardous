# Hazardous

**Hazardous** is a Go CLI linter that scans `.sh` and `Makefile` files for potentially dangerous shell commands and warns before they cause irreversible damage. It works similarly to `go vet`.

## Detected Commands

| Command | Condition | Example |
|---|---|---|
| `rm` | `-r`, `-f`, `-rf`, `-fr`, `--recursive`, `--force` | `rm -rf /tmp/build` |
| `chmod` | `-R`/`--recursive`, or world-writable mode (`777`, `666`, `a+w`, `o+w`, `+w`, `a+rwx`) | `chmod 777 /etc/passwd` |
| `dd` | `of=/dev/…` (writing to a block device) | `dd if=/dev/zero of=/dev/sda` |
| `mkfs` / `mkfs.*` | any invocation | `mkfs.ext4 /dev/sdb1` |
| `fdisk` | with a `/dev/…` argument | `fdisk /dev/sda` |
| `curl` / `wget` pipe | piped into `bash`, `sh`, `zsh`, `dash`, or `fish` | `curl https://… \| bash` |

## Installation

```bash
go install github.com/hiteshrepo/hazardous@latest
```

## Usage

Scan a single file:
```bash
hazardous script.sh
hazardous Makefile
```

Scan an entire project (walks all `.sh` and `Makefile` files recursively):
```bash
hazardous ./...
```

With custom options:
```bash
hazardous --allow-extensions=.sh,Makefile --exclude-dirs=node_modules,vendor ./...
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--allow-extensions` | `.sh,Makefile` | Comma-separated file extensions to scan |
| `--exclude-dirs` | `node_modules,linters` | Comma-separated directory names to skip |

### Exit codes

| Code | Meaning |
|---|---|
| `0` | No issues found |
| `1` | One or more hazardous commands detected |

This makes hazardous suitable as a CI gate — a non-zero exit code will fail the pipeline.

### Output format

Each issue is printed to stderr:

```
2024/10/24 19:12:43 unsafe code found at position <line>,<col> in <filepath>
```

## How It Works

Hazardous uses two distinct scanning strategies depending on the file type.

### Shell scripts (`.sh`)

Shell scripts are parsed into a full AST using [`mvdan.cc/sh`](https://github.com/mvdan/sh). The AST is walked node-by-node:

- **`CallExpr` nodes** — individual commands — are checked against every known hazardous command pattern.
- **`BinaryCmd` nodes with a `|` operator** — pipe expressions — are checked for download-tool-to-shell patterns (`curl`/`wget` piped into a shell).

This means detection works correctly inside functions, conditionals, loops, and nested scopes — not just top-level commands.

### Makefiles

Makefiles are scanned line by line. Each line is tokenised by whitespace and checked against all hazardous patterns sequentially. Makefile execution prefixes (`@`, `-`, `+`) are stripped from command tokens before matching.

### Adding a new hazardous command

1. Add a new check function in `pkg/hazardous/hazardous.go`:
   - For shell: add a `checkXxx(cmd *syntax.CallExpr, filepath string) *issue.Issue` function and a new `case` in `CheckHazardousCommand`.
   - For Makefiles: add an `xxxMakefileLine(line string) (string, uint)` function and call it from `CheckMakefileLine`.
2. Add unit tests in `pkg/hazardous/hazardous_test.go`.
3. Add integration test fixtures under `testdata/scripts/` as `.txtar` files.

## Development

```bash
# Build
go build -v ./...

# Run all tests
go test ./...

# Run tests with verbose output (matches CI)
gotestsum --format testname -- --timeout 5m ./...

# Run benchmarks
go test -bench=. ./pkg/hazardous/...

# Lint
make lint

# Format
make fmt

# Install and run locally
make run-hazardous
```

## Contributing

To suggest an additional unsafe command detection, please [open an issue](https://github.com/hiteshrepo/hazardous/issues).
