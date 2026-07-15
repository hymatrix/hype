# VMDocker V2 Profile Workflow CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Hype's VMDocker V1 CLI integration with a composable VMDocker V2 profile workflow covering get, init, profile scaffolding, module build, spawn, and export.

**Architecture:** Keep repository and external build behavior behind `internal/vmdocker.Manager` and its fakeable runner. Delegate module construction to `vmdockerv2/cmd/module`; implement generic spawn/export in `internal/cli` with the existing Hymx SDK. Split profile, module-build, and runtime code by responsibility.

**Tech Stack:** Go 1.23, Cobra, Hymx SDK v0.4.8, `goar/schema.Tag`, standard-library filesystem/process APIs, Go `testing`.

## Global Constraints

- Work only in `/Users/webbergao/work/src/HymxWorkspace/hype`; run Git there.
- Follow `/Users/webbergao/work/src/HymxWorkspace/docs/golang-coding-standards.md`.
- Replace V1 outright; do not retain `--version`, semver discovery, a V1 selector, or a deprecated alias.
- `get` defaults to `https://github.com/cryptowizard0/vmdockerv2.git`, directory `./vmdockerv2`, and ref `main`.
- Do not build/download `vmdocker_agent`; consume `--agent-bin` or `VMDOCKER_AGENT_BIN`.
- Module build must execute `go run ./cmd/module`; do not import/copy VMDocker V2 build packages.
- Commands output IDs but never write module IDs or pids into `.env`.
- Do not add `respawn`, roundtrip orchestration, runtime presets, or node lifecycle commands.
- Do not modify Web UI product code. Its existing VMDocker Get incompatibility is accepted.
- Default tests must not require Docker, Redis, GitHub, or a live node.
- Preserve existing Claude/OpenClaw behavior.

## File map

- Modify `internal/vmdocker/{constants,get,runner,manager}.go`; delete V1-only `version.go`.
- Create `internal/vmdocker/{profile,module}.go` and focused tests.
- Modify `internal/cli/vmdocker.go`; create `vmdocker_{profile,module,runtime}.go` and tests.
- Modify `internal/cli/runtime_helpers.go`, `internal/cli/openclaw.go`, and `internal/cli/usage_strings.go` only where shared output/help requires it.
- Modify `internal/openclawui/catalog_integration_test.go` only to remove the obsolete `--version` catalog assertion; no UI production changes.
- Modify `README.md` for the V2 workflow.


---

### Task 1: Migrate `vmdocker get` to V2 refs

**Files:**
- Modify: `internal/vmdocker/constants.go`
- Modify: `internal/vmdocker/get.go`
- Delete: `internal/vmdocker/version.go`
- Modify: `internal/vmdocker/vmdocker_test.go`
- Modify: `internal/cli/vmdocker.go`
- Create: `internal/cli/vmdocker_test.go`
- Modify: `internal/cli/usage_strings.go`
- Modify: `internal/openclawui/catalog_integration_test.go`

**Interfaces:**
- Consumes: `CommandRunner.Output`, `fileExists`, and Manager filesystem hooks.
- Produces: `Manager.Get(ctx context.Context, dir, ref string) (resolvedRef, binaryPath string, err error)`, `DefaultRef`, and CLI flags `--dir`/`--ref`.

- [ ] **Step 1: Write failing V2 ref tests**

Replace semver-oriented get tests with:

```go
func TestGetDefaultsToMainAndBuildsV2(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "vmdockerv2")
	runner := &fakeRunner{run: func(dir string, env []string, name string, args ...string) (string, error) {
		switch strings.Join(append([]string{name}, args...), " ") {
		case "git remote get-url origin": return RepoURL, nil
		case "git rev-parse FETCH_HEAD": return "new-commit", nil
		case "git rev-parse HEAD": return "old-commit", nil
		case "git status --short --untracked-files=no": return "", nil
		default: return "", nil
		}
	}}
	manager := newTestManager(runner)
	ref, binaryPath, err := manager.Get(context.Background(), dir, "")
	if err != nil { t.Fatal(err) }
	if ref != "main" { t.Fatalf("ref = %q", ref) }
	if binaryPath != filepath.Join(dir, "build", "hymx-node") { t.Fatalf("binary = %q", binaryPath) }
	for _, want := range []string{
		"git clone https://github.com/cryptowizard0/vmdockerv2.git " + dir,
		"git fetch --force origin main",
		"git checkout --detach new-commit",
		"go mod tidy",
		"go build -o ./build/hymx-node ./cmd",
	} {
		if !containsCall(runner.calls, want) { t.Fatalf("missing %q in %#v", want, runner.calls) }
	}
}

func TestGetRefusesDirtyCheckoutBeforeSwitch(t *testing.T) {
	runner := &fakeRunner{run: func(dir string, env []string, name string, args ...string) (string, error) {
		switch strings.Join(append([]string{name}, args...), " ") {
		case "git remote get-url origin": return RepoURL, nil
		case "git rev-parse FETCH_HEAD": return "new-commit", nil
		case "git rev-parse HEAD": return "old-commit", nil
		case "git status --short --untracked-files=no": return " M go.mod", nil
		default: return "", nil
		}
	}}
	_, _, err := newTestManager(runner).Get(context.Background(), t.TempDir(), "feature/profile")
	if err == nil || !strings.Contains(err.Error(), "tracked changes") {
		t.Fatalf("expected tracked changes error, got %v", err)
	}
}
```

Create `internal/cli/vmdocker_test.go` with the CLI contract test:

```go
func TestVmdockerV2Defaults(t *testing.T) {
	root := NewRootCmd()
	get, _, err := root.Find([]string{"vmdocker", "get"})
	if err != nil { t.Fatal(err) }
	ref, _ := get.Flags().GetString("ref")
	dir, _ := get.Flags().GetString("dir")
	if ref != "main" || dir != "./vmdockerv2" { t.Fatalf("ref=%q dir=%q", ref, dir) }
	if get.Flags().Lookup("version") != nil { t.Fatal("--version must be removed") }
	initCmd, _, err := root.Find([]string{"vmdocker", "init"})
	if err != nil { t.Fatal(err) }
	initDir, _ := initCmd.Flags().GetString("dir")
	if initDir != "./vmdockerv2" { t.Fatalf("init dir = %q", initDir) }
}
```

Add `newTestManager(runner *fakeRunner) *Manager` using the injected functions already repeated in the test file. Keep init tests unchanged.

- [ ] **Step 2: Verify tests fail under V1 behavior**

