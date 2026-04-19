# InfVerif (Go)

[中文版](README_CN.md)

A Go reimplementation of Microsoft's `InfVerif.exe` (Driver Package INF Verifier), (v10.0.22621.382).

## Overview

InfVerif validates Windows driver package `.inf` files against Microsoft's specification requirements.

## Features

- **Multiple validation modes** matching the original binary's bit-flag architecture:
  - `/c` — Configurability check (flag `0x01`)
  - `/u` — Universal Driver check (flag `0x03`)
  - `/w` — Windows Driver check (flag `0x07`)
  - `/k` — Windows Update submission check (flag `0x43`)
- **INF information display** (`/info`) — file hash, family ID, driver type, devices
- **Dependency analysis** (`/depends`) — Include/Needs dependency tree
- **Output formats**: console (default), MSBuild (`/msbuild`), CSV (`/csv`)
- **Error management**: `/werror`, `/errorlist`, `/errorlevel`
- **File handling**: wildcards, `/recurse`, `/exclude`
- Verbose mode (`/v`) with output identical to the original binary

## Build

```bash
go build -o infverif.exe ./cmd/infverif/
```

## Usage

```
infverif [/v] [[/c] | [/u] | [/w] | [/k]] [/wbuild <Major.Minor.Build>]
         [/info] [/depends] [/stampinf] [/l <path>]
         [/osver <TargetOSVersion>] [/product <ias file>]
         [/csv <file>] [/errorlist <file>] [/errorlevel <n>]
         [/werror] [/exclude <file>] [/levelsort] [/msbuild]
         [/inbox] [/append] [/fileroot <path>] [/recurse] <files>
```

### Examples

```bash
# Basic validation
infverif driver.inf

# Universal Driver check with verbose output
infverif /v /u driver.inf

# Windows Driver check
infverif /v /w driver.inf

# Windows Update submission check
infverif /k driver.inf

# Display INF summary information
infverif /info driver.inf

# Show Include/Needs dependencies
infverif /depends driver.inf

# Output to CSV
infverif /csv results.csv /recurse *.inf

# Treat warnings as errors
infverif /werror /u driver.inf

# MSBuild-compatible output
infverif /msbuild /w driver.inf

# Suppress specific errors (1310-1319 cannot be suppressed)
infverif /errorlist allowed.csv driver.inf
```

### CLI Flags

| Flag          | Argument  | Description                                        |
| ------------- | --------- | -------------------------------------------------- |
| `/v`          | —         | Verbose output                                     |
| `/c`          | —         | Configurability check mode                         |
| `/u`          | —         | Universal Driver check mode                        |
| `/w`          | —         | Windows Driver check mode                          |
| `/k`          | —         | Windows Update submission check mode               |
| `/info`       | —         | Display INF summary (no validation)                |
| `/depends`    | —         | Display Include/Needs dependencies                 |
| `/stampinf`   | —         | Treat `$ARCH$` as valid architecture               |
| `/l`          | `<path>`  | HTML log output directory                          |
| `/osver`      | `<ver>`   | Target OS version filter                           |
| `/wbuild`     | `<M.m.B>` | Minimum build for `/w` enforcement                 |
| `/product`    | `<file>`  | Product definition `.ias` file                     |
| `/csv`        | `<file>`  | CSV output file                                    |
| `/msbuild`    | —         | MSBuild-compatible error format                    |
| `/errorlist`  | `<file>`  | Error suppression list (CSV)                       |
| `/errorlevel` | `<n>`     | Error level threshold (1=ERROR, 2=WARNING, 3=INFO) |
| `/werror`     | —         | Treat warnings as errors                           |
| `/exclude`    | `<file>`  | File exclusion list                                |
| `/levelsort`  | —         | Sort output by error level                         |
| `/inbox`      | —         | Inbox driver validation mode                       |
| `/append`     | —         | Append to CSV (don't overwrite)                    |
| `/fileroot`   | `<path>`  | File root for path resolution                      |
| `/recurse`    | —         | Recursive directory search                         |

## Project Structure

```
infverif/
├── cmd/infverif/       # CLI entry point
│   └── main.go
├── pkg/
│   ├── infparser/      # INF file parser (UTF-16LE/BE/UTF-8)
│   │   └── parser.go
│   └── verifier/       # Validation engine
│       └── verifier.go
└── go.mod
```

## License

MIT License
