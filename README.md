# InfVerif (Go)

[中文版](README_CN.md)

A Go reimplementation of Microsoft's `InfVerif.exe` (Driver Package INF Verifier), compatible with v10.0.26200+.

## Overview

InfVerif validates Windows driver package `.inf` files against Microsoft's specification requirements.

## Features

- **Multiple validation modes** matching the original binary's bit-flag architecture:
  - `/h` — WHQL signature requirements check (flag `0x80`)
  - `/c` — Configurability check (flag `0x01`)
  - `/u` — Universal Driver check (flag `0x03`)
  - `/w` — Windows Driver check (flag `0x07`)
  - `/k` — Declarative Driver requirements check (flag `0x43`)
- **INF information display** (`/info`) — file hash, family ID, driver type, devices
- **Dependency analysis** (`/depends`) — Include/Needs dependency tree
- **Syntax analysis** (`/syntax`) — INF directive syntax with minimum OS version
- **Error code help** (`/code`) — lookup error descriptions by code
- **Rule version control** (`/rulever`) — specify rule version for `/h` mode (named versions: `vnext`, `24h2`, `25h2`, etc.)
- **Provider validation** (`/provider`) — enforce provider name matching
- **Exception management**: `/noexceptions`, `/showexceptions`, `/hdcrules`
- **Output formats**: console (default), MSBuild (`/msbuild`), CSV (`/csv`)
- **Error management**: `/werror`, `/errorlist`, `/errorlevel`
- **File handling**: wildcards, `/recurse`, `/exclude`
- Verbose mode (`/v`) with output compatible with the original binary

## Build

```bash
go build -o infverif.exe ./cmd/infverif/
```

## Usage

```
infverif [/code <error code>] [/v] [[/h] | [/w] | [/u] | [/k]]
         [/rulever <Major.Minor.Build> | vnext]
         [/wbuild <Major.Minor.Build>] [/info] [/stampinf]
         [/l <path>] [/osver <TargetOSVersion>] [/product <ias file>]
         [/provider <ProviderName>] [/noexceptions] [/syntax]
         [/csv <file>] [/errorlist <file>] [/errorlevel <n>]
         [/werror] [/exclude <file>] [/levelsort] [/msbuild]
         [/inbox] [/append] [/fileroot <path>] [/recurse] <files>
```

### Examples

```bash
# Basic validation
infverif driver.inf

# WHQL signature requirements check
infverif /h /v driver.inf

# Signature check with specific rule version
infverif /h /rulever 25h2 driver.inf

# Signature check without exceptions
infverif /h /noexceptions driver.inf

# Universal Driver check with verbose output
infverif /v /u driver.inf

# Windows Driver check
infverif /v /w driver.inf

# Declarative Driver requirements check
infverif /k driver.inf

# Display INF summary information
infverif /info driver.inf

# Show Include/Needs dependencies
infverif /depends driver.inf

# INF syntax analysis
infverif /syntax driver.inf

# Lookup error code help
infverif /code 1203

# Enforce provider name
infverif /provider "MyCompany" driver.inf

# Show HDC rules and exceptions
infverif /hdcrules
infverif /showexceptions

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

**Modes** (mutually exclusive):

| Flag       | Description                                         |
| ---------- | --------------------------------------------------- |
| `/h`       | WHQL signature requirements check (flag `0x80`)     |
| `/c`       | Configurability check (flag `0x01`)                 |
| `/u`       | Universal Driver check (flag `0x03`)                |
| `/w`       | Windows Driver check (flag `0x07`)                  |
| `/k`       | Declarative Driver requirements check (flag `0x43`) |
| `/info`    | Display INF summary (no validation)                 |
| `/depends` | Display Include/Needs dependencies                  |
| `/syntax`  | INF syntax report with minimum OS versions          |

**Options**:

| Flag              | Argument  | Description                                        |
| ----------------- | --------- | -------------------------------------------------- |
| `/v`              | —         | Verbose output                                     |
| `/code`           | `<code>`  | Display help for a specific error code             |
| `/rulever`        | `<M.m.B>` | Rule version for `/h` mode (`vnext`, `24h2`, etc.) |
| `/provider`       | `<name>`  | Enforce provider name matching                     |
| `/noexceptions`   | —         | Disable exceptions in `/h` mode                    |
| `/attestation`    | —         | Attestation signing mode                           |
| `/hdcrules`       | —         | Display HDC error code rules                       |
| `/showexceptions` | —         | Display all exception entries                      |
| `/stampinf`       | —         | Treat `$ARCH$` as valid architecture               |
| `/l`              | `<path>`  | HTML log output directory                          |
| `/osver`          | `<ver>`   | Target OS version filter                           |
| `/wbuild`         | `<M.m.B>` | Minimum build for `/w` enforcement                 |
| `/product`        | `<file>`  | Product definition `.ias` file                     |
| `/dll`            | `<path>`  | External validation DLL                            |
| `/csv`            | `<file>`  | CSV output file                                    |
| `/msbuild`        | —         | MSBuild-compatible error format                    |
| `/errorlist`      | `<file>`  | Error suppression list (CSV)                       |
| `/errorlevel`     | `<n>`     | Error level threshold (1=ERROR, 2=WARNING, 3=INFO) |
| `/werror`         | —         | Treat warnings as errors                           |
| `/exclude`        | `<file>`  | File exclusion list                                |
| `/levelsort`      | —         | Sort output by error level                         |
| `/inbox`          | —         | Inbox driver validation mode                       |
| `/append`         | —         | Append to CSV (don't overwrite)                    |
| `/fileroot`       | `<path>`  | File root for path resolution                      |
| `/recurse`        | —         | Recursive directory search                         |
| `/logging`        | —         | Enable logging output                              |
| `/verboseparams`  | —         | Display InfVerif parameter flags                   |
| `/samples`        | —         | Samples mode                                       |
| `/wdk`            | —         | WDK mode                                           |

## Project Structure

```
infverif/
├── cmd/infverif/       # CLI entry point
│   └── main.go
├── pkg/
│   ├── infparser/      # INF file parser (UTF-16LE/BE/UTF-8)
│   │   └── parser.go
│   └── verifier/       # Validation engine
│       ├── verifier.go   # Core validation logic
│       ├── errordb.go    # Error code database & HDC rules
│       └── exceptions.go # Exception tables & rule versions
└── go.mod
```

## License

MIT License