Run:

```bash
go test ./internal/vmdocker -run 'TestGet(Default|Refuses)' -count=1
```

Expected: FAIL because current code validates/discovers semver and uses the V1 repository.

- [ ] **Step 3: Implement V2 constants and arbitrary-ref checkout**

Use:

```go
const (
	RepoURL         = "https://github.com/cryptowizard0/vmdockerv2.git"
	repoURLNoSuffix = "https://github.com/cryptowizard0/vmdockerv2"
	repoURLSSH      = "git@github.com:cryptowizard0/vmdockerv2.git"
	DefaultRef      = "main"
)
```

Rewrite get with these concrete operations:

```go
func (m *Manager) Get(ctx context.Context, dir, ref string) (string, string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil { return "", "", err }
	ref = strings.TrimSpace(ref)
	if ref == "" { ref = DefaultRef }
	info, err := m.stat(absDir)
	switch {
	case err == nil && !info.IsDir(): return "", "", fmt.Errorf("vmdocker dir is not a directory: %s", absDir)
	case os.IsNotExist(err):
		if err := m.mkdirAll(filepath.Dir(absDir), 0o755); err != nil { return "", "", err }
		if _, err := m.runner.Output(ctx, "", nil, "git", "clone", RepoURL, absDir); err != nil { return "", "", err }
	case err != nil: return "", "", err
	}
	binaryPath := filepath.Join(absDir, filepath.FromSlash(NodeBinary))
	shouldBuild, err := m.checkoutRef(ctx, absDir, ref, binaryPath)
	if err != nil { return "", "", err }
	if shouldBuild {
		if err := m.mkdirAll(filepath.Dir(binaryPath), 0o755); err != nil { return "", "", err }
		if _, err := m.runner.Output(ctx, absDir, nil, "go", "mod", "tidy"); err != nil { return "", "", err }
		if _, err := m.runner.Output(ctx, absDir, nil, "go", "build", "-o", "./build/hymx-node", "./cmd"); err != nil { return "", "", err }
	}
	return ref, binaryPath, nil
}

func (m *Manager) checkoutRef(ctx context.Context, dir, ref, binaryPath string) (bool, error) {
	remote, err := m.runner.Output(ctx, dir, nil, "git", "remote", "get-url", "origin")
	if err != nil { return false, fmt.Errorf("existing dir is not a vmdockerv2 git repository: %w", err) }
	if !isVmdockerRemote(remote) { return false, fmt.Errorf("existing dir is not a vmdockerv2 git repository: %s", dir) }
	if _, err := m.runner.Output(ctx, dir, nil, "git", "fetch", "--force", "origin", ref); err != nil { return false, err }
	target, err := m.runner.Output(ctx, dir, nil, "git", "rev-parse", "FETCH_HEAD")
	if err != nil { return false, err }
	current, err := m.runner.Output(ctx, dir, nil, "git", "rev-parse", "HEAD")
	if err != nil { return false, err }
	target = strings.TrimSpace(target)
	if target == strings.TrimSpace(current) { return !fileExists(m.stat, binaryPath), nil }
	status, err := m.runner.Output(ctx, dir, nil, "git", "status", "--short", "--untracked-files=no")
	if err != nil { return false, err }
	if strings.TrimSpace(status) != "" { return false, errors.New("vmdockerv2 checkout has tracked changes; refusing to switch ref") }
	if _, err := m.runner.Output(ctx, dir, nil, "git", "checkout", "--detach", target); err != nil { return false, err }
	return true, nil
}
```

Delete `version.go` and semver tests. In Cobra, replace `--version` with `--ref` default `vmdockerpkg.DefaultRef`, change both get/init directory defaults to `./vmdockerv2`, print `vmdocker ref: <ref>`, and define `usage_vmdocker_ref = "VMDocker Git branch, tag, or commit"`. Remove only the obsolete `vmdocker get` table case from `TestRunRouteBuildsExpectedCommands`.

- [ ] **Step 4: Format and test Task 1**

```bash
gofmt -w internal/vmdocker/constants.go internal/vmdocker/get.go internal/vmdocker/vmdocker_test.go internal/cli/vmdocker.go internal/cli/vmdocker_test.go internal/cli/usage_strings.go internal/openclawui/catalog_integration_test.go
go test ./internal/vmdocker ./internal/cli ./internal/openclawui -count=1
```

Expected: PASS; `git status --short` shows `version.go` deleted and only Task 1 files changed.

- [ ] **Step 5: Commit Task 1**

```bash
git add internal/vmdocker/constants.go internal/vmdocker/get.go internal/vmdocker/version.go internal/vmdocker/vmdocker_test.go internal/cli/vmdocker.go internal/cli/vmdocker_test.go internal/cli/usage_strings.go internal/openclawui/catalog_integration_test.go
git commit -m "feat: migrate vmdocker get to v2 refs"
```


---

### Task 2: Add the testagent-derived English profile scaffold

**Files:**
- Create: `internal/vmdocker/profile.go`
- Create: `internal/vmdocker/profile_test.go`
- Create: `internal/cli/vmdocker_profile.go`
- Create: `internal/cli/vmdocker_profile_test.go`
- Modify: `internal/cli/vmdocker.go`
- Modify: `internal/cli/usage_strings.go`

**Interfaces:**
- Consumes: Manager filesystem hooks.
- Produces: `Manager.InitProfile(dir, baseImage string) (absoluteDir string, err error)` and `hype vmdocker profile init --dir --from`.

- [ ] **Step 1: Write failing scaffold tests**

Create `internal/vmdocker/profile_test.go`:

```go
func TestInitProfileCreatesTestagentScaffold(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agent")
	absDir, err := NewManager().InitProfile(dir, "registry.example/base:1")
	if err != nil { t.Fatal(err) }
	if absDir != dir { t.Fatalf("absDir = %q, want %q", absDir, dir) }
	profile, err := os.ReadFile(filepath.Join(dir, "profile.toml"))
	if err != nil { t.Fatal(err) }
	for _, want := range []string{
		`FROM = "registry.example/base:1"`, `tools = []`, `RUN = []`,
		`# CMD = ["openclaw", "gateway", "--serve"]`,
		`public = ["~/skills/*", "~/persona/*", "~/.hermes/plugin/*"]`,
		`# Declarative recipe for a vmdockerv2 agent module.`,
	} {
		if !strings.Contains(string(profile), want) { t.Fatalf("profile missing %q:\n%s", want, profile) }
	}
	for _, tc := range []struct{ path, want string }{
		{filepath.Join(dir, "skills", "soul.md"), "MY-SOUL\n"},
		{filepath.Join(dir, "persona", "style.md"), "terse, precise\n"},
		{filepath.Join(dir, "bin", ".keep"), ""},
	} {
		got, err := os.ReadFile(tc.path)
		if err != nil || string(got) != tc.want { t.Fatalf("%s = %q, %v; want %q", tc.path, got, err, tc.want) }
	}
}

func TestInitProfileRefusesNonEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("keep"), 0o644); err != nil { t.Fatal(err) }
	_, err := NewManager().InitProfile(dir, "example/base:latest")
	if err == nil || !strings.Contains(err.Error(), "not empty") { t.Fatalf("expected non-empty error, got %v", err) }
}
```

Add these exact cases:

```go
func TestInitProfileRequiresFrom(t *testing.T) {
	_, err := NewManager().InitProfile(filepath.Join(t.TempDir(), "agent"), "  ")
	if err == nil || err.Error() != "from is required" { t.Fatalf("err = %v", err) }
}

func TestInitProfileRejectsFileTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent")
	if err := os.WriteFile(path, []byte("occupied"), 0o644); err != nil { t.Fatal(err) }
	_, err := NewManager().InitProfile(path, "example/base:latest")
	if err == nil || !strings.Contains(err.Error(), "not a directory") { t.Fatalf("err = %v", err) }
}
```

- [ ] **Step 2: Verify the scaffold tests fail**

```bash
go test ./internal/vmdocker -run TestInitProfile -count=1
```

Expected: FAIL because `InitProfile` does not exist.

- [ ] **Step 3: Implement the exact embedded template and files**

Create `internal/vmdocker/profile.go` with this template constant and method:

```go
const profileTemplate = `# Declarative recipe for a vmdockerv2 agent module.
#   [dockerfile] -> input to the standardized Dockerfile generator
#   [vmdocker]   -> public allowlist used by runtime Export/Import
# The two sections are independent.

[dockerfile]
# Full base image name, used verbatim as Dockerfile FROM (no alias mapping).
# RUNTIME_TYPE does not belong here. It is passed at spawn time through the
# Container-Env-RUNTIME_TYPE tag and controls the adapter readiness check.
FROM = %s

# Directory containing user executables. The whole directory is copied to
# /usr/local/bin and made executable. Required; it may be empty when kept by .keep.
bin = "bin"

# Optional startup command using Dockerfile CMD syntax. The adapter remains the
# ENTRYPOINT and runs this command. Arrays use exec form; strings use shell form.
# No-op modules can omit CMD.
# CMD = ["your-engine", "--serve"]

# Optional cross-distribution tool packages installed during the image build.
tools = []

# Optional Dockerfile RUN bodies. Values do not include the leading "RUN ".
RUN = []

# CMD = ["openclaw", "gateway", "--serve"]

[vmdocker]
# Export allowlist relative to HOME. Export preserves these paths, and spawn
# overlays them into a fresh workspace.
#   "~/directory/*" selects a directory recursively; "~/file" selects one file.
# Everything under HOME that is not listed remains private and is never exported.
public = ["~/skills/*", "~/persona/*", "~/.hermes/plugin/*"]
`

func (m *Manager) InitProfile(dir, baseImage string) (string, error) {
	baseImage = strings.TrimSpace(baseImage)
	if baseImage == "" { return "", errors.New("from is required") }
	absDir, err := filepath.Abs(dir)
	if err != nil { return "", err }
	info, err := m.stat(absDir)
	switch {
	case err == nil && !info.IsDir():
		return "", fmt.Errorf("profile target is not a directory: %s", absDir)
	case err == nil:
		entries, err := m.readDir(absDir)
		if err != nil { return "", err }
		if len(entries) != 0 { return "", fmt.Errorf("profile target directory is not empty: %s", absDir) }
	case !os.IsNotExist(err):
		return "", err
	}
	files := map[string]string{
		"profile.toml": fmt.Sprintf(profileTemplate, strconv.Quote(baseImage)),
		"bin/.keep": "", "skills/soul.md": "MY-SOUL\n", "persona/style.md": "terse, precise\n",
	}
	for rel, content := range files {
		path := filepath.Join(absDir, filepath.FromSlash(rel))
		if err := m.mkdirAll(filepath.Dir(path), 0o755); err != nil { return "", err }
		if err := m.writeFile(path, []byte(content), 0o644); err != nil { return "", err }
	}
	return absDir, nil
}
```

Use `strconv.Quote` exactly so the base image stays valid TOML.

- [ ] **Step 4: Add and test the Cobra command**

Create a `profile` parent and `init` child in `internal/cli/vmdocker_profile.go`:

```go
func newVmdockerProfileCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "profile", Short: "Manage VMDocker profile scaffolds"}
	cmd.AddCommand(newVmdockerProfileInitCmd())
	return cmd
}

func newVmdockerProfileInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "init", Short: "Create a VMDocker profile scaffold",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return replPromptRequiredStringFlags(cmd, []replFlagPrompt{
				{Name: "dir", Prompt: "dir (--dir) " + usage_vmdocker_profile_dir + ": "},
				{Name: "from", Prompt: "from (--from) " + usage_vmdocker_profile_from + ": "},
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("dir")
			from, _ := cmd.Flags().GetString("from")
			absDir, err := vmdockerpkg.NewManager().InitProfile(dir, from)
			if err != nil { return err }
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "profile directory: %s\n", absDir)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "profile file: %s\n", filepath.Join(absDir, "profile.toml"))
			return nil
		},
	}
	cmd.Flags().String("dir", "", usage_vmdocker_profile_dir)
	cmd.Flags().String("from", "", usage_vmdocker_profile_from)
	_ = cmd.MarkFlagRequired("dir")
	_ = cmd.MarkFlagRequired("from")
	return cmd
}
```

Use these result lines:

```go
_, _ = fmt.Fprintf(cmd.OutOrStdout(), "profile directory: %s\n", absDir)
_, _ = fmt.Fprintf(cmd.OutOrStdout(), "profile file: %s\n", filepath.Join(absDir, "profile.toml"))
```

Wire `newVmdockerProfileCmd()` into `newVmdockerCmd()`. Define `usage_vmdocker_profile_dir = "Target agent profile directory"` and `usage_vmdocker_profile_from = "Full base image name"`. In `vmdocker_profile_test.go`, execute `NewRootCmd()` against a temp path and assert all four files plus both output lines.

Run:

```bash
gofmt -w internal/vmdocker/profile.go internal/vmdocker/profile_test.go internal/cli/vmdocker_profile.go internal/cli/vmdocker_profile_test.go internal/cli/vmdocker.go internal/cli/usage_strings.go
go test ./internal/vmdocker ./internal/cli -run 'TestInitProfile|TestVmdockerProfile' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit Task 2**

