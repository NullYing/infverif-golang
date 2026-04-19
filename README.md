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
- **Error code help** (`/code`) — lookup error descriptions by code
- **Rule version control** (`/rulever`) — specify rule version for `/h` mode (named versions: `vnext`, `24h2`, `25h2`, etc.)
- **Provider validation** (`/provider`) — enforce provider name matching
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
         [/provider <ProviderName>] <files>
```

### Examples

```bash
# Basic validation
infverif driver.inf

# WHQL signature requirements check
infverif /h /v driver.inf

# Signature check with specific rule version
infverif /h /rulever 25h2 driver.inf

# Universal Driver check with verbose output
infverif /v /u driver.inf

# Windows Driver check
infverif /v /w driver.inf

# Declarative Driver requirements check
infverif /k driver.inf

# Display INF summary information
infverif /info driver.inf

# Lookup error code help
infverif /code 1203

# Enforce provider name
infverif /provider "MyCompany" driver.inf

# Validate pre-stampinf INF files
infverif /stampinf driver.inf

# Specify minimum Windows build for /w mode
infverif /w /wbuild 10.0.19041 driver.inf
```

### CLI Flags

**Modes** (mutually exclusive):

| Flag    | Description                                         |
| ------- | --------------------------------------------------- |
| `/h`    | WHQL signature requirements check (flag `0x80`)     |
| `/c`    | Configurability check (flag `0x01`)                 |
| `/u`    | Universal Driver check (flag `0x03`)                |
| `/w`    | Windows Driver check (flag `0x07`)                  |
| `/k`    | Declarative Driver requirements check (flag `0x43`) |
| `/info` | Display INF summary (no validation)                 |

**Options**:

| Flag        | Argument  | Description                                                |
| ----------- | --------- | ---------------------------------------------------------- |
| `/v`        | —         | Verbose output                                             |
| `/code`     | `<code>`  | Display help for a specific error code                     |
| `/rulever`  | `<M.m.B>` | Rule version for `/h` mode (`vnext`, `24h2`, `25h2`, etc.) |
| `/provider` | `<name>`  | Enforce provider name matching                             |
| `/stampinf` | —         | Treat `$ARCH$` as valid architecture                       |
| `/l`        | `<path>`  | HTML log output directory                                  |
| `/osver`    | `<ver>`   | Target OS version filter                                   |
| `/wbuild`   | `<M.m.B>` | Minimum build for `/w` enforcement (default `10.0.17763`)  |
| `/product`  | `<file>`  | Product definition `.ias` file                             |

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
