# InfVerif (Go)

[English](README.md)

使用 Go 语言重新实现微软 `InfVerif.exe`（Driver Package INF Verifier），原始二进制版本 v10.0.22621.382。

## 概述

InfVerif 用于验证 Windows 驱动程序包 `.inf` 文件是否符合微软的规范要求。

## 功能特性

- **多种验证模式**，与原始二进制的位标志架构一致：
  - `/c` — 配置能力检查（标志 `0x01`）
  - `/u` — 通用驱动检查（标志 `0x03`）
  - `/w` — Windows 驱动检查（标志 `0x07`）
  - `/k` — Windows Update 提交检查（标志 `0x43`）
- **INF 信息显示**（`/info`）— 文件哈希、Family ID、驱动类型、设备信息
- **依赖分析**（`/depends`）— Include/Needs 依赖关系树
- **输出格式**：控制台（默认）、MSBuild（`/msbuild`）、CSV（`/csv`）
- **错误管理**：`/werror`、`/errorlist`、`/errorlevel`
- **文件处理**：通配符、`/recurse`、`/exclude`
- 详细模式（`/v`）输出与原始二进制完全一致

## 构建

```bash
go build -o infverif.exe ./cmd/infverif/
```

## 用法

```
infverif [/v] [[/c] | [/u] | [/w] | [/k]] [/wbuild <Major.Minor.Build>]
         [/info] [/depends] [/stampinf] [/l <path>]
         [/osver <TargetOSVersion>] [/product <ias file>]
         [/csv <file>] [/errorlist <file>] [/errorlevel <n>]
         [/werror] [/exclude <file>] [/levelsort] [/msbuild]
         [/inbox] [/append] [/fileroot <path>] [/recurse] <files>
```

### 示例

```bash
# 基本验证
infverif driver.inf

# 通用驱动检查（详细输出）
infverif /v /u driver.inf

# Windows 驱动检查
infverif /v /w driver.inf

# Windows Update 提交检查
infverif /k driver.inf

# 显示 INF 摘要信息
infverif /info driver.inf

# 显示 Include/Needs 依赖
infverif /depends driver.inf

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

| 标志          | 参数      | 说明                                       |
| ------------- | --------- | ------------------------------------------ |
| `/v`          | —         | 详细输出                                   |
| `/c`          | —         | 配置能力检查模式                           |
| `/u`          | —         | 通用驱动检查模式                           |
| `/w`          | —         | Windows 驱动检查模式                       |
| `/k`          | —         | Windows Update 提交检查模式                |
| `/info`       | —         | 显示 INF 摘要信息（不验证）                |
| `/depends`    | —         | 显示 Include/Needs 依赖                    |
| `/stampinf`   | —         | 将 `$ARCH$` 视为有效架构                   |
| `/l`          | `<path>`  | HTML 日志输出目录                          |
| `/osver`      | `<ver>`   | 目标 OS 版本过滤                           |
| `/wbuild`     | `<M.m.B>` | `/w` 模式最低 build 版本                   |
| `/product`    | `<file>`  | 产品定义 `.ias` 文件                       |
| `/csv`        | `<file>`  | CSV 输出文件                               |
| `/msbuild`    | —         | MSBuild 兼容错误格式                       |
| `/errorlist`  | `<file>`  | 错误抑制列表（CSV）                        |
| `/errorlevel` | `<n>`     | 错误级别阈值（1=ERROR, 2=WARNING, 3=INFO） |
| `/werror`     | —         | 将警告视为错误                             |
| `/exclude`    | `<file>`  | 文件排除列表                               |
| `/levelsort`  | —         | 按错误级别排序输出                         |
| `/inbox`      | —         | 收件箱驱动验证模式                         |
| `/append`     | —         | CSV 追加模式（不覆盖）                     |
| `/fileroot`   | `<path>`  | 文件根目录（路径解析）                     |
| `/recurse`    | —         | 递归搜索子目录                             |

## 项目结构

```
infverif/
├── cmd/infverif/       # CLI 入口
│   └── main.go
├── pkg/
│   ├── infparser/      # INF 文件解析器（UTF-16LE/BE/UTF-8）
│   │   └── parser.go
│   └── verifier/       # 验证引擎
│       └── verifier.go
└── go.mod
```

## 许可

MIT License