```bash
git add internal/vmdocker/profile.go internal/vmdocker/profile_test.go internal/cli/vmdocker_profile.go internal/cli/vmdocker_profile_test.go internal/cli/vmdocker.go internal/cli/usage_strings.go
git commit -m "feat: scaffold vmdocker v2 profiles"
```


---

### Task 3: Delegate module builds with streaming output

**Files:**
- Modify: `internal/vmdocker/runner.go`
- Modify: `internal/vmdocker/manager.go`
- Modify: `internal/vmdocker/vmdocker_test.go`
- Create: `internal/vmdocker/module.go`
- Create: `internal/vmdocker/module_test.go`
- Create: `internal/cli/vmdocker_module.go`
- Modify: `internal/cli/vmdocker.go`
- Modify: `internal/cli/usage_strings.go`

**Interfaces:**
- Consumes: `ParseEnvFile`, Manager hooks, and `CommandRunner`.
- Produces: `CommandRunner.Run`, `Manager.SetErrorOutput`, `ModuleBuildOptions`, `Manager.BuildModule`, and `hype vmdocker module build`.

- [ ] **Step 1: Write failing delegated-build tests**

Extend fakeRunner with `runStream`, then create `module_test.go`:

```go
func TestBuildModuleResolvesInputsAndStreams(t *testing.T) {
	checkout := t.TempDir()
	profile := filepath.Join(t.TempDir(), "profile.toml")
	agentBin := filepath.Join(t.TempDir(), "vmdocker-agent")
	if err := os.WriteFile(profile, []byte("[dockerfile]\n"), 0o644); err != nil { t.Fatal(err) }
	if err := os.WriteFile(agentBin, []byte("bin"), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(checkout, ".env"), []byte("VMDOCKER_URL=http://file\nVMDOCKER_PRIVATE_KEY=file-key\n"), 0o600); err != nil { t.Fatal(err) }
	var gotDir string
	var gotEnv, gotArgs []string
	runner := &fakeRunner{runStream: func(dir string, env []string, stdout, stderr io.Writer, name string, args ...string) error {
		gotDir, gotEnv, gotArgs = dir, append([]string(nil), env...), append([]string{name}, args...)
		_, _ = io.WriteString(stdout, "building\n")
		_, _ = io.WriteString(stderr, "progress\n")
		return nil
	}}
	manager := newTestManager(runner)
	var stdout, stderr bytes.Buffer
	manager.SetOutput(&stdout)
	manager.SetErrorOutput(&stderr)
	err := manager.BuildModule(context.Background(), ModuleBuildOptions{
		CheckoutDir: checkout, ProfilePath: profile, AgentBinPath: agentBin,
		NodeURL: "http://flag", PrivateKey: "flag-key",
	})
	if err != nil { t.Fatal(err) }
	if gotDir != checkout { t.Fatalf("dir = %q", gotDir) }
	wantArgs := []string{"go", "run", "./cmd/module", "--profile", profile, "--agent-bin", agentBin}
	if !slices.Equal(gotArgs, wantArgs) { t.Fatalf("args = %#v", gotArgs) }
	if !hasEnv(gotEnv, "VMDOCKER_URL=http://flag") || !hasEnv(gotEnv, "VMDOCKER_PRIVATE_KEY=flag-key") { t.Fatalf("env = %#v", gotEnv) }
	if stdout.String() != "building\n" || stderr.String() != "progress\n" { t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String()) }
}
```

Add a precedence test that writes all three values into `.env`, sets different process values with `t.Setenv`, passes different option values, and asserts the option values reach `runner.Run`. Add these validation rows:

```go
tests := []struct {
	name string
	opts ModuleBuildOptions
	wantErr string
}{
	{name: "missing profile", opts: ModuleBuildOptions{CheckoutDir: checkout, AgentBinPath: agentBin, PrivateKey: "key"}, wantErr: "profile is required"},
	{name: "missing adapter", opts: ModuleBuildOptions{CheckoutDir: checkout, ProfilePath: profile, PrivateKey: "key"}, wantErr: "agent-bin is required"},
	{name: "missing private key", opts: ModuleBuildOptions{CheckoutDir: checkout, ProfilePath: profile, AgentBinPath: agentBin}, wantErr: "private-key is required"},
}
```

For runner failure, set `runStream` to `return errors.New("build failed")` and assert `BuildModule` returns that text without writing a duplicate log line.

- [ ] **Step 2: Verify missing interfaces fail**

```bash
go test ./internal/vmdocker -run TestBuildModule -count=1
```

Expected: FAIL because streaming/build interfaces do not exist.

- [ ] **Step 3: Add streaming execution**

Extend runner.go:

```go
type CommandRunner interface {
	Output(ctx context.Context, dir string, env []string, name string, args ...string) (string, error)
	Run(ctx context.Context, dir string, env []string, stdout, stderr io.Writer, name string, args ...string) error
}

func (ExecCommandRunner) Run(ctx context.Context, dir string, env []string, stdout, stderr io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr = dir, append(os.Environ(), env...), stdout, stderr
	if err := cmd.Run(); err != nil { return fmt.Errorf("%s %v failed: %w", name, args, err) }
	return nil
}
```

Add `errOut io.Writer` to Manager and:

```go
func (m *Manager) SetErrorOutput(out io.Writer) { m.errOut = out }
```

Make fakeRunner.Run record calls and invoke runStream when non-nil.

- [ ] **Step 4: Implement resolution and delegation**

Create module.go:

