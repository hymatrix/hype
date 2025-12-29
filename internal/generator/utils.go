package generator

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"unicode"

	"github.com/hymatrix/hype/internal/templates"
)

func renderTemplateFile(tmplName string, outPath string, data any) error {
	b, err := templates.FS.ReadFile(tmplName)
	if err != nil {
		return err
	}
	funcs := template.FuncMap{
		"capitalize": func(s string) string {
			if s == "" {
				return s
			}
			r := []rune(s)
			r[0] = unicode.ToUpper(r[0])
			return string(r)
		},
	}
	t, err := template.New(tmplName).Funcs(funcs).Parse(string(b))
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return err
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return nil
}

func writeRawFile(srcInFS string, outPath string) error {
	b, err := templates.FS.ReadFile(srcInFS)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, b, 0o644); err != nil {
		return err
	}
	return nil
}

func runCmd(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v failed: %w", name, args, err)
	}
	return nil
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		if shouldSkip(p, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		return copyFile(p, out)
	})
}

func shouldSkip(path string, d fs.DirEntry) bool {
	if d.IsDir() {
		ignored := []string{".git", "vendor", "bin", "dist", "build", "node_modules", ".idea", ".vscode"}
		base := filepath.Base(path)
		for _, ig := range ignored {
			if base == ig {
				return true
			}
		}
		return false
	}
	base := filepath.Base(path)
	if base == "go.mod" || base == "go.sum" {
		return true
	}
	return false
}
