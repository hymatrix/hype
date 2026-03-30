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

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
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
			HypeVersion: hypeVersion,
			HymxVersion: hymxVersion,
		})
	})

	handleOpenclaw := func(subcmd string) http.HandlerFunc {
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

			result, err := runOpenclaw(context.Background(), cfg, subcmd, req, store)
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

	mux.HandleFunc("POST /api/openclaw/spawn", handleOpenclaw("spawn"))
	mux.HandleFunc("POST /api/openclaw/conf-tg", handleOpenclaw("conf-tg"))
	mux.HandleFunc("POST /api/openclaw/pair-tg", handleOpenclaw("pair-tg"))
	mux.HandleFunc("POST /api/openclaw/chat", handleOpenclaw("chat"))
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