```go
type ModuleBuildOptions struct {
	CheckoutDir, ProfilePath, AgentBinPath, NodeURL, PrivateKey string
}

func (m *Manager) BuildModule(ctx context.Context, opts ModuleBuildOptions) error {
	checkout, err := filepath.Abs(opts.CheckoutDir)
	if err != nil { return err }
	if info, err := m.stat(checkout); err != nil || !info.IsDir() {
		return fmt.Errorf("vmdockerv2 checkout not found: %s", checkout)
	}
	envValues, err := m.readBuildEnv(checkout)
	if err != nil { return err }
	agentBin := firstValue(opts.AgentBinPath, os.Getenv("VMDOCKER_AGENT_BIN"), envValues["VMDOCKER_AGENT_BIN"])
	nodeURL := firstValue(opts.NodeURL, os.Getenv("VMDOCKER_URL"), envValues["VMDOCKER_URL"], "http://127.0.0.1:8080")
	privateKey := firstValue(opts.PrivateKey, os.Getenv("VMDOCKER_PRIVATE_KEY"), os.Getenv("HYPE_PRIVATE_KEY"), os.Getenv("PRV_KEY"), envValues["VMDOCKER_PRIVATE_KEY"])
	if privateKey == "" { return errors.New("private-key is required") }
	profile, err := absoluteRegularFile(opts.ProfilePath, m.stat, "profile")
	if err != nil { return err }
	agentBin, err = absoluteRegularFile(agentBin, m.stat, "agent-bin")
	if err != nil { return err }
	stdout, stderr := m.out, m.errOut
	if stdout == nil { stdout = io.Discard }
	if stderr == nil { stderr = io.Discard }
	return m.runner.Run(ctx, checkout, []string{
		"VMDOCKER_URL=" + nodeURL,
		"VMDOCKER_PRIVATE_KEY=" + privateKey,
	}, stdout, stderr, "go", "run", "./cmd/module", "--profile", profile, "--agent-bin", agentBin)
}
```

Implement the helpers exactly:

```go
func (m *Manager) readBuildEnv(checkout string) (map[string]string, error) {
	path := strings.TrimSpace(os.Getenv("VMDOCKER_ENV_FILE"))
	if path == "" {
		path = filepath.Join(checkout, ".env")
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(checkout, path)
	}
	values, err := ParseEnvFile(path, m.readFile)
	if os.IsNotExist(err) { return map[string]string{}, nil }
	return values, err
}

func firstValue(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" { return value }
	}
	return ""
}

func absoluteRegularFile(path string, stat func(string) (os.FileInfo, error), label string) (string, error) {
	if strings.TrimSpace(path) == "" { return "", fmt.Errorf("%s is required", label) }
	absPath, err := filepath.Abs(path)
	if err != nil { return "", err }
	info, err := stat(absPath)
	if err != nil { return "", fmt.Errorf("%s not found: %s: %w", label, absPath, err) }
	if !info.Mode().IsRegular() { return "", fmt.Errorf("%s is not a regular file: %s", label, absPath) }
	return absPath, nil
}
```

- [ ] **Step 5: Add the Cobra command**

Create `newVmdockerModuleCmd()` as a parent that adds `newVmdockerModuleBuildCmd()`. The build child defines `--dir ./vmdockerv2`, required `--profile`, optional `--agent-bin`, `--node-url`, and `--private-key`; its RunE sets `manager.SetOutput(cmd.OutOrStdout())` and `manager.SetErrorOutput(cmd.ErrOrStderr())`, then calls:

```go
return manager.BuildModule(commandContext(cmd), vmdockerpkg.ModuleBuildOptions{
	CheckoutDir: dir, ProfilePath: profile, AgentBinPath: agentBin,
	NodeURL: nodeURL, PrivateKey: privateKey,
})
```

Wire `newVmdockerModuleCmd()`. Define help strings `Target VMDocker V2 checkout`, `Path to profile.toml`, `Path to vmdocker-agent binary`, `Node URL`, and `Module signing private key`. Do not parse output or persist the ID.

- [ ] **Step 6: Format and test Task 3**

```bash
gofmt -w internal/vmdocker/runner.go internal/vmdocker/manager.go internal/vmdocker/vmdocker_test.go internal/vmdocker/module.go internal/vmdocker/module_test.go internal/cli/vmdocker_module.go internal/cli/vmdocker.go internal/cli/usage_strings.go
go test ./internal/vmdocker ./internal/cli -count=1
```

Expected: PASS, including init orchestration tests.

- [ ] **Step 7: Commit Task 3**

```bash
git add internal/vmdocker/runner.go internal/vmdocker/manager.go internal/vmdocker/vmdocker_test.go internal/vmdocker/module.go internal/vmdocker/module_test.go internal/cli/vmdocker_module.go internal/cli/vmdocker.go internal/cli/usage_strings.go
git commit -m "feat: build vmdocker modules from profiles"
```


---

### Task 4: Add generic VMDocker spawn

**Files:**
- Create: `internal/cli/vmdocker_runtime.go`
- Create: `internal/cli/vmdocker_runtime_test.go`
- Modify: `internal/cli/vmdocker.go`
- Modify: `internal/cli/runtime_helpers.go`
- Modify: `internal/cli/openclaw.go`
- Modify: `internal/cli/usage_strings.go`

**Interfaces:**
- Consumes: `newSDK`, `validateRuntimeBackend`, `firstNonEmptyEnv`, `maxIDChars`, and Hymx `serverSchema.Response`.
- Produces: `vmdockerRuntimeClient`, `vmdockerRuntimeClientFactory`, `buildVmdockerSpawnTags`, `writeRuntimeResult`, and `hype vmdocker spawn`.

- [ ] **Step 1: Write failing tag and command tests**

Create vmdocker_runtime_test.go with this reusable fake:

```go
type fakeVmdockerRuntimeClient struct {
	spawnModule, spawnScheduler string
	spawnTags []goarSchema.Tag
	spawnResponse *serverSchema.Response
	spawnErr error
	messageTarget string
	messageTags []goarSchema.Tag
	messageResponse *serverSchema.Response
	messageErr error
}

func (f *fakeVmdockerRuntimeClient) SpawnAndWait(module, scheduler string, tags []goarSchema.Tag) (*serverSchema.Response, error) {
	f.spawnModule, f.spawnScheduler = module, scheduler
	f.spawnTags = append([]goarSchema.Tag(nil), tags...)
	return f.spawnResponse, f.spawnErr
}
func (f *fakeVmdockerRuntimeClient) SendMessageAndWait(target, data string, tags []goarSchema.Tag) (*serverSchema.Response, error) {
	f.messageTarget = target
	f.messageTags = append([]goarSchema.Tag(nil), tags...)
	return f.messageResponse, f.messageErr
}
func (f *fakeVmdockerRuntimeClient) Close() {}
```

Test exact mappings:

