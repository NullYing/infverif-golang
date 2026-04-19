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
- **依赖分析**（`/depends`）— Include/Needs 依赖关系树
- **语法分析**（`/syntax`）— INF 指令语法及最低 OS 版本
- **错误码查询**（`/code`）— 按错误码查询描述信息
- **规则版本控制**（`/rulever`）— 指定 `/h` 模式的规则版本（命名版本：`vnext`、`24h2`、`25h2` 等）
- **提供商验证**（`/provider`）— 强制匹配提供商名称
- **例外管理**：`/noexceptions`、`/showexceptions`、`/hdcrules`
- **输出格式**：控制台（默认）、MSBuild（`/msbuild`）、CSV（`/csv`）
- **错误管理**：`/werror`、`/errorlist`、`/errorlevel`
- **文件处理**：通配符、`/recurse`、`/exclude`
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
         [/provider <ProviderName>] [/noexceptions] [/syntax]
         [/csv <file>] [/errorlist <file>] [/errorlevel <n>]
         [/werror] [/exclude <file>] [/levelsort] [/msbuild]
         [/inbox] [/append] [/fileroot <path>] [/recurse] <files>
```

### 示例

```bash
# 基本验证
infverif driver.inf

# WHQL 签名要求检查
infverif /h /v driver.inf

# 使用指定规则版本进行签名检查
infverif /h /rulever 25h2 driver.inf

# 签名检查（禁用例外）
infverif /h /noexceptions driver.inf

# 通用驱动检查（详细输出）
infverif /v /u driver.inf

# Windows 驱动检查
infverif /v /w driver.inf

# 声明性驱动要求检查
infverif /k driver.inf

# 显示 INF 摘要信息
infverif /info driver.inf

# 显示 Include/Needs 依赖
infverif /depends driver.inf

# INF 语法分析
infverif /syntax driver.inf

# 查询错误码帮助
infverif /code 1203

# 强制匹配提供商名称
infverif /provider "MyCompany" driver.inf

# 显示 HDC 规则和例外
infverif /hdcrules
infverif /showexceptions

# 输出到 CSV 文件
infverif /csv results.csv /recurse *.inf

# 将警告视为错误
infverif /werror /u driver.inf

# MSBuild 兼容输出格式
infverif /msbuild /w driver.inf

# 抑制特定错误（1310-1319 不可抑制）
infverif /errorlist allowed.csv driver.inf
```

### 命令行参数

**模式**（互斥）：

| 标志       | 说明                              |
| ---------- | --------------------------------- |
| `/h`       | WHQL 签名要求检查（标志 `0x80`）  |
| `/c`       | 配置能力检查（标志 `0x01`）       |
| `/u`       | 通用驱动检查（标志 `0x03`）       |
| `/w`       | Windows 驱动检查（标志 `0x07`）   |
| `/k`       | 声明性驱动要求检查（标志 `0x43`） |
| `/info`    | 显示 INF 摘要信息（不验证）       |
| `/depends` | 显示 Include/Needs 依赖           |
| `/syntax`  | INF 语法报告及最低 OS 版本        |

**选项**：

| 标志              | 参数      | 说明                                       |
| ----------------- | --------- | ------------------------------------------ |
| `/v`              | —         | 详细输出                                   |
| `/code`           | `<code>`  | 查询指定错误码的帮助信息                   |
| `/rulever`        | `<M.m.B>` | `/h` 模式规则版本（`vnext`、`24h2` 等）    |
| `/provider`       | `<name>`  | 强制匹配提供商名称                         |
| `/noexceptions`   | —         | 在 `/h` 模式中禁用例外                     |
| `/attestation`    | —         | 认证签名模式                               |
| `/hdcrules`       | —         | 显示 HDC 错误码规则                        |
| `/showexceptions` | —         | 显示所有例外条目                           |
| `/stampinf`       | —         | 将 `$ARCH$` 视为有效架构                   |
| `/l`              | `<path>`  | HTML 日志输出目录                          |
| `/osver`          | `<ver>`   | 目标 OS 版本过滤                           |
| `/wbuild`         | `<M.m.B>` | `/w` 模式最低 build 版本                   |
| `/product`        | `<file>`  | 产品定义 `.ias` 文件                       |
| `/dll`            | `<path>`  | 外部验证 DLL                               |
| `/csv`            | `<file>`  | CSV 输出文件                               |
| `/msbuild`        | —         | MSBuild 兼容错误格式                       |
| `/errorlist`      | `<file>`  | 错误抑制列表（CSV）                        |
| `/errorlevel`     | `<n>`     | 错误级别阈值（1=ERROR, 2=WARNING, 3=INFO） |
| `/werror`         | —         | 将警告视为错误                             |
| `/exclude`        | `<file>`  | 文件排除列表                               |
| `/levelsort`      | —         | 按错误级别排序输出                         |
| `/inbox`          | —         | 收件箱驱动验证模式                         |
| `/append`         | —         | CSV 追加模式（不覆盖）                     |
| `/fileroot`       | `<path>`  | 文件根目录（路径解析）                     |
| `/recurse`        | —         | 递归搜索子目录                             |
| `/logging`        | —         | 启用日志输出                               |
| `/verboseparams`  | —         | 显示 InfVerif 参数标志                     |
| `/samples`        | —         | 示例模式                                   |
| `/wdk`            | —         | WDK 模式                                   |

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
