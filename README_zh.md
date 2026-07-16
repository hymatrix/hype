# hype

> hymx Node 官方的**脚手架、管理与运行时驱动** CLI。

[English](README.md) | **中文**

`hype` 带你从一个空目录走到一个运行中的 agent:脚手架生成 hymx 项目、驱动 VMDocker V2 模块工作流(构建 → spawn → export)、以及对接 Openclaw / Claude 运行时——全部通过一个二进制、一个 REPL 或内嵌 Web UI 完成。

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
7. [Web UI](#web-ui)
8. [脚手架项目结构](#脚手架项目结构)
9. [从源码构建与发布](#从源码构建与发布)
10. [说明](#说明)

---

## 概述

**是什么。** `hype`(`github.com/hymatrix/hype`)是 hymx Node 的命令行伴侣。它生成 Go 项目脚手架、管理 VM 模块、端到端驱动 VMDocker V2 运行时,并内置一个 Web UI。

**能做什么。**

- **项目脚手架** —— 生成 hymx Node 项目并挂载 VM 模块(`new`、`vmm`、`mount`、`module`、`run`、`get`)。
- **VMDocker V2 工作流** —— 拉取、初始化、构建、spawn、export 基于 profile 的模块(`vmdocker …`)。
- **运行时驱动** —— 通过 hymx SDK 拉起并对接 Openclaw 与 Claude 运行时(`openclaw …`、`claude …`)。
- **数据迁移** —— 在 Redis 与 JSONL 之间搬运进程数据(`db-import`、`db-export`)。
- **交互与 UI** —— 内置 REPL(`repl`)与本地 Web UI(`ui`)。

**生态位置。** `hype` 是 hymx 栈里最外层的编排器:

```
hype (CLI / 编排器)
  └── hymx            平台 / 节点 —— 暴露 /vmm/*,挂载各模块格式
        └── vmdocker  VM 外壳 —— 构建镜像、spawn 与调度容器
              └── vmdocker-agent   镜像内适配器(PID 1),运行你的运行时
```

**怎么运行。** 按需选择:一次性子命令(`hype <cmd>`)、交互式 REPL(无参数运行 `hype`)、或 Web UI(`hype ui`)。

---

## 安装

任选其一:

```bash
# npm(会下载匹配的 GitHub Release 二进制)
npm install -g @hymx/hype

# Go 工具链
go install github.com/hymatrix/hype/cmd/hype@latest   # 或 @v0.1.0

# 从源码构建(详见「从源码构建与发布」)
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

### 运行时(Runtime)

| 命令 | 作用 | 关键参数 |
|------|------|----------|
| `openclaw spawn` | Spawn 一个 Openclaw 进程(可选带 Telegram) | **`-m/--module-id`**、**`-s/--scheduler`**、**`--gateway-token`**、`--model`、`--provider`、`--api-key` |
| `openclaw conf-tg` | 为进程配置 Telegram | **`-p/--pid`**、**`--bot-token`** |
| `openclaw pair-tg` | 批准 Telegram 配对 | **`-p/--pid`**、**`-c/--code`** |
| `openclaw chat` | 向进程发一条 chat 命令 | **`-p/--pid`**、**`-c/--command`** |
| `claude spawn` | Spawn 一个 Claude 运行时进程 | **`-m/--module-id`**、**`-s/--scheduler`**、**`--api-key`**、`--base-url`、`--model` |
| `claude chat` | 向 Claude 进程发一条消息 | **`-p/--pid`**、**`-c/--command`** |
| `claude exec` | 向 Claude 进程发一个通用 prompt | **`--pid`**、**`-p/--prompt`** |

### 数据(Data)

| 命令 | 作用 | 关键参数 |
|------|------|----------|
| `db-import` | 把 JSONL 文件导入 Redis | **`-r/--redis-url`**、**`-f/--file`**、`-F/--force` |
| `db-export` | 从 Redis 导出进程数据为 JSONL | **`-r/--redis-url`**、**`-o/--out`**、`-p/--pid`、`--progress-every` |

### 交互与其他

| 命令 | 作用 | 关键参数 |
|------|------|----------|
| `repl` | 进入交互模式(无参数运行时的默认) | — |
| `ui` | 启动内嵌 Web UI | `--listen`(`127.0.0.1:7788`)、`--timeout-ms`(`90000`) |
| `version` | 打印版本信息 | — |

---

## 详细使用参数

### 通用选项与环境变量回退

以下约定被多个命令共享:

- **`-u/--node-url`** —— hymx 节点 URL。默认 `http://127.0.0.1:8080`。`vmdocker spawn`/`export` 还会读 `VMDOCKER_URL`。
- **`-k/--private-key`** —— 签名私钥。运行时命令的回退顺序:`--private-key` → `HYPE_PRIVATE_KEY` → `PRV_KEY` → `VMDOCKER_PRIVATE_KEY`。
- **`--json`** —— 机器可读输出(`openclaw`、`claude`、`vmdocker spawn`/`export`)。

| 命令组 | 环境变量回退 |
|--------|--------------|
| `vmdocker spawn` | `VMDOCKER_MODULE_ID`、`VMDOCKER_SCHEDULER`、`RUNTIME_TYPE`、`RUNTIME_BACKEND`、`VMDOCKER_URL`、`VMDOCKER_PRIVATE_KEY`/`HYPE_PRIVATE_KEY`/`PRV_KEY` |
| `vmdocker export` | `VMDOCKER_EXPORT_PID`、`VMDOCKER_URL`、`VMDOCKER_PRIVATE_KEY`/`HYPE_PRIVATE_KEY`/`PRV_KEY` |
| `vmdocker module build` | `VMDOCKER_AGENT_BIN`、`VMDOCKER_URL`,以及上面的私钥链(也读检出目录的 `.env`) |
| `claude spawn` | `VMDOCKER_MODULE_ID`、`VMDOCKER_SCHEDULER`、`RUNTIME_BACKEND`、`ANTHROPIC_API_KEY`、`ANTHROPIC_BASE_URL`、`ANTHROPIC_MODEL`/`CLAUDE_MODEL`、`CLAUDE_CODE_FLAGS` |
| `ui` | `OPENCLAW_WEBUI_LISTEN`、`OPENCLAW_WEBUI_TIMEOUT_MS` |

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

### 运行时命令

`openclaw` 和 `claude` 都共享持久 flag:`-u/--node-url`(默认 `http://127.0.0.1:8080`)、`-k/--private-key`、`--json`。

#### `openclaw spawn`
用 module + scheduler + spawn tag 创建一个 Openclaw 进程。

- **`-m/--module-id`**、**`-s/--scheduler`**、**`--gateway-token`**(均必填)。
- `--model`、`--provider`、`--api-key` —— 模型选择。`model=opencode-go/kimi-k2.5` 且 `--provider` 为空,等价于 `--model kimi-k2.5 --provider opencode-go`。若设了 `--api-key` 而 `--model` 无 provider 前缀,则 `--provider` 必填。
- `--runtime-backend` —— `docker` 或 `sandbox`。省略时 `vmdocker` 按操作系统选择(macOS → `sandbox`,Linux → `docker`)。
- `--bot-token`、`--default-account`(`main`)、`--dm-policy`(`open`)、`--allow-from`(`*`)—— 若带 `--bot-token`,`hype` 会立即对新 pid 执行 `ConfigureTelegram`。

```bash
./build/hype openclaw spawn \
  --module-id <moduleId> --scheduler <scheduler> \
  --model kimi-k2.5 --provider opencode-go --api-key <providerApiKey> \
  --gateway-token openclaw-test-token --runtime-backend sandbox \
  --bot-token <telegramBotToken> --default-account main --dm-policy open --allow-from '*' \
  --private-key <privateKey>
```

#### `openclaw conf-tg`
为已存在的进程配置 Telegram(`ConfigureTelegram`)。

- **`-p/--pid`**、**`--bot-token`**(必填)。
- `--default-account`(`main`)、`--dm-policy`(`pairing`)、`--allow-from`(`*`)。`dm-policy=open` 要求 `--allow-from` 含 `*`。

#### `openclaw pair-tg`
批准一次 Telegram 配对(`ApproveTelegramPairing`)。

- **`-p/--pid`**、**`-c/--code`**(必填)。
- `--channel`(`telegram`)、`--dm-policy`(`pairing`)。

#### `openclaw chat`
向进程发一条 chat 命令(`Chat`)。

- **`-p/--pid`**、**`-c/--command`**(必填)。

#### `claude spawn`
创建一个 Claude 运行时进程。env/tag 布局对齐 `vmdocker-agent` 里的 Claude 运行时;spawn 固定设置 `Container-Env-RUNTIME_TYPE=claude`。

- **`-m/--module-id`**、**`-s/--scheduler`**、**`--api-key`**(必填)。
- `--base-url`、`--model`、`--code-flags`、`--runtime-backend`(可选)。

```bash
./build/hype claude spawn \
  --module-id <moduleId> --scheduler <scheduler> \
  --api-key <anthropicApiKey> --base-url <anthropicBaseURL> --model <anthropicModel> \
  --runtime-backend sandbox --private-key <privateKey>
```

#### `claude chat` / `claude exec`
`chat` 发送 `Action=Chat`;`exec` 发送 `Action=Execute`。`hype claude -p "..." --pid <pid>` 是 `hype claude exec --pid <pid> -p "..."` 的快捷方式。

- `chat`:**`-p/--pid`**、**`-c/--command`**(必填)。
- `exec`:**`--pid`**、**`-p/--prompt`**(必填)。

```bash
./build/hype claude exec --pid <pid> --prompt "你好，请介绍一下自己" --private-key <privateKey>
```

完整的 Claude 命令指南见 [`tools/claude/README.md`](tools/claude/README.md)。

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
用 `hype`(无参数)或 `hype repl` 启动。用 `HYPE_PRIVATE_KEY=0x... hype` 预置运行时命令的私钥。

- 直接输入子命令,无需 `hype` 前缀——如 `version`、`new -m ...`、`db-export ...`。
- 漏掉的必填 flag 会被逐个交互式提示。
- `help` / `?` 显示 `hype --help`;`exit` / `quit` / Ctrl-D 退出。
- 行首加 `!` 通过 `bash` 执行,如 `!ls`。

#### `version`
打印版本信息(`hype version`、`hype --version` 或 `hype -v`),包含 hype 版本、内嵌的 hymx 节点版本,以及 Go 构建细节。

---

## Web UI

内嵌在 `hype` 二进制中、面向 `openclaw`、`claude`、`vmdocker` 命令的本地 Web UI。

```bash
hype ui
# Openclaw UI listening on http://127.0.0.1:7788
```

- 仅本地的 API server(`127.0.0.1:7788`);它执行当前的 `hype` 二进制来跑 `openclaw`、`claude`、`vmdocker` 子命令。
- 展示结构化 JSON 与原始 stdout/stderr;在命令预览中对敏感字段打码。
- 密钥只保存在内存中(不做 `localStorage` 持久化)。
- 在内存中导入本地 `.env`(文件选择器或路径),预填匹配的 Openclaw 字段(`moduleId`、来自 `VMDOCKER_SCHEDULER` 的 `scheduler`、`model`、`provider`、`apiKey`、Telegram 设置),并用于 `vmdocker init`。`View Env` 展示每一条导入的条目。

**flag 与环境变量:**

- `--listen` —— 默认 `127.0.0.1:7788`(环境变量 `OPENCLAW_WEBUI_LISTEN`)。
- `--timeout-ms` —— 默认 `90000`(环境变量 `OPENCLAW_WEBUI_TIMEOUT_MS`)。
- 若 UI 的 privateKey 字段为空,会使用 `HYPE_PRIVATE_KEY` 或 `PRV_KEY`。

**相对路径**从启动 `hype ui` 的工作目录解析。从 `build/` 启动(`cd build && ./hype ui`)会把 `./.env` 和 `./vmdocker` 解析到 `build/` 下;从仓库根启动(`./build/hype ui`)则解析到根目录下。

开发模式(Vite 在 `http://127.0.0.1:5173`,API 在 `http://127.0.0.1:7788`):

```bash
./frontend/scripts/dev.sh
```

更多细节:[`frontend/README.md`](frontend/README.md)。

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
# 用 make
make build            # -> build/hype

# 直接用 go(先构建前端资产——它们会被内嵌)
npm --prefix ./frontend ci
npm --prefix ./frontend run build
go build -o build/hype ./cmd/hype
```

### 从源码安装

```bash
npm --prefix ./frontend ci && npm --prefix ./frontend run build
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
- **已知限制:** 内嵌 Web UI 的 VMDocker Get 动作仍会发出已移除的 `--version` flag,不属于纯 CLI 的 V2 工作流。
- Claude 命令指南:[`tools/claude/README.md`](tools/claude/README.md)。