```go
func TestBuildVmdockerSpawnTags(t *testing.T) {
	tags, err := buildVmdockerSpawnTags("claude", "docker", []string{"TOKEN=a=b", "EMPTY="})
	if err != nil { t.Fatal(err) }
	for _, want := range []goarSchema.Tag{
		{Name: "Container-Env-RUNTIME_TYPE", Value: "claude"},
		{Name: "Container-Env-TOKEN", Value: "a=b"},
		{Name: "Container-Env-EMPTY", Value: ""},
		{Name: "Runtime-Backend", Value: "docker"},
	} {
		if !hasTag(tags, want.Name, want.Value) { t.Fatalf("missing %#v in %#v", want, tags) }
	}
}
```

Add these exact invalid rows:

```go
tests := []struct{ name, backend string; env []string; wantErr string }{
	{name: "bad key", env: []string{"BAD-KEY=x"}, wantErr: "invalid environment key"},
	{name: "duplicate", env: []string{"TOKEN=a", "TOKEN=b"}, wantErr: "duplicate environment key"},
	{name: "reserved", env: []string{"RUNTIME_TYPE=claude"}, wantErr: "RUNTIME_TYPE is reserved"},
	{name: "bad backend", backend: "podman", wantErr: "runtime-backend must be one of"},
}
```

Add this command test:

```go
func TestVmdockerSpawnCommand(t *testing.T) {
	fake := &fakeVmdockerRuntimeClient{spawnResponse: &serverSchema.Response{Id: "pid-1", Message: "ok"}}
	cmd := newVmdockerSpawnCmdWithClient(func(nodeURL, privateKey string) (vmdockerRuntimeClient, error) {
		return fake, nil
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--module-id", "mod-1", "--scheduler", "scheduler-1", "--private-key", "key", "--runtime-type", "claude", "--env", "TOKEN=a=b"})
	if err := cmd.Execute(); err != nil { t.Fatal(err) }
	if fake.spawnModule != "mod-1" || fake.spawnScheduler != "scheduler-1" { t.Fatalf("module=%q scheduler=%q", fake.spawnModule, fake.spawnScheduler) }
	if !hasTag(fake.spawnTags, "Container-Env-TOKEN", "a=b") { t.Fatalf("tags = %#v", fake.spawnTags) }
	if !strings.Contains(out.String(), "spawn ok, pid: pid-1") { t.Fatalf("output = %q", out.String()) }
}
```

Add one test covering environment hydration, JSON, and secret-safe output:

```go
func TestVmdockerSpawnJSONHydratesEnvWithoutEchoingValues(t *testing.T) {
	t.Setenv("VMDOCKER_MODULE_ID", "mod-env")
	t.Setenv("VMDOCKER_SCHEDULER", "scheduler-env")
	t.Setenv("RUNTIME_TYPE", "claude")
	t.Setenv("RUNTIME_BACKEND", "sandbox")
	t.Setenv("VMDOCKER_URL", "http://node-env")
	t.Setenv("VMDOCKER_PRIVATE_KEY", "key-env")
	fake := &fakeVmdockerRuntimeClient{spawnResponse: &serverSchema.Response{Id: "pid-1"}}
	var gotNodeURL, gotPrivateKey string
	cmd := newVmdockerSpawnCmdWithClient(func(nodeURL, privateKey string) (vmdockerRuntimeClient, error) {
		gotNodeURL, gotPrivateKey = nodeURL, privateKey
		return fake, nil
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--json", "--env", "SECRET=do-not-print"})
	if err := cmd.Execute(); err != nil { t.Fatal(err) }
	if gotNodeURL != "http://node-env" || gotPrivateKey != "key-env" { t.Fatalf("node=%q key=%q", gotNodeURL, gotPrivateKey) }
	if fake.spawnModule != "mod-env" || fake.spawnScheduler != "scheduler-env" { t.Fatalf("module=%q scheduler=%q", fake.spawnModule, fake.spawnScheduler) }
	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil { t.Fatal(err) }
	if payload["action"] != "spawn" || payload["pid"] != "pid-1" { t.Fatalf("payload = %#v", payload) }
	if strings.Contains(out.String(), "do-not-print") { t.Fatalf("secret leaked: %s", out.String()) }
}
```

- [ ] **Step 2: Verify the spawn tests fail**

```bash
go test ./internal/cli -run 'TestBuildVmdockerSpawnTags|TestVmdockerSpawn' -count=1
```

Expected: FAIL because runtime boundary, tag builder, and command do not exist.

- [ ] **Step 3: Extract writer-aware result formatting**

Add to runtime_helpers.go:

```go
func writeRuntimeResult(out io.Writer, jsonOut bool, payload map[string]interface{}, fallback string) error {
	if !jsonOut {
		_, err := fmt.Fprintln(out, fallback)
		return err
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil { return err }
	_, err = fmt.Fprintln(out, string(b))
	return err
}
```

Keep the current openclaw helper signature, but delegate:

```go
func printOpenclawResult(jsonOut bool, payload map[string]interface{}, fallback string) error {
	return writeRuntimeResult(os.Stdout, jsonOut, payload, fallback)
}
```

- [ ] **Step 4: Implement generic tags and SDK boundary**

Define:

```go
type vmdockerRuntimeClient interface {
	SpawnAndWait(module, scheduler string, tags []goarSchema.Tag) (*serverSchema.Response, error)
	SendMessageAndWait(target, data string, tags []goarSchema.Tag) (*serverSchema.Response, error)
	Close()
}
type vmdockerRuntimeClientFactory func(nodeURL, privateKey string) (vmdockerRuntimeClient, error)

func newVmdockerRuntimeClient(nodeURL, privateKey string) (vmdockerRuntimeClient, error) {
	return newSDK(nodeURL, privateKey)
}
```

Implement the tag builder exactly:

```go
var vmdockerEnvKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func buildVmdockerSpawnTags(runtimeType, backend string, assignments []string) ([]goarSchema.Tag, error) {
	if err := validateRuntimeBackend(backend); err != nil { return nil, err }
	tags := make([]goarSchema.Tag, 0, len(assignments)+2)
	if runtimeType = strings.TrimSpace(runtimeType); runtimeType != "" {
		tags = append(tags, goarSchema.Tag{Name: containerEnvTagPrefix + "RUNTIME_TYPE", Value: runtimeType})
	}
	seen := make(map[string]struct{}, len(assignments))
	for _, assignment := range assignments {
		key, value, ok := strings.Cut(assignment, "=")
		key = strings.TrimSpace(key)
		if !ok || !vmdockerEnvKeyPattern.MatchString(key) { return nil, fmt.Errorf("invalid environment key in %q", assignment) }
		if key == "RUNTIME_TYPE" { return nil, errors.New("RUNTIME_TYPE is reserved; use --runtime-type") }
		if _, exists := seen[key]; exists { return nil, fmt.Errorf("duplicate environment key: %s", key) }
		seen[key] = struct{}{}
		tags = append(tags, goarSchema.Tag{Name: containerEnvTagPrefix + key, Value: value})
	}
	if backend = strings.TrimSpace(backend); backend != "" {
		tags = append(tags, goarSchema.Tag{Name: "Runtime-Backend", Value: backend})
	}
	return tags, nil
}
```

