package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	repoRoot, err := findRepoRoot()
	if err != nil {
		log.Fatalf("resolve repo root failed: %v", err)
	}

	listen := defaultListen
	if v := strings.TrimSpace(os.Getenv("OPENCLAW_WEBUI_LISTEN")); v != "" {
		listen = v
	}

	mux := http.NewServeMux()
	store := &spawnStore{}
	registerAPIRoutes(mux, repoRoot, listen, store)
	registerStaticRoutes(mux, repoRoot)

	server := &http.Server{
		Addr:    listen,
		Handler: loggingMiddleware(mux),
	}

	log.Printf("openclaw webui listening on http://%s", listen)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped with error: %v", err)
	}
}

func registerAPIRoutes(mux *http.ServeMux, repoRoot, listen string, store *spawnStore) {
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		binary, err := resolveHypeBinary(repoRoot)
		if err != nil {
			writeJSON(w, http.StatusOK, healthResponse{OK: false, HypeBinary: err.Error(), Listen: listen})
			return
		}
		hypeVersion, hymxVersion := readHypeVersions(binary, repoRoot)
		writeJSON(w, http.StatusOK, healthResponse{
			OK:          true,
			HypeBinary:  binary,
			Listen:      listen,
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
					Error:   fmt.Sprintf("invalid request body: %v", err),
					Command: []string{},
				})
				return
			}

			result, err := runOpenclaw(context.Background(), repoRoot, subcmd, req, store)
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

func registerStaticRoutes(mux *http.ServeMux, repoRoot string) {
	distDir := filepath.Join(repoRoot, "frontend", "dist")
	staticFS := http.FileServer(http.Dir(distDir))

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

		indexPath := filepath.Join(distDir, "index.html")
		if _, err := os.Stat(indexPath); err != nil {
			if r.URL.Path == "/" {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte("frontend build not found. run: cd frontend && npm install && npm run build"))
				return
			}
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, indexPath)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			log.Printf("%s %s", r.Method, r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	current := cwd
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(current, "cmd", "hype", "main.go")); err == nil {
				return current, nil
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", errors.New("cannot locate hype repo root")
}

func readHypeVersions(binaryPath, repoRoot string) (string, string) {
	cmd := exec.Command(binaryPath, "-v")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", ""
	}

	var hypeVersion string
	var hymxVersion string
	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Version:") {
			hypeVersion = strings.TrimSpace(strings.TrimPrefix(trimmed, "Version:"))
		}
		if strings.HasPrefix(trimmed, "HymxVersion:") {
			hymxVersion = strings.TrimSpace(strings.TrimPrefix(trimmed, "HymxVersion:"))
		}
	}
	return hypeVersion, hymxVersion
}
