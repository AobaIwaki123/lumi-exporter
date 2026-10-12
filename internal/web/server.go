// Package web provides the thin pull-and-render HTTP service for the PoC.
package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/AobaIwaki123/lumi-exporter/internal/pull"
	"github.com/AobaIwaki123/lumi-exporter/internal/registry"
)

//go:embed assets/index.html assets/openapi.yaml
var assets embed.FS

type Server struct {
	config registry.Config
	puller *pull.Client
	logger *slog.Logger
}

func NewServer(config registry.Config, puller *pull.Client, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{config: config, puller: puller, logger: logger}
}

type Catalog struct {
	PulledAt string        `json:"pulled_at"`
	Results  []pull.Result `json:"results"`
}

type trackingWriter struct {
	http.ResponseWriter
	status  int
	started time.Time
}

func (w *trackingWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.Header().Set("X-Runtime-Ms", fmt.Sprint(time.Since(w.started).Milliseconds()))
	w.ResponseWriter.WriteHeader(status)
}

func (w *trackingWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	started := time.Now()
	tracked := &trackingWriter{ResponseWriter: writer, started: started}
	defer func() {
		s.logger.Info("request",
			"method", request.Method,
			"path", request.URL.RequestURI(),
			"status", tracked.status,
			"duration_ms", time.Since(started).Milliseconds(),
			"remote", request.RemoteAddr,
		)
	}()

	path := strings.TrimSuffix(request.URL.Path, "/")
	if path == "" {
		path = "/"
	}
	if request.Method == http.MethodOptions {
		setCORS(tracked)
		tracked.WriteHeader(http.StatusNoContent)
		return
	}
	switch {
	case path == "/" || path == "/index.html":
		s.serveAsset(tracked, request, "assets/index.html", "text/html; charset=utf-8")
	case path == "/openapi.yaml":
		s.serveAsset(tracked, request, "assets/openapi.yaml", "application/yaml; charset=utf-8")
	case path == "/healthz":
		if request.Method != http.MethodGet {
			methodNotAllowed(tracked, http.MethodGet)
			return
		}
		writeJSON(tracked, http.StatusOK, map[string]any{"status": "ok", "service": "lumi-exporter"})
	case path == "/api/v1/sources":
		if request.Method != http.MethodGet {
			methodNotAllowed(tracked, http.MethodGet)
			return
		}
		setCORS(tracked)
		writeJSON(tracked, http.StatusOK, s.config)
	case path == "/api/v1/catalog" || path == "/api/v1/pull":
		if request.Method != http.MethodGet {
			methodNotAllowed(tracked, http.MethodGet)
			return
		}
		setCORS(tracked)
		results := pull.PullAll(request.Context(), s.puller, s.config)
		writeJSON(tracked, http.StatusOK, Catalog{PulledAt: time.Now().UTC().Format(time.RFC3339Nano), Results: results})
	case strings.HasPrefix(path, "/api/v1/sources/") && strings.HasSuffix(path, "/pull"):
		s.serveSourcePull(tracked, request, path)
	default:
		writeJSON(tracked, http.StatusNotFound, map[string]any{"error": map[string]string{"code": "not_found", "message": "endpoint not found"}})
	}
}

func (s *Server) serveSourcePull(writer *trackingWriter, request *http.Request, path string) {
	if request.Method != http.MethodGet {
		methodNotAllowed(writer, http.MethodGet)
		return
	}
	setCORS(writer)
	encoded := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(path, "/api/v1/sources/"), "/"), "/pull")
	id, err := url.PathUnescape(encoded)
	if err != nil || id == "" {
		writeJSON(writer, http.StatusBadRequest, map[string]any{"error": map[string]string{"code": "invalid_source", "message": "invalid source id"}})
		return
	}
	source, ok := s.config.Find(id)
	if !ok {
		writeJSON(writer, http.StatusNotFound, map[string]any{"error": map[string]string{"code": "source_not_found", "message": "source is not registered"}})
		return
	}
	if !source.Enabled {
		writeJSON(writer, http.StatusOK, pull.Result{Source: source, PulledAt: time.Now().UTC().Format(time.RFC3339Nano), Error: "source is disabled"})
		return
	}
	writeJSON(writer, http.StatusOK, s.puller.Pull(request.Context(), source))
}

func (s *Server) serveAsset(writer *trackingWriter, request *http.Request, name, contentType string) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		methodNotAllowed(writer, http.MethodGet)
		return
	}
	body, err := assets.ReadFile(name)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]any{"error": map[string]string{"code": "asset_not_found", "message": "embedded asset not found"}})
		return
	}
	writer.Header().Set("Content-Type", contentType)
	writer.Header().Set("Content-Length", fmt.Sprint(len(body)))
	writer.WriteHeader(http.StatusOK)
	if request.Method == http.MethodGet {
		_, _ = writer.Write(body)
	}
}

func setCORS(writer http.ResponseWriter) {
	writer.Header().Set("Access-Control-Allow-Origin", "*")
	writer.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	writer.Header().Set("Access-Control-Expose-Headers", "X-Runtime-Ms")
}

func methodNotAllowed(writer *trackingWriter, allowed string) {
	writer.Header().Set("Allow", allowed)
	writeJSON(writer, http.StatusMethodNotAllowed, map[string]any{"error": map[string]string{"code": "method_not_allowed", "message": "method not allowed"}})
}

func writeJSON(writer *trackingWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		status = http.StatusInternalServerError
		body = []byte(`{"error":{"code":"serialization_failed","message":"could not encode response"}}`)
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("Content-Length", fmt.Sprint(len(body)))
	writer.WriteHeader(status)
	_, _ = writer.Write(body)
}