- [ ] **Step 5: Implement and wire spawn**

Provide:

```go
func newVmdockerSpawnCmd() *cobra.Command {
	return newVmdockerSpawnCmdWithClient(newVmdockerRuntimeClient)
}
```

Define `vmdockerSpawnOptions` and keep execution separate from Cobra:

```go
type vmdockerSpawnOptions struct {
	nodeURL, privateKey, moduleID, scheduler, runtimeType, runtimeBackend string
	env []string
	jsonOut bool
}

func runVmdockerSpawn(factory vmdockerRuntimeClientFactory, out io.Writer, opts vmdockerSpawnOptions) error {
	if strings.TrimSpace(opts.moduleID) == "" { return errors.New("module-id is required") }
	if len(opts.moduleID) > maxIDChars { return fmt.Errorf("module-id is too long (max %d)", maxIDChars) }
	if strings.TrimSpace(opts.scheduler) == "" { return errors.New("scheduler is required") }
	if len(opts.scheduler) > maxIDChars { return fmt.Errorf("scheduler is too long (max %d)", maxIDChars) }
	if strings.TrimSpace(opts.privateKey) == "" { return errors.New("private-key is required") }
	tags, err := buildVmdockerSpawnTags(opts.runtimeType, opts.runtimeBackend, opts.env)
	if err != nil { return err }
	client, err := factory(opts.nodeURL, opts.privateKey)
	if err != nil { return err }
	defer client.Close()
	res, err := client.SpawnAndWait(opts.moduleID, opts.scheduler, tags)
	if err != nil { return err }
	payload := map[string]interface{}{"action": "spawn", "pid": res.Id, "response_id": res.Id, "message": res.Message}
	return writeRuntimeResult(out, opts.jsonOut, payload, fmt.Sprintf("spawn ok, pid: %s", res.Id))
}
```

The injectable Cobra constructor hydrates module ID, scheduler, runtime type, and backend from `VMDOCKER_MODULE_ID`, `VMDOCKER_SCHEDULER`, `RUNTIME_TYPE`, and `RUNTIME_BACKEND`; resolves node URL as flag > `VMDOCKER_URL` > `http://127.0.0.1:8080`; resolves key as flag > `VMDOCKER_PRIVATE_KEY` > `HYPE_PRIVATE_KEY` > `PRV_KEY`; then calls `runVmdockerSpawn(factory, cmd.OutOrStdout(), opts)`.

The success payload is therefore:

```go
payload := map[string]interface{}{
	"action": "spawn", "pid": res.Id, "response_id": res.Id, "message": res.Message,
}
return writeRuntimeResult(cmd.OutOrStdout(), jsonOut, payload, fmt.Sprintf("spawn ok, pid: %s", res.Id))
```

Flags: `--module-id/-m`, `--scheduler/-s`, `--runtime-type`, `--runtime-backend`, `StringArray("env", nil, ...)`, `--node-url/-u`, `--private-key/-k`, `--json`. Wire into newVmdockerCmd.

- [ ] **Step 6: Format and run CLI regressions**

```bash
gofmt -w internal/cli/vmdocker_runtime.go internal/cli/vmdocker_runtime_test.go internal/cli/vmdocker.go internal/cli/runtime_helpers.go internal/cli/openclaw.go internal/cli/usage_strings.go
go test ./internal/cli -count=1
```

Expected: PASS, including Claude/OpenClaw tests.

- [ ] **Step 7: Commit Task 4**

```bash
git add internal/cli/vmdocker_runtime.go internal/cli/vmdocker_runtime_test.go internal/cli/vmdocker.go internal/cli/runtime_helpers.go internal/cli/openclaw.go internal/cli/usage_strings.go
git commit -m "feat: add generic vmdocker spawn"
```


---

### Task 5: Add VMDocker export with strict result decoding

**Files:**
- Modify: `internal/cli/vmdocker_runtime.go`
- Modify: `internal/cli/vmdocker_runtime_test.go`
- Modify: `internal/cli/vmdocker.go`
- Modify: `internal/cli/usage_strings.go`

**Interfaces:**
- Consumes: Task 4 runtime client/factory, output helper, and Hymx `vmmSchema.VmmResult`.
- Produces: `decodeVmdockerExportResult(message string) (moduleID string, err error)` and `hype vmdocker export`.

- [ ] **Step 1: Write failing decoder and command tests**

Add:

