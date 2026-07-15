package vmdocker

import (
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type Manager struct {
	runner    CommandRunner
	httpGet   HTTPGetter
	stat      func(string) (os.FileInfo, error)
	readDir   func(string) ([]fs.DirEntry, error)
	readFile  func(string) ([]byte, error)
	writeFile func(string, []byte, os.FileMode) error
	chmod     func(string, os.FileMode) error
	mkdirAll  func(string, os.FileMode) error
	remove    func(string) error
	rename    func(string, string) error
	sleep     func(time.Duration)
	listen    func(network, address string) (net.Listener, error)
	dial      func(network, address string, timeout time.Duration) (net.Conn, error)
	glob      func(pattern string) ([]string, error)
	kill      func(int, syscall.Signal) error
	out       io.Writer
	errOut    io.Writer
}

func NewManager() *Manager {
	return &Manager{
		runner:    ExecCommandRunner{},
		httpGet:   DefaultHTTPGetter(),
		stat:      os.Stat,
		readDir:   os.ReadDir,
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
		chmod:     os.Chmod,
		mkdirAll:  os.MkdirAll,
		remove:    os.Remove,
		rename:    os.Rename,
		sleep:     time.Sleep,
		listen:    net.Listen,
		dial:      net.DialTimeout,
		glob:      filepath.Glob,
		kill:      syscall.Kill,
	}
}

func (m *Manager) SetOutput(out io.Writer) {
	m.out = out
}

func (m *Manager) SetErrorOutput(out io.Writer) {
	m.errOut = out
}

func (m *Manager) printf(format string, args ...any) {
	if m.out == nil {
		return
	}
	_, _ = fmt.Fprintf(m.out, format, args...)
}
