package openclawui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	frontendassets "github.com/hymatrix/hype/frontend"
)

func ResolveConfig(cfg Config) (Config, error) {
	if strings.TrimSpace(cfg.Listen) == "" {
		cfg.Listen = defaultListen
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultCommandTimeout
	}
	if strings.TrimSpace(cfg.BinaryPath) == "" {
		binaryPath, err := os.Executable()
		if err != nil {
			return Config{}, err
		}
		cfg.BinaryPath = binaryPath
	}
	if strings.TrimSpace(cfg.WorkingDir) == "" {
		workingDir, err := os.Getwd()
		if err != nil {
			return Config{}, err
		}
		cfg.WorkingDir = workingDir
	}
	return cfg, nil
}

func Run(cfg Config) error {
	return RunContext(context.Background(), cfg)
}

func RunContext(ctx context.Context, cfg Config) error {
	cfg, err := ResolveConfig(cfg)
	if err != nil {
		return err
	}

	handler, err := NewHandler(cfg)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:    cfg.Listen,
		Handler: handler,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	return nil
}

func NewHandler(cfg Config) (http.Handler, error) {
	cfg, err := ResolveConfig(cfg)
	if err != nil {
		return nil, err
	}

	distFS, err := fs.Sub(frontendassets.DistFS, "dist")
	if err != nil {
		return nil, err
	}
	indexHTML, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	store := &spawnStore{}
	registerAPIRoutes(mux, cfg, store)
	registerStaticRoutes(mux, distFS, indexHTML)
	return loggingMiddleware(mux), nil
}

func registerAPIRoutes(mux *http.ServeMux, cfg Config, store *spawnStore) {
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		hypeVersion, hymxVersion := readHypeVersions(cfg.BinaryPath, cfg.WorkingDir)
		writeJSON(w, http.StatusOK, healthResponse{
			OK:          true,
			HypeBinary:  cfg.BinaryPath,
			Listen:      cfg.Listen,
			WorkingDir:  cfg.WorkingDir,
			VmdockerDir: suggestedVmdockerDir(cfg.WorkingDir),
			HypeVersion: hypeVersion,
			HymxVersion: hymxVersion,
		})
	})

	mux.HandleFunc("GET /api/catalog", func(w http.ResponseWriter, r *http.Request) {
		roots, err := buildCatalog()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, catalogResponse{
			OK:    true,
			Roots: roots,
		})
	})

	mux.HandleFunc("POST /api/env/load", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req envLoadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, envLoadResponse{
				OK:    false,
				Error: "invalid request body: " + err.Error(),
			})
			return
		}

		envPath := strings.TrimSpace(req.Path)
		if envPath == "" {
			writeJSON(w, http.StatusBadRequest, envLoadResponse{
				OK:    false,
				Error: "path is required",
			})
			return
		}
		if !filepath.IsAbs(envPath) && strings.TrimSpace(cfg.WorkingDir) != "" {
			envPath = filepath.Join(cfg.WorkingDir, envPath)
		}
		envPath = filepath.Clean(envPath)

		content, err := os.ReadFile(envPath)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, envLoadResponse{
				OK:    false,
				Error: err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, envLoadResponse{
			OK:       true,
			FileName: filepath.Base(envPath),
			Path:     envPath,
			Content:  string(content),
		})
	})

	handleCommand := func(root, subcmd string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req openclawRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, openclawResponse{
					OK:      false,
					Error:   "invalid request body: " + err.Error(),
					Command: []string{},
				})
				return
			}

			var (
				result runResult
				err    error
			)
			switch root {
			case "openclaw":
				result, err = runOpenclaw(r.Context(), cfg, subcmd, req, store)
			case "claude":
				result, err = runClaude(r.Context(), cfg, subcmd, req, store)
			case "vmdocker":
				result, err = runVmdocker(r.Context(), cfg, subcmd, req, store)
			default:
				err = errors.New("unsupported command root")
			}
			if err != nil {
				status := http.StatusBadRequest
				if !errors.Is(err, errValidation) {
					status = http.StatusInternalServerError
				}
				writeJSON(w, status, openclawResponse{
					OK:          false,
					Error:       err.Error(),
					Command:     []string{},
					SpawnedPIDs: store.list(),
				})
				return
			}
			writeJSON(w, http.StatusOK, result.response)
		}
	}

	mux.HandleFunc("POST /api/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req runRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, openclawResponse{
				OK:      false,
				Error:   "invalid request body: " + err.Error(),
				Command: []string{},
			})
			return
		}

		result, err := runCatalogCommand(r.Context(), cfg, req, store)
		if err != nil {
			status := http.StatusBadRequest
			if !errors.Is(err, errValidation) {
				status = http.StatusInternalServerError
			}
			writeJSON(w, status, openclawResponse{
				OK:          false,
				Error:       err.Error(),
				Command:     []string{},
				SpawnedPIDs: store.list(),
			})
			return
		}
		writeJSON(w, http.StatusOK, result.response)
	})

	mux.HandleFunc("POST /api/openclaw/spawn", handleCommand("openclaw", "spawn"))
	mux.HandleFunc("POST /api/openclaw/conf-tg", handleCommand("openclaw", "conf-tg"))
	mux.HandleFunc("POST /api/openclaw/pair-tg", handleCommand("openclaw", "pair-tg"))
	mux.HandleFunc("POST /api/openclaw/chat", handleCommand("openclaw", "chat"))
	mux.HandleFunc("POST /api/claude/spawn", handleCommand("claude", "spawn"))
	mux.HandleFunc("POST /api/claude/chat", handleCommand("claude", "chat"))
	mux.HandleFunc("POST /api/claude/exec", handleCommand("claude", "exec"))
	mux.HandleFunc("POST /api/vmdocker/get", handleCommand("vmdocker", "get"))
	mux.HandleFunc("POST /api/vmdocker/init", handleCommand("vmdocker", "init"))
}

func registerStaticRoutes(mux *http.ServeMux, distFS fs.FS, indexHTML []byte) {
	staticFS := http.FileServer(http.FS(distFS))

	mux.Handle("/assets/", staticFS)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/assets/") {
			staticFS.ServeHTTP(w, r)
			return
		}

		if filepath := strings.TrimPrefix(path.Clean(r.URL.Path), "/"); filepath != "." && filepath != "" {
			if _, err := fs.Stat(distFS, filepath); err == nil {
				staticFS.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(indexHTML))
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func suggestedVmdockerDir(workingDir string) string {
	workingDir = strings.TrimSpace(workingDir)
	if workingDir == "" {
		return "./vmdocker"
	}

	siblingDir := filepath.Join(filepath.Dir(workingDir), "vmdocker")
	if info, err := os.Stat(siblingDir); err == nil && info.IsDir() {
		return siblingDir
	}

	localDir := filepath.Join(workingDir, "vmdocker")
	if info, err := os.Stat(localDir); err == nil && info.IsDir() {
		return localDir
	}

	return "./vmdocker"
}