```go
func TestDecodeVmdockerExportResult(t *testing.T) {
	tests := []struct{ name, message, want, wantErr string }{
		{"success", `{"Data":"mod-2","Error":""}`, "mod-2", ""},
		{"vmm error", `{"Data":"","Error":"export failed"}`, "", "export failed"},
		{"malformed", `{`, "", "decode export result"},
		{"empty id", `{"Data":"   ","Error":""}`, "", "empty module id"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeVmdockerExportResult(tc.message)
			if got != tc.want { t.Fatalf("got %q, want %q", got, tc.want) }
			if tc.wantErr == "" && err != nil { t.Fatal(err) }
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) { t.Fatalf("err = %v", err) }
		})
	}
}
```

Add this command success test:

```go
func TestVmdockerExportCommand(t *testing.T) {
	fake := &fakeVmdockerRuntimeClient{messageResponse: &serverSchema.Response{
		Id: "response-1", Message: `{"Data":"mod-2","Error":""}`,
	}}
	cmd := newVmdockerExportCmdWithClient(func(nodeURL, privateKey string) (vmdockerRuntimeClient, error) {
		return fake, nil
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--pid", "pid-1", "--private-key", "key"})
	if err := cmd.Execute(); err != nil { t.Fatal(err) }
	if fake.messageTarget != "pid-1" { t.Fatalf("target = %q", fake.messageTarget) }
	if len(fake.messageTags) != 1 || fake.messageTags[0] != (goarSchema.Tag{Name: "Action", Value: "Export"}) {
		t.Fatalf("tags = %#v", fake.messageTags)
	}
	if !strings.Contains(out.String(), "export ok, module id: mod-2") { t.Fatalf("output = %q", out.String()) }
}
```

Add the failure-output assertion:

```go
func TestVmdockerExportCommandDoesNotPrintSuccessOnNodeError(t *testing.T) {
	fake := &fakeVmdockerRuntimeClient{messageResponse: &serverSchema.Response{Message: `{"Error":"export failed"}`}}
	cmd := newVmdockerExportCmdWithClient(func(nodeURL, privateKey string) (vmdockerRuntimeClient, error) { return fake, nil })
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--pid", "pid-1", "--private-key", "key"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "export failed") { t.Fatalf("err = %v", err) }
	if strings.Contains(out.String(), "export ok") { t.Fatalf("unexpected success output: %q", out.String()) }
}
```

Add a JSON assertion using the same fake:

```go
func TestVmdockerExportJSON(t *testing.T) {
	fake := &fakeVmdockerRuntimeClient{messageResponse: &serverSchema.Response{Id: "response-1", Message: `{"Data":"mod-2"}`}}
	cmd := newVmdockerExportCmdWithClient(func(nodeURL, privateKey string) (vmdockerRuntimeClient, error) { return fake, nil })
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--pid", "pid-1", "--private-key", "do-not-print", "--json"})
	if err := cmd.Execute(); err != nil { t.Fatal(err) }
	var payload map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil { t.Fatal(err) }
	if payload["module_id"] != "mod-2" { t.Fatalf("payload = %#v", payload) }
	if strings.Contains(out.String(), "do-not-print") { t.Fatalf("private key leaked: %s", out.String()) }
}
```

- [ ] **Step 2: Verify export tests fail**

```bash
go test ./internal/cli -run 'TestDecodeVmdockerExportResult|TestVmdockerExport' -count=1
```

Expected: FAIL because decoder/export command do not exist.

- [ ] **Step 3: Implement strict decode**

```go
func decodeVmdockerExportResult(message string) (string, error) {
	var result vmmSchema.VmmResult
	if err := json.Unmarshal([]byte(message), &result); err != nil {
		return "", fmt.Errorf("decode export result: %w", err)
	}
	if strings.TrimSpace(result.Error) != "" { return "", fmt.Errorf("export failed on node: %s", result.Error) }
	moduleID := strings.TrimSpace(result.Data)
	if moduleID == "" { return "", errors.New("export returned empty module id") }
	return moduleID, nil
}
```

- [ ] **Step 4: Implement and wire export**

Add `newVmdockerExportCmd()` / `newVmdockerExportCmdWithClient(factory)`. Define `--pid/-p`, `--node-url/-u`, `--private-key/-k`, and `--json`. Require pid/key; pid falls back to `VMDOCKER_EXPORT_PID`; node/key precedence matches spawn. The RunE core is:

```go
res, err := client.SendMessageAndWait(pid, "", []goarSchema.Tag{{Name: "Action", Value: "Export"}})
if err != nil { return err }
moduleID, err := decodeVmdockerExportResult(res.Message)
if err != nil { return err }
```

After decode, output:

```go
payload := map[string]interface{}{
	"action": "export", "pid": pid, "module_id": moduleID,
	"response_id": res.Id, "message": res.Message,
}
return writeRuntimeResult(cmd.OutOrStdout(), jsonOut, payload, fmt.Sprintf("export ok, module id: %s", moduleID))
```

Do not add dry-run, respawn, or env-file writes. Wire export into newVmdockerCmd.

- [ ] **Step 5: Format and test Task 5**

```bash
gofmt -w internal/cli/vmdocker_runtime.go internal/cli/vmdocker_runtime_test.go internal/cli/vmdocker.go internal/cli/usage_strings.go
go test ./internal/cli -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Task 5**

```bash
git add internal/cli/vmdocker_runtime.go internal/cli/vmdocker_runtime_test.go internal/cli/vmdocker.go internal/cli/usage_strings.go
git commit -m "feat: export vmdocker processes as modules"
```


---

### Task 6: Document and verify the complete workflow

**Files:**
- Modify: `README.md`

**Interfaces:**
- Consumes: commands from Tasks 1-5.
- Produces: end-user command reference and final repository validation.

- [ ] **Step 1: Replace the V1 README reference with V2 commands**

Document these forms:

```text
hype vmdocker get [--dir ./vmdockerv2] [--ref main]
hype vmdocker init [--dir ./vmdockerv2] --env-file <path>
hype vmdocker profile init --dir <agent-dir> --from <base-image>
hype vmdocker module build --dir ./vmdockerv2 --profile <profile.toml> --agent-bin <vmdocker-agent> --private-key <key>
hype vmdocker spawn --module-id <id> --scheduler <address> --runtime-type <type> [--runtime-backend docker|sandbox] [--env KEY=VALUE]
hype vmdocker export --pid <pid> --private-key <key>
```

Document flag/environment precedence, the testagent-derived scaffold, module delegation, exact generic tag mappings, export returning a reusable module ID, and no automatic env-file writes. Add an explicit note that the embedded Web UI's VMDocker Get action still uses removed `--version` and is unsupported by this CLI-only migration.

- [ ] **Step 2: Verify CLI help and removed commands**

```bash
go run ./cmd/hype vmdocker --help
go run ./cmd/hype vmdocker get --help
go run ./cmd/hype vmdocker profile init --help
go run ./cmd/hype vmdocker module build --help
go run ./cmd/hype vmdocker spawn --help
go run ./cmd/hype vmdocker export --help
```

Expected: tree contains get/init/profile/module/spawn/export; get shows `--ref main` and `--dir ./vmdockerv2`; no output contains `--version` or `respawn`.

- [ ] **Step 3: Run final formatting, checks, tests, and build**

```bash
gofmt -w internal/cli internal/vmdocker
git diff --check
go test ./...
go build -o ./build/hype ./cmd/hype
```

Expected: every command exits 0; tests use no external infrastructure; build produces `build/hype`.

- [ ] **Step 4: Review scope and dependency boundaries**

```bash
git status --short
git diff --stat HEAD~5
git diff HEAD~5 -- internal/cli internal/vmdocker README.md
```

Expected: no Web UI production file, frontend file, CI file, dependency file, or unrelated repository content changed. Confirm no command writes IDs into env files and `go.mod` has no VMDocker V2 package dependency.

- [ ] **Step 5: Commit documentation**

```bash
git add README.md
git commit -m "docs: document vmdocker v2 CLI workflow"
```
