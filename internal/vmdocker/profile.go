package vmdocker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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
	if baseImage == "" {
		return "", errors.New("from is required")
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	targetExisted := false
	info, err := m.stat(absDir)
	switch {
	case err == nil && !info.IsDir():
		return "", fmt.Errorf("profile target is not a directory: %s", absDir)
	case err == nil:
		targetExisted = true
		entries, err := m.readDir(absDir)
		if err != nil {
			return "", err
		}
		if len(entries) != 0 {
			return "", fmt.Errorf("profile target directory is not empty: %s", absDir)
		}
	case !os.IsNotExist(err):
		return "", err
	}

	files := map[string]string{
		"profile.toml":     fmt.Sprintf(profileTemplate, strconv.Quote(baseImage)),
		"bin/.keep":        "",
		"skills/soul.md":   "MY-SOUL\n",
		"persona/style.md": "terse, precise\n",
	}
	written := make([]string, 0, len(files))
	for rel, content := range files {
		path := filepath.Join(absDir, filepath.FromSlash(rel))
		if err := m.mkdirAll(filepath.Dir(path), 0o755); err != nil {
			m.cleanupProfileInitFailure(absDir, written, targetExisted)
			return "", err
		}
		if err := m.writeFile(path, []byte(content), 0o644); err != nil {
			m.cleanupProfileInitFailure(absDir, written, targetExisted)
			return "", err
		}
		written = append(written, path)
	}
	return absDir, nil
}

func (m *Manager) cleanupProfileInitFailure(absDir string, written []string, targetExisted bool) {
	if !targetExisted {
		_ = m.removeAllPath(absDir)
		return
	}
	for i := len(written) - 1; i >= 0; i-- {
		_ = m.removePath(written[i])
	}
	for _, rel := range []string{"bin", "skills", "persona"} {
		_ = m.removePath(filepath.Join(absDir, rel))
	}
}

func (m *Manager) removePath(path string) error {
	if m.remove == nil {
		return os.Remove(path)
	}
	return m.remove(path)
}

func (m *Manager) removeAllPath(path string) error {
	if m.removeAll == nil {
		return os.RemoveAll(path)
	}
	return m.removeAll(path)
}
