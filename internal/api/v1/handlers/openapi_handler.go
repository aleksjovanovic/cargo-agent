package handlers

import (
	"net/http"
	"os"
	"path/filepath"
)

// func (h *Handler) OpenAPIHandler() http.HandlerFunc {
// 	exe, _ := os.Executable()
// 	base := filepath.Dir(exe)
// 	specPath := filepath.Join(base, "internal", "docs", "cargo-agent.yaml")

// 	return func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Set("Content-Type", "application/x-yaml; charset=utf-8")
// 		http.ServeFile(w, r, specPath)
// 	}
// }

func (h *Handler) OpenAPIHandler() http.HandlerFunc {
	// Za dev okruženje radimo relativno na CWD (go run)
	wd, _ := os.Getwd()
	specPath := filepath.Join(wd, "internal", "docs", "cargo-agent.yaml")

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml; charset=utf-8")
		http.ServeFile(w, r, specPath)
	}
}

func (h *Handler) OpenAPIUIHandler() http.HandlerFunc {
	const page = `<!doctype html>
<html>
  <head>
    <meta charset="utf-8"/>
    <title>Cargo Agent API Docs</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist/swagger-ui.css"/>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js"></script>
    <script>
      window.ui = SwaggerUIBundle({
        url: '/cargo-agent/v1/docs/cargo-agent.yaml', // koristimo apsolutni URL
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
        layout: "BaseLayout"
      });
    </script>
  </body>
</html>`
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(page))
	}
}
