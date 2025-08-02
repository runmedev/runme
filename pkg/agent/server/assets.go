package server

import (
	"bytes"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/url"
	"os"

	"github.com/runmedev/runme/v3/pkg/agent/logs"

	agentv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"

	"github.com/go-logr/zapr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

//go:embed dist/index.*
var embeddedAssets embed.FS

// getAssetFileSystem returns a filesystem for serving web assets
// Uses staticAssets directory if specified, otherwise uses embedded assets
// Returns error if neither option is available
func getAssetFileSystem(staticAssets string) (fs.FS, error) {
	log := zapr.NewLogger(zap.L())
	// If staticAssets is provided, prefer it
	if staticAssets != "" {
		log.Info("Serving static assets", "dir", staticAssets)
		return os.DirFS(staticAssets), nil
	}

	// Try to use embedded assets
	distFS, _ := fs.Sub(embeddedAssets, "dist")
	_, err := distFS.Open("index.html")
	if err == nil {
		log.Info("Serving embedded assets")
		return distFS, nil
	}

	// Neither staticAssets is set nor embedded assets are available
	return nil, errors.New("no assets available: neither staticAssets directory is configured nor embedded assets could be found")
}

// processIndexHTMLWithConfig reads the index.html file and injects configuration values
// such as authentication requirements into the HTML content
func (s *Server) processIndexHTMLWithConfig(assetsFS fs.FS) ([]byte, error) {
	// Read index.html
	file, err := assetsFS.Open("index.html")
	if err != nil {
		return nil, errors.Wrap(err, "failed to open index.html")
	}
	defer func() {
		if err := file.Close(); err != nil {
			zap.L().Error("failed to close index.html file", zap.Error(err))
		}
	}()

	// Read the file content
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(file); err != nil {
		return nil, errors.Wrap(err, "failed to read index.html content")
	}
	content := buf.Bytes()

	type initialState struct {
		RequireAuth  bool                  `json:"requireAuth"`
		WebAppConfig *agentv1.WebAppConfig `json:"webApp,omitempty"`
	}

	state := initialState{RequireAuth: false, WebAppConfig: s.webAppConfig}
	if s.serverConfig.OIDC != nil {
		state.RequireAuth = true
	}

	jsonState, err := json.Marshal(state)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal initial state")
	}

	// Replace the assignment in index.html
	placeholder := "window.__INITIAL_STATE__ = {}"
	replacement := "window.__INITIAL_STATE__ = " + string(jsonState)
	content = bytes.ReplaceAll(content, []byte(placeholder), []byte(replacement))

	return content, nil
}

// singlePageAppHandler serves a single-page app from static or embedded assets,
// falling back to index for client-side routing when files don't exist.
func (s *Server) singlePageAppHandler() (http.Handler, error) {
	if s.assetsFS == nil {
		// This shouldn't happen because this should have been initialized in new.
		return nil, errors.New("assets fs not configured")
	}
	fileServer := http.FileServer(http.FS(s.assetsFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := "/"
		if len(r.URL.Path) > 1 {
			path = r.URL.Path[1:]
		}

		// If path is empty, file doesn't exist, or it's index.html, serve processed index
		if path == "/" || path == "index.html" || os.IsNotExist(func() error {
			_, err := s.assetsFS.Open(path)
			return err
		}()) {
			// Read and process index.html
			s.serveIndexHTML(w, r)
		}

		fileServer.ServeHTTP(w, r)
	}), nil
}

// serveIndexHTML is the handler that serves the main SPA page.
func (s *Server) serveIndexHTML(w http.ResponseWriter, r *http.Request) {
	if s.serverConfig.WebAppURL != "" {
		// If we are serving on a different URL then we just redirect
		redirectURL, err := url.Parse(s.serverConfig.WebAppURL)
		if err != nil {
			log := logs.FromContext(r.Context())
			log.Error(err, "Invalid target URL: %v", s.serverConfig.WebAppURL)
		}

		redirectURL.Path = r.URL.Path
		redirectURL.RawQuery = r.URL.RawQuery
		http.Redirect(w, r, redirectURL.String(), http.StatusFound)
		return
	}
	// Read and process index.html
	content, err := s.processIndexHTMLWithConfig(s.assetsFS)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set content type and write the modified content
	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write(content)
}
