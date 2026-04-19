# InfVerif (Go)

[English](README.md)

使用 Go 语言重新实现微软 `InfVerif.exe`（Driver Package INF Verifier），兼容 v10.0.26200+ 版本。

## 概述

InfVerif 用于验证 Windows 驱动程序包 `.inf` 文件是否符合微软的规范要求。

## 功能特性

- **多种验证模式**，与原始二进制的位标志架构一致：
  - `/h` — WHQL 签名要求检查（标志 `0x80`）
  - `/c` — 配置能力检查（标志 `0x01`）
  - `/u` — 通用驱动检查（标志 `0x03`）
  - `/w` — Windows 驱动检查（标志 `0x07`）
  - `/k` — 声明性驱动要求检查（标志 `0x43`）
- **INF 信息显示**（`/info`）— 文件哈希、Family ID、驱动类型、设备信息
- **错误码查询**（`/code`）— 按错误码查询描述信息
- **规则版本控制**（`/rulever`）— 指定 `/h` 模式的规则版本（命名版本：`vnext`、`24h2`、`25h2` 等）
- **提供商验证**（`/provider`）— 强制匹配提供商名称
- 详细模式（`/v`）输出与原始二进制兼容

## 构建

```bash
go build -o infverif.exe ./cmd/infverif/
```

## 用法

```
infverif [/code <error code>] [/v] [[/h] | [/w] | [/u] | [/k]]
         [/rulever <Major.Minor.Build> | vnext]
         [/wbuild <Major.Minor.Build>] [/info] [/stampinf]
         [/l <path>] [/osver <TargetOSVersion>] [/product <ias file>]
         [/provider <ProviderName>] <files>
```

### 示例

```bash
# 基本验证
infverif driver.inf

# WHQL 签名要求检查
infverif /h /v driver.inf

# 使用指定规则版本进行签名检查
infverif /h /rulever 25h2 driver.inf

# 通用驱动检查（详细输出）
infverif /v /u driver.inf

# Windows 驱动检查
infverif /v /w driver.inf

# 声明性驱动要求检查
infverif /k driver.inf

# 显示 INF 摘要信息
infverif /info driver.inf

# 查询错误码帮助
infverif /code 1203

# 强制匹配提供商名称
infverif /provider "MyCompany" driver.inf

# 验证 stampinf 前的 INF 文件
infverif /stampinf driver.inf

# 指定 /w 模式最低 Windows build 版本
infverif /w /wbuild 10.0.19041 driver.inf
```

### 命令行参数

**模式**（互斥）：

| 标志    | 说明                              |
| ------- | --------------------------------- |
| `/h`    | WHQL 签名要求检查（标志 `0x80`）  |
| `/c`    | 配置能力检查（标志 `0x01`）       |
| `/u`    | 通用驱动检查（标志 `0x03`）       |
| `/w`    | Windows 驱动检查（标志 `0x07`）   |
| `/k`    | 声明性驱动要求检查（标志 `0x43`） |
| `/info` | 显示 INF 摘要信息（不验证）       |

**选项**：

| 标志        | 参数      | 说明                                            |
| ----------- | --------- | ----------------------------------------------- |
| `/v`        | —         | 详细输出                                        |
| `/code`     | `<code>`  | 查询指定错误码的帮助信息                        |
| `/rulever`  | `<M.m.B>` | `/h` 模式规则版本（`vnext`、`24h2`、`25h2` 等） |
| `/provider` | `<name>`  | 强制匹配提供商名称                              |
| `/stampinf` | —         | 将 `$ARCH$` 视为有效架构                        |
| `/l`        | `<path>`  | HTML 日志输出目录                               |
| `/osver`    | `<ver>`   | 目标 OS 版本过滤                                |
| `/wbuild`   | `<M.m.B>` | `/w` 模式最低 build 版本（默认 `10.0.17763`）   |
| `/product`  | `<file>`  | 产品定义 `.ias` 文件                            |

## 项目结构

```
infverif/
├── cmd/infverif/       # CLI 入口
│   └── main.go
├── pkg/
│   ├── infparser/      # INF 文件解析器（UTF-16LE/BE/UTF-8）
│   │   └── parser.go
│   └── verifier/       # 验证引擎
│       ├── verifier.go   # 核心验证逻辑
│       ├── errordb.go    # 错误码数据库及 HDC 规则
│       └── exceptions.go # 例外表及规则版本
└── go.mod
```

## 许可

MIT License
