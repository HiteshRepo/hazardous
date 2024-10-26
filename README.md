# Hazardous

**Hazardous** is a Golang linter designed to scan files for potentially hazardous commands.

Future updates aim to add support for more commands that are considered unsafe.

### Detecting Unsafe `rm -rf` Usage

**Hazardous** issues warnings when it encounters potentially unsafe `rm -rf` commands, like:

```
2024/10/24 19:12:43 unsafe code found at position 12,3 in linters/hazardous/examples/unsafe-1.sh
```

### Detecting Unassigned Variables

It also flags unassigned variables as follows:

```
2024/10/24 19:20:20 un-assigned variable 'DOCKER_IMAGE_BASE' found at position 7,6 in linters/hazardous/examples/unsafe-1/Makefile 2024/10/24 19:20:20 un-assigned variable 'GIT_COMMIT' found at position 9,3 in linters/hazardous/examples/unsafe-2/Makefile
```

## Installation

Install **Hazardous** as a Go module with:

```bash
go install github.com/hiteshrepo/hazardous@latest
```

## Usage

Hazardous operates similarly to go vet. After installation, run it with:
```bash
hazardous --allow-extensions=.sh,Makefile --exclude-dirs=node_modules,linters <directory>/<file>
```

You can also apply it across multiple directories, for example, to scan your whole project:
```bash
hazardous --allow-extensions=.sh,Makefile --exclude-dirs=node_modules,linters ./...
```

### Flags
- `--allow-extensions`: Scans only files with specified extensions, separated by commas (e.g., `.sh,Makefile`).
- `--exclude-dirs`: Excludes directories from scanning, also comma-separated (e.g., `node_modules,linters`).

## Limitations

Currently, Hazardous scans only `.sh` and `Makefile` files, detecting:
- Unsafe rm -rf commands
- Unassigned variables

## Improvements

To suggest additional unsafe command detections, please open an issue.
