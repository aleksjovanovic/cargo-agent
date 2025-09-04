// internal/api/v1/handlers/openapi_handler.go
package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aleksjovanovic/cargo-agent/internal/ctxmeta"
	"github.com/aleksjovanovic/cargo-agent/internal/logger"
)

// ────────────────────────────────────────────────────────────────────────────────
// Spec file loading & caching
// ────────────────────────────────────────────────────────────────────────────────
//
// Default filesystem path to the OpenAPI YAML when OPENAPI_SPEC_PATH is not set.
// Adjust to your repo layout if needed. You said the file lives at
// internal/docs/cargo-agent.yaml, so we point there by default.
const defaultSpecPath = "internal/docs/cargo-agent.yaml"

// When OPENAPI_SPEC_RELOAD=1 (or "true"), the handler re-reads the YAML on
// each request. Handy in local dev so you don't have to restart the server
// to see doc changes.
var devReload = os.Getenv("OPENAPI_SPEC_RELOAD") == "1" || strings.EqualFold(os.Getenv("OPENAPI_SPEC_RELOAD"), "true")

// Small, process-level cache for the spec contents and metadata.
// We compute a weak ETag from the file contents and keep a Last-Modified
// timestamp taken from the file's modtime (or "now" as a fallback).
var (
	specMu    sync.RWMutex
	specBytes []byte
	specETag  string
	modTime   time.Time
	specPath  = firstNonEmpty(os.Getenv("OPENAPI_SPEC_PATH"), defaultSpecPath)
	onceLoad  sync.Once
)

// UI default can be switched via env (swagger|redoc). We default to Swagger
// because it’s interactive (“Try it out”), which matches your previous setup.
var defaultUI = func() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("OPENAPI_UI_DEFAULT")))
	if v == "redoc" {
		return "redoc"
	}
	return "swagger"
}()

// firstNonEmpty returns the first non-empty string from the given args.
func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}

// readSpec loads the YAML from disk, computes an ETag and sets Last-Modified.
// In dev mode we call this for every request; in prod we load once.
func readSpec() error {
	abs, _ := filepath.Abs(specPath)

	b, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("read openapi spec failed (%s): %w", abs, err)
	}
	if len(b) == 0 {
		return fmt.Errorf("openapi spec is empty (%s)", abs)
	}

	// Try to get filesystem modtime; if that fails, use "now".
	mt := time.Now().UTC()
	if fi, statErr := os.Stat(specPath); statErr == nil && !fi.ModTime().IsZero() {
		mt = fi.ModTime().UTC()
	}

	sum := sha256.Sum256(b)

	specMu.Lock()
	specBytes = b
	specETag = `W/"` + hex.EncodeToString(sum[:8]) + `"` // short, weak ETag; good enough for docs
	modTime = mt
	specMu.Unlock()

	return nil
}

// ensureSpecLoaded makes sure we have something in memory.
// In prod, it's executed once on first use; in dev, we skip once.Do and re-read every time.
func ensureSpecLoaded() error {
	if devReload {
		return readSpec()
	}
	var err error
	onceLoad.Do(func() {
		err = readSpec()
	})
	return err
}

// ────────────────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────────────────
//
// OpenAPIHandler serves the raw YAML spec.
//
// Features:
// - Reads from OPENAPI_SPEC_PATH (or internal/docs/cargo-agent.yaml by default).
// - Adds ETag and Last-Modified headers and honors If-None-Match / If-Modified-Since.
// - Sets Content-Type to application/yaml and a short Cache-Control.
// - Logs with request id for traceability.
//
// NOTE: If you mount this under /cargo-agent/v1/docs/ as in routes, the UI can reach it
// at /cargo-agent/v1/docs/cargo-agent.yaml (relative to the docs root).
func (h *Handler) OpenAPIHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := ctxmeta.RequestID(r.Context())

		// (Re)load spec when needed.
		if err := ensureSpecLoaded(); err != nil {
			logger.Error("openapi.load_failed", "rid", rid, "error", err)
			http.Error(w, "OpenAPI spec not available", http.StatusInternalServerError)
			return
		}

		// Snapshot values under read lock.
		specMu.RLock()
		body := specBytes
		etag := specETag
		mt := modTime
		specMu.RUnlock()

		// Conditional GET using ETag (If-None-Match).
		if inm := r.Header.Get("If-None-Match"); inm != "" && etag != "" && inm == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		// Conditional GET using If-Modified-Since (best-effort).
		if ims := r.Header.Get("If-Modified-Since"); ims != "" && !mt.IsZero() {
			if t, err := time.Parse(http.TimeFormat, ims); err == nil {
				// If the file hasn't changed since IMS, return 304.
				// Use !mt.After(t) to treat equal times as "not modified".
				if !mt.After(t.UTC()) {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}
		}

		// Headers: type + caching + validators.
		// We allow short caching; ETag/Last-Modified let clients validate cheaply.
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=60")
		if etag != "" {
			w.Header().Set("ETag", etag)
		}
		if !mt.IsZero() {
			w.Header().Set("Last-Modified", mt.Format(http.TimeFormat))
		}

		// Log once per serve (avoid logging for 304 to keep noise down).
		logger.Info("openapi.serve_yaml", "rid", rid, "bytes", len(body), "path", r.URL.Path)

		_, _ = w.Write(body)
	}
}

