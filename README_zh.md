# hype

> hymx Node 官方的**脚手架与管理** CLI。

[English](README.md) | **中文**

`hype` 带你从一个空目录走到一个运行中的模块:脚手架生成 hymx 项目,再端到端驱动 VMDocker V2 工作流——拉取、构建、spawn、export 基于 profile 的模块——全部通过一个二进制或一个交互式 REPL 完成。

- **第一次用?** 直接看 [快速上手](#快速上手),5 分钟跑通端到端流程。
- **想查某个命令?** 看 [命令列表](#命令列表) 速查表。

---

## 目录

1. [概述](#概述)
2. [安装](#安装)
3. [快速上手](#快速上手)
4. [vmdocker 使用](#vmdocker-使用)
5. [命令列表](#命令列表)
6. [详细使用参数](#详细使用参数)
7. [脚手架项目结构](#脚手架项目结构)
8. [从源码构建与发布](#从源码构建与发布)
9. [说明](#说明)

---

## 概述

**是什么。** `hype`(`github.com/hymatrix/hype`)是 hymx Node 的命令行伴侣。它生成 Go 项目脚手架、管理 VM 模块,并端到端驱动 VMDocker V2 运行时。

**能做什么。**

- **项目脚手架** —— 生成 hymx Node 项目并挂载 VM 模块(`new`、`vmm`、`mount`、`module`、`run`、`get`)。
- **VMDocker V2 工作流** —— 拉取、初始化、构建、spawn、export 基于 profile 的模块(`vmdocker …`)。
- **数据迁移** —— 在 Redis 与 JSONL 之间搬运进程数据(`db-import`、`db-export`)。
- **交互** —— 内置 REPL(`repl`,或直接无参数运行 `hype`)。

**生态位置。** `hype` 是 hymx 栈里最外层的编排器:

```
hype (CLI / 编排器)
  └── hymx            平台 / 节点 —— 暴露 /vmm/*,挂载各模块格式
        └── vmdocker  VM 外壳 —— 构建镜像、spawn 与调度容器
              └── vmdocker-agent   镜像内适配器(PID 1),运行你的运行时
```

**怎么运行。** 按需选择:一次性子命令(`hype <cmd>`),或交互式 REPL(无参数运行 `hype`)。

---

## 安装

任选其一:

```bash
# npm(会下载匹配的 GitHub Release 二进制)
npm install -g @hymx/hype

# Go 工具链
go install github.com/hymatrix/hype/cmd/hype@latest   # 或 @v0.1.0

# 从源码构建
make build        # -> build/hype
```

然后确保二进制在 `PATH` 中:

- 用 `go install` 时,把 `$(go env GOPATH)/bin`(或 `GOBIN`)加入 `PATH`。
- npm 分发目前支持 macOS/Linux 的 `x64` 与 `arm64`。

验证:

```bash
hype --version    # 或:hype -v
```

本文中,`hype …` 与本地的 `./build/hype …` 可互换使用。

---

## 快速上手

跑通一个运行中进程的最快路径:**VMDocker V2** 工作流,从拉取到 spawn。

### 前置条件

- 本地可用的 **Docker** 与 **Redis**(`vmdocker init` 会用到)。
- **Go 1.24+**(用于构建拉取下来的 VMDocker 节点)。
- 用于 构建 / spawn / export 的**签名私钥**(`0x…`)。
- spawn 用的 **scheduler 地址**。
- 模块构建用的 **`vmdocker-agent`** 二进制(`hype` 不会替你构建它)。

### 步骤

```bash
# 1. 拉取并构建 VMDocker V2 检出(默认 clone 到 ./vmdockerv2)
hype vmdocker get

# 2. 启动 Redis + 本地节点,再执行 examples init。
#    ./local.env 至少要包含:VMDOCKER_PRIVATE_KEY=0x...
hype vmdocker init --dir ./vmdockerv2 --env-file ./local.env

# 3. 生成一个 agent profile 脚手架(目标目录须为空或不存在)
hype vmdocker profile init --dir ./agent --from docker/sandbox-templates:claude-code

# 4. 由 profile 构建并签名模块
hype vmdocker module build \
  --dir ./vmdockerv2 \
  --profile ./agent/profile.toml \
  --agent-bin ./bin/vmdocker-agent \
  --private-key 0x...

# 5. 用模块 spawn 一个进程(打印 pid)
hype vmdocker spawn \
  --module-id <module-id> \
  --scheduler <scheduler-address> \
  --runtime-type claude \
  --runtime-backend sandbox \
  --private-key 0x...

# 6.(可选)把运行中的进程导出成可复用的 module id
hype vmdocker export --pid <pid> --private-key 0x...
```

**成功的样子:** 第 5 步打印 `spawn ok, pid: <pid>`,第 6 步打印 `export ok, module id: <module-id>`。把返回的 module id 直接再喂给 `hype vmdocker spawn --module-id <id>` 即可。

想先交互式探索?无参数运行 `hype` 进入 REPL——你漏掉的必填 flag 它会逐个提示你输入。

> 下一步:读 [vmdocker 使用](#vmdocker-使用) 了解每一步在做什么,或看深入版 [VMDocker V2 CLI 工作流](docs/vmdocker-v2-cli-workflow.md)(英文)。

---

## vmdocker 使用

`vmdocker` 命令组驱动完整的 VMDocker V2 生命周期。`hype` 会把各种 ID 打印到 stdout,但**从不把 module id 或 process id 写回 `.env`**。

### 核心概念

- **Module(模块)** —— 一个签名后、自包含的单元(`mod-<id>.json`),携带容器镜像加一份 `profile.toml`。spawn 时若本地没有该镜像,会从模块里加载。
- **`profile.toml`** —— 模块的声明式配方:一个 `[dockerfile]` 段(作为标准化 Dockerfile 生成器的输入——你不用手写 Dockerfile)加一个 `[vmdocker]` 段(`public` 导出白名单)。
- **Spawn** —— 从模块创建一个进程(pid)。`--runtime-type`(如 `claude`)决定镜像内适配器的就绪判定;`--runtime-backend` 为 `docker` 或 `sandbox`。
- **Export** —— 把运行中进程的 public 状态克隆成一个新的、可复用的 module id。没有 `respawn`;用 `export`,再用返回的 id 去 `spawn`。

### 最小 `profile.toml`

`hype vmdocker profile init` 会写出这份脚手架(外加 `bin/.keep`、`skills/soul.md`、`persona/style.md`):

```toml
[dockerfile]
# 完整的 base 镜像名,原样用作 Dockerfile FROM。
FROM = "docker/sandbox-templates:claude-code"

# 用户可执行文件目录,会被 copy 到 /usr/local/bin。必填,可为空。
bin = "bin"

# 可选启动命令(Dockerfile CMD 语法);适配器仍是 ENTRYPOINT。
# CMD = ["your-engine", "--serve"]

# 可选的跨发行版工具包,在镜像构建时安装。
tools = []

# 可选的 Dockerfile RUN 体(不含开头的 "RUN ")。
RUN = []

[vmdocker]
# 相对 HOME 的导出白名单;未列出的一切保持私有。
public = ["~/skills/*", "~/persona/*", "~/.hermes/plugin/*"]
```

只有 `[dockerfile].FROM` 和 `[dockerfile].bin` 是必填;`CMD` 可选。

### 工作流一览

| 步骤 | 命令 | 作用 |
|------|------|------|
| 拉取 | `vmdocker get` | Clone 一个 `vmdockerv2` ref 并构建 `build/hymx-node`。 |
| 初始化 | `vmdocker init` | 启动 Redis + 节点,等待健康检查,执行 examples init。 |
| Profile | `vmdocker profile init` | 生成一个 `profile.toml` agent 目录脚手架。 |
| 构建 | `vmdocker module build` | 由 profile 构建并签名模块。 |
| Spawn | `vmdocker spawn` | 从模块 spawn 一个进程(打印 pid)。 |
| Export | `vmdocker export` | 把运行中进程导出成新的 module id。 |
| 清理 | `vmdocker clean` | 停止本地节点并删除生成的运行时文件。 |

随时用下面命令重置本地状态:

```bash
hype vmdocker clean --dir ./vmdockerv2
```

每一步的具体行为、输入优先级与 tag 映射,见完整的 [VMDocker V2 CLI 工作流](docs/vmdocker-v2-cli-workflow.md)(英文)与下方 [详细使用参数](#详细使用参数)。

---

## 命令列表

运行 `hype <command> --help` 查看最权威、最新的参数。必填 flag 用**加粗**。

### 脚手架(Scaffolding)

| 命令 | 作用 | 关键参数 |
|------|------|----------|
| `new` | 新建一个 hymx Go 项目脚手架 | **`-m/--module`**、`-o/--out`(`.`) |
| `get` | 通过 go 工具链拉取一个 VMM 包并挂载 | **`-p/--package`** |
| `vmm` | 生成 VM 模块并自动挂载进 `cmd/main.go` | **`-n/--name`**、**`-f/--format`** |
| `mount` | 把已存在的 VM 模块挂载进 `cmd/main.go` | **`-n/--name`** |
| `module` | 按名字生成并挂载模块 | **`-n/--name`**、`-u/--node-url`、`-k/--private-key` |
| `run` | 运行生成的项目(`cd cmd && go run ./`) | `-m/--mode`(`normal`\|`rebuild`) |

### vmdocker

| 命令 | 作用 | 关键参数 |
|------|------|----------|
| `vmdocker get` | Clone 一个 `vmdockerv2` ref 并构建 `hymx-node` | `--ref`(`main`)、`--dir`(`./vmdockerv2`) |
| `vmdocker init` | 启动 Redis + 节点,执行 examples init | `--dir`、**`--env-file`** |
| `vmdocker profile init` | 生成 `profile.toml` agent 目录脚手架 | **`--dir`**、**`--from`** |
| `vmdocker module build` | 由 profile 构建并签名模块 | **`--profile`**、`--agent-bin`、`--dir`、`--node-url`、`--private-key` |
| `vmdocker spawn` | 从模块 spawn 一个进程 | **`-m/--module-id`**、**`-s/--scheduler`**、**`-k/--private-key`**、`--runtime-type`、`--runtime-backend`、`--env` |
| `vmdocker export` | 把运行中进程导出成 module id | **`-p/--pid`**、**`-k/--private-key`** |
| `vmdocker clean` | 停止节点 + 删除生成的运行时文件 | `--dir` |

### 数据(Data)

| 命令 | 作用 | 关键参数 |
|------|------|----------|
| `db-import` | 把 JSONL 文件导入 Redis | **`-r/--redis-url`**、**`-f/--file`**、`-F/--force` |
| `db-export` | 从 Redis 导出进程数据为 JSONL | **`-r/--redis-url`**、**`-o/--out`**、`-p/--pid`、`--progress-every` |

### 交互与其他

| 命令 | 作用 | 关键参数 |
|------|------|----------|
| `repl` | 进入交互模式(无参数运行时的默认) | — |
| `version` | 打印版本信息 | — |

---

## 详细使用参数

### 通用选项与环境变量回退

以下约定被多个命令共享:

- **`-u/--node-url`** —— hymx 节点 URL。默认 `http://127.0.0.1:8080`。`vmdocker spawn`/`export` 还会读 `VMDOCKER_URL`。
- **`-k/--private-key`** —— 签名私钥。回退顺序:`--private-key` → `HYPE_PRIVATE_KEY` → `PRV_KEY` → `VMDOCKER_PRIVATE_KEY`。
- **`--json`** —— 机器可读输出(`vmdocker spawn`/`export`)。

| 命令 | 环境变量回退 |
|------|--------------|
| `vmdocker spawn` | `VMDOCKER_MODULE_ID`、`VMDOCKER_SCHEDULER`、`RUNTIME_TYPE`、`RUNTIME_BACKEND`、`VMDOCKER_URL`、`VMDOCKER_PRIVATE_KEY`/`HYPE_PRIVATE_KEY`/`PRV_KEY` |
| `vmdocker export` | `VMDOCKER_EXPORT_PID`、`VMDOCKER_URL`、`VMDOCKER_PRIVATE_KEY`/`HYPE_PRIVATE_KEY`/`PRV_KEY` |
| `vmdocker module build` | `VMDOCKER_AGENT_BIN`、`VMDOCKER_URL`,以及上面的私钥链(也读检出目录的 `.env`) |

### 脚手架命令

#### `new`
新建一个 hymx Go 项目脚手架。用给定 module 初始化 `go.mod`,并在生成的项目里运行 `go mod tidy`。

- **`-m/--module`**(必填)—— Go module 名,如 `github.com/<user>/<pkg>`。
- `-o/--out` —— 输出基目录。默认 `.`。

```bash
hype new -m github.com/<user>/<pkg> -o ./_sandbox
```

包名由输出目录名派生。

#### `get`
通过 Go 工具链拉取一个 VMM 包并挂载。(注意别和 `vmdocker get` 混淆,后者是 clone VMDocker V2 仓库并构建节点。)

- **`-p/--package`**(必填)—— VMM 包的 Go module 路径。

#### `vmm`
生成一个 VM 模块并自动挂载进 `cmd/main.go`。在生成的项目根目录(含 `cmd/main.go` 的目录)下运行。会插入 import 和 `s.Mount(<vmm>Schema.ModuleFormat, <vmm>.Spawn)`。

- **`-n/--name`**(必填)—— vmm 名字。
- **`-f/--format`**(必填)—— 模块格式,如 `hymx.token.foo.0.0.1`。

#### `mount`
把已存在的 VM 模块挂载进 `cmd/main.go`。从 `<projectDir>/<vmm>/schema/schema.go` 读取模块格式。

- **`-n/--name`**(必填)—— vmm 名字。

#### `module`
按名字生成并挂载模块,从 `<projectDir>/<name>/schema/schema.go` 读取 `ModuleFormat`。经 SDK 保存的模块落在 `cmd/mod/mod-<itemId>.json`。

- **`-n/--name`**(必填)—— 模块名字。
- `-u/--node-url` —— 节点 URL。默认 `http://127.0.0.1:8080`。
- `-k/--private-key` —— 以太坊 ECDSA secp256k1 私钥 hex(`0x` 前缀)。

#### `run`
运行生成的项目。从项目根目录执行 `cd cmd && go run ./ [--mode <mode>]`。

- `-m/--mode` —— 启动模式,`normal` 或 `rebuild`。默认 `normal`。

### vmdocker 命令

#### `vmdocker get`
Clone `https://github.com/cryptowizard0/vmdockerv2.git` 并构建 `./build/hymx-node`。

- `--ref` —— Git 分支、tag 或可达的 commit。默认 `main`。(commit SHA 必须能从远端广播的 ref 可达。)
- `--dir` —— 目标 clone 目录。默认 `./vmdockerv2`。

若检出已存在,`hype` 会校验它是 VMDocker V2 仓库、fetch 该 ref、在被跟踪文件有改动时拒绝切换,并在需要时重建 `build/hymx-node`。

```bash
hype vmdocker get
hype vmdocker get --dir ./_sandbox/vmdockerv2 --ref feature/profile
```

#### `vmdocker init`
启动本地服务并执行 examples 初始化。要求 `<dir>/build/hymx-node` 已存在。通过 Docker 容器 `hype-vmdocker-redis` 在 `6379` 端口启动 Redis,用 `./build/hymx-node start --config ./cmd/config.yaml` 启动节点,等待 `http://127.0.0.1:8080/info`,再把解析后的 `.env` 注入 `go run ./examples init`。

- `--dir` —— VMDocker 检出目录。默认 `./vmdockerv2`。
- **`--env-file`**(必填)—— 给 `examples init` 用的 `.env`。必须含 `VMDOCKER_PRIVATE_KEY`;`VMDOCKER_URL=http://127.0.0.1:8080` 会被自动注入。

```bash
hype vmdocker init --env-file ./local.env
```

#### `vmdocker profile init`
创建一个由 `vmdockerv2/testagent` 派生的最小 profile 脚手架。写出 `profile.toml`、`bin/.keep`、`skills/soul.md`、`persona/style.md`。拒绝写入非空目录。

- **`--dir`**(必填)—— 目标 agent profile 目录。
- **`--from`**(必填)—— 用作 `profile.toml` 中 `FROM` 的完整 base 镜像名。

```bash
hype vmdocker profile init --dir ./agent --from docker/sandbox-templates:claude-code
```

#### `vmdocker module build`
把模块创建委托给 VMDocker V2:在检出目录的 `cmd/` 下运行 `go run ./module --profile <profile> --agent-bin <agent>`,流式转发其输出,然后把生成的 `cmd/mod-<itemId>.json` 移动到 `cmd/mod/`,以便本地启动的节点能加载。`hype` 不会构建或下载 agent 二进制。

- **`--profile`**(必填)—— `profile.toml` 路径。
- `--agent-bin` —— `vmdocker-agent` 二进制路径。回退到 `VMDOCKER_AGENT_BIN`。
- `--dir` —— VMDocker V2 检出。默认 `./vmdockerv2`。
- `--node-url` —— 优先级:flag → `VMDOCKER_URL` → 检出 `.env` → `http://127.0.0.1:8080`。
- `--private-key` —— 优先级:flag → `VMDOCKER_PRIVATE_KEY` → `HYPE_PRIVATE_KEY` → `PRV_KEY` → 检出 `.env`。

```bash
hype vmdocker module build \
  --dir ./vmdockerv2 --profile ./agent/profile.toml \
  --agent-bin ./bin/vmdocker-agent --private-key 0x...
```

#### `vmdocker spawn`
从模块 spawn 一个进程。只打印 pid。

- **`-m/--module-id`**(必填)—— module ID。环境变量:`VMDOCKER_MODULE_ID`。
- **`-s/--scheduler`**(必填)—— scheduler 地址。环境变量:`VMDOCKER_SCHEDULER`。
- **`-k/--private-key`**(必填)—— 签名私钥。环境变量链同上。
- `--runtime-type` —— 运行时类型。环境变量:`RUNTIME_TYPE`。
- `--runtime-backend` —— `docker` 或 `sandbox`。环境变量:`RUNTIME_BACKEND`。
- `--env KEY=VALUE` —— 可重复的容器环境变量赋值。
- `-u/--node-url` —— 节点 URL。环境变量:`VMDOCKER_URL`;默认 `http://127.0.0.1:8080`。
- `--json` —— JSON 输出。

**Tag 映射:** `--runtime-type claude` → `Container-Env-RUNTIME_TYPE=claude`;`--env TOKEN=a=b` → `Container-Env-TOKEN=a=b`;`--runtime-backend docker` → `Runtime-Backend=docker`。`RUNTIME_TYPE` 为 `--runtime-type` 保留;env key 须匹配 `^[A-Za-z_][A-Za-z0-9_]*$`;重复的 `--env` key 会被拒绝。

```bash
hype vmdocker spawn \
  --module-id <id> --scheduler <address> \
  --runtime-type claude --runtime-backend sandbox \
  --env TOKEN=a=b --private-key 0x...
```

#### `vmdocker export`
向运行中的进程发送 `Action=Export`,打印 `export ok, module id: <id>`。返回的 id 可直接喂给 `vmdocker spawn --module-id <id>`。

- **`-p/--pid`**(必填)—— 进程 ID。环境变量:`VMDOCKER_EXPORT_PID`。
- **`-k/--private-key`**(必填)—— 签名私钥。环境变量链同上。
- `-u/--node-url` —— 节点 URL。环境变量:`VMDOCKER_URL`;默认 `http://127.0.0.1:8080`。
- `--json` —— JSON 输出。

#### `vmdocker clean`
停止 `cmd/hymx-*.lock` 记录的本地节点进程,删除 `cmd/hymx-*.lock`、`cmd/hymx_*.log`、`cmd/hymx-node` 和 `cmd/mod/*.json`,并移除托管的 Redis 容器。**不会**删除 `build/hymx-node`、根 `mod/*.json`、profile、agent 或检出目录。

- `--dir` —— VMDocker V2 检出。默认 `./vmdockerv2`。

### 数据命令

#### `db-import`
把 JSONL 数据文件导入 Redis,对每一项调用 `IDB.Commit`。每行是 `{pid, nonce, msg, assign}`;各项按 `nonce` 排序,必须从 0 开始且连续递增。

- **`-r/--redis-url`**(必填)—— 如 `redis://@localhost:6379/0`。
- **`-f/--file`**(必填)—— `.jsonl` 文件路径。
- `-F/--force` —— 即使 `msg.Id` 已存在也写入(否则跳过重复项)。

```bash
hype db-import --redis-url redis://@localhost:6379/0 --file ./data.jsonl
```

#### `db-export`
从 Redis 导出进程数据为 JSONL(支持 `.gz`)。

- **`-r/--redis-url`**(必填)—— Redis 连接 URL。
- **`-o/--out`**(必填)—— 输出文件(`.jsonl` / `.jsonl.gz`),当 `--pid` 为空时可为一个目录。
- `-p/--pid` —— 进程 id。**可选**:留空则导出全部进程。
- `--progress-every` —— 每 N 行打印一次进度。默认 `1000`。

```bash
hype db-export --redis-url redis://@localhost:6379/0 --pid process-123 --out ./process-123.jsonl.gz
hype db-export --redis-url redis://@localhost:6379/0 --out ./exports   # 全部进程 -> 目录
```

### 交互与其他

#### REPL
用 `hype`(无参数)或 `hype repl` 启动。用 `HYPE_PRIVATE_KEY=0x... hype` 预置联网命令的私钥。

- 直接输入子命令,无需 `hype` 前缀——如 `version`、`new -m ...`、`db-export ...`。
- 漏掉的必填 flag 会被逐个交互式提示。
- `help` / `?` 显示 `hype --help`;`exit` / `quit` / Ctrl-D 退出。
- 行首加 `!` 通过 `bash` 执行,如 `!ls`。

#### `version`
打印版本信息(`hype version`、`hype --version` 或 `hype -v`),包含 hype 版本、内嵌的 hymx 节点版本,以及 Go 构建细节。

---

## 脚手架项目结构

`hype new` 在 `<out>/<pkg>/` 下生成一个项目:

```
<out>/<pkg>/
├── cmd/
│   ├── main.go
│   ├── flags.go
│   ├── const.go
│   ├── cmds.go
│   ├── cfgchainkit.go
│   ├── cfgnode.go
│   ├── cfgpay.go
│   ├── config.yaml
│   ├── config_chainkit.yaml
│   ├── config_payment.yaml
│   ├── config_test_network.yaml
│   └── mod/
│       ├── *.json                 # 从模板复制(去掉 .tmpl 后缀)
│       └── mod-<itemId>.json      # 执行 SDK 模块保存时生成
└── <pkg>/<pkg>.go                 # 接口文件
```

从项目根目录构建生成的项目:

```bash
cd <out>/<pkg>
go build -o ./<pkg> ./cmd
```

脚手架依赖(如 `github.com/spf13/viper`、`github.com/urfave/cli/v2`、`github.com/hymatrix/hymx`)在生成时通过 `go mod tidy` 拉取。

---

## 从源码构建与发布

### 构建

```bash
make build                        # -> build/hype
# 或直接用 go:
go build -o build/hype ./cmd/hype
```

### 从源码安装

```bash
go install ./cmd/hype     # 或:make install
```

### 发布流程(维护者)

- 推送一个 `v*` tag 触发 GoReleaser,上传 GitHub Release 资产。
- 发布 GitHub Release 触发 `@hymx/hype` 的 npm publish 作业,该作业会先校验 `checksums.txt` 与所有支持的 tarball 可达。
- 在 GitHub Actions secrets 中配置 `NPM_TOKEN` 用于 npm 发布。

---

## 说明

- **`hype get` 与 `vmdocker get` 是不同命令**:`hype get` 通过 go 工具链拉取一个 VMM 包;`vmdocker get` 是 clone VMDocker V2 仓库并构建节点。
- `hype` 从不把 module id 或 process id 写进 `.env`。