// OpenAPIUIHandler serves the docs UI.
//
// We now DEFAULT to Swagger UI (interactive “Try it out”, like your old setup).
// You can still switch to ReDoc with ?ui=redoc. Both UIs also respect a custom
// ?url=... query param for the spec. If omitted, they fall back to ./cargo-agent.yaml
// (relative to the docs mount).
func (h *Handler) OpenAPIUIHandler() http.HandlerFunc {
	// Minimal HTML templates for Redoc and Swagger UI.
	// Kept inline for simplicity; feel free to move to files if you prefer.

	const redocHTML = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Cargo Agent API Docs (ReDoc)</title>
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <style>
    html,body,#redoc{height:100%}
    body{margin:0;background:#0b1220}
    .switch{position:fixed;top:10px;right:12px;z-index:10}
    .switch a{color:#93c5fd;font-family:system-ui,Segoe UI,Roboto,Arial,sans-serif;text-decoration:none;font-weight:600}
  </style>
</head>
<body>
  <div class="switch"><a href="?ui=swagger">Switch to Swagger UI</a></div>
  <div id="redoc"></div>
  <script>
    // Let users override the spec via ?url=...; otherwise use a relative path
    // so the UI works under /cargo-agent/v1/docs/.
    (function () {
      const qs = new URLSearchParams(window.location.search);
      const specUrl = qs.get('url') || './cargo-agent.yaml';
      const options = {
        scrollYOffset: 50,
        theme: {
          colors: { primary: { main: '#2563eb' } },
          sidebar: { backgroundColor: '#0b1220', textColor: '#e5e7eb' },
          typography: { fontSize: '14px' }
        }
      };
      function start(){ Redoc.init(specUrl, options, document.getElementById('redoc')); }
      if (window.Redoc) start();
      else {
        const s = document.createElement('script');
        s.src = 'https://cdn.jsdelivr.net/npm/redoc@next/bundles/redoc.standalone.js';
        s.onload = start;
        document.body.appendChild(s);
      }
    })();
  </script>
</body>
</html>`

	const swaggerHTML = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Cargo Agent API Docs (Swagger UI)</title>
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
  <style>
    html,body{height:100%;margin:0}
    #swagger-ui{height:100%}
    .switch{position:fixed;top:10px;right:12px;z-index:10}
    .switch a{color:#2563eb;font-family:system-ui,Segoe UI,Roboto,Arial,sans-serif;text-decoration:none;font-weight:600}
  </style>
</head>
<body>
  <div class="switch"><a href="?ui=redoc">Switch to ReDoc</a></div>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      const qs = new URLSearchParams(window.location.search);
      const specUrl = qs.get('url') || './cargo-agent.yaml';
      SwaggerUIBundle({
        url: specUrl,
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis],
        layout: "BaseLayout"
      });
    };
  </script>
</body>
</html>`

	// Parse templates once to fail-fast if there is a syntax error.
	redocT := template.Must(template.New("redoc").Parse(redocHTML))
	swaggerT := template.Must(template.New("swagger").Parse(swaggerHTML))

	return func(w http.ResponseWriter, r *http.Request) {
		rid := ctxmeta.RequestID(r.Context())

		// Simple selector via ?ui=..., otherwise fall back to env default (Swagger).
		ui := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("ui")))
		if ui == "" {
			ui = defaultUI
		}

		// Docs pages shouldn't be cached aggressively (especially in dev).
		// If you want stronger caching in prod, relax this to "public, max-age=300".
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		var err error
		switch ui {
		case "redoc":
			logger.Info("openapi.ui_redoc", "rid", rid, "path", r.URL.Path)
			err = redocT.Execute(w, nil)
		default: // "swagger"
			logger.Info("openapi.ui_swagger", "rid", rid, "path", r.URL.Path)
			err = swaggerT.Execute(w, nil)
		}
		if err != nil {
			logger.Error("openapi.ui_render_failed", "rid", rid, "error", err)
			http.Error(w, "failed to render docs UI", http.StatusInternalServerError)
			return
		}
	}
}
