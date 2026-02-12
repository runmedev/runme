package server

import (
	"bytes"
	"io/fs"
	"net/http"
	"net/url"
	"os"

	"github.com/jlewi/monogo/helpers"

	agentv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
	"github.com/runmedev/runme/v3/pkg/agent/config"
	"github.com/runmedev/runme/v3/pkg/agent/logs"

	"github.com/go-logr/zapr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

// AssetsFileSystemProvider is an interface for providing asset filesystems.
// This allows the server to be decoupled from how assets are constructed.
type AssetsFileSystemProvider interface {
	GetAssetsFileSystem() (fs.FS, error)
}

// staticAssetsFileSystemProvider provides assets from a static directory.
type staticAssetsFileSystemProvider struct {
	staticAssets string
}

// NewStaticAssetsFileSystemProvider creates a provider that serves assets from a directory.
func NewStaticAssetsFileSystemProvider(staticAssets string) AssetsFileSystemProvider {
	return &staticAssetsFileSystemProvider{
		staticAssets: staticAssets,
	}
}

// GetAssetsFileSystem implements AssetsFileSystemProvider by returning a filesystem
// for the static assets directory.
func (s *staticAssetsFileSystemProvider) GetAssetsFileSystem() (fs.FS, error) {
	if s.staticAssets == "" {
		return nil, errors.New("static assets directory is not configured")
	}
	log := zapr.NewLogger(zap.L())
	log.Info("Serving static assets", "dir", s.staticAssets)
	return os.DirFS(s.staticAssets), nil
}

// fallbackAssetsFileSystemProvider tries multiple providers in order until one succeeds.
type fallbackAssetsFileSystemProvider struct {
	providers []AssetsFileSystemProvider
}

// NewFallbackAssetsFileSystemProvider creates a provider that tries multiple providers
// in order until one succeeds. Returns the first error if all providers fail.
func NewFallbackAssetsFileSystemProvider(providers ...AssetsFileSystemProvider) AssetsFileSystemProvider {
	return &fallbackAssetsFileSystemProvider{
		providers: providers,
	}
}

// GetAssetsFileSystem implements AssetsFileSystemProvider by trying each provider
// in order until one succeeds.
func (f *fallbackAssetsFileSystemProvider) GetAssetsFileSystem() (fs.FS, error) {
	var lastErr error
	for _, provider := range f.providers {
		fs, err := provider.GetAssetsFileSystem()
		if err == nil {
			return fs, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		return nil, errors.New("no asset providers configured")
	}
	return nil, errors.Wrapf(lastErr, "all asset providers failed")
}

// defaultAssetsFileSystemProvider is the default implementation that serves
// assets from the static assets directory.
type defaultAssetsFileSystemProvider struct {
	staticAssets string
}

// NewDefaultAssetsFileSystemProvider creates a new default asset filesystem provider
// that serves assets from the static assets directory.
func NewDefaultAssetsFileSystemProvider(staticAssets string) AssetsFileSystemProvider {
	return &defaultAssetsFileSystemProvider{
		staticAssets: staticAssets,
	}
}

// GetAssetsFileSystem implements AssetsFileSystemProvider by serving static assets.
func (d *defaultAssetsFileSystemProvider) GetAssetsFileSystem() (fs.FS, error) {
	fs, err := NewStaticAssetsFileSystemProvider(d.staticAssets).GetAssetsFileSystem()
	if err != nil {
		return nil, errors.New("no assets available: staticAssets directory is not configured")
	}
	return fs, nil
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

	state := agentv1.InitialConfigState{
		WebApp:      s.webAppConfig,
		RequireAuth: false,
		SystemShell: config.SystemShell(),
	}

	if s.serverConfig.OIDC != nil {
		state.RequireAuth = true
	}

	jsonState, err := protojson.Marshal(&state)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal initial state")
	}

	// Replace the assignment in index.html
	placeholder := "window.__INITIAL_STATE__ = {}"
	replacement := "window.__INITIAL_STATE__ = " + string(jsonState)
	content = bytes.ReplaceAll(content, []byte(placeholder), []byte(replacement))

	return content, nil
}

// singlePageAppHandler serves a single-page app from static assets,
// falling back to index for client-side routing when files don't exist.
func (s *Server) singlePageAppHandler() (http.Handler, error) {
	if s.assetsFS == nil {
		// This shouldn't happen because this should have been initialized in new.
		return nil, errors.New("assets fs not configured")
	}
	fileServer := http.FileServer(http.FS(s.assetsFS))

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := "/"
		if len(r.URL.Path) > 1 {
			path = r.URL.Path[1:]
		}

		// If path is empty, file doesn't exist, or it's index.html, serve processed index
		if path == "/" || path == "index.html" || os.IsNotExist(func() error {
			f, err := s.assetsFS.Open(path)
			if f != nil {
				helpers.DeferIgnoreError(f.Close)
			}
			return err
		}()) {
			// Read and process index.html
			s.serveIndexHTML(w, r)
			return
		}

		fileServer.ServeHTTP(w, r)
	})

	// Allow the SPA to fetch assets/configs from a dev server origin.
	// We don't allow credentials for static assets and we do not allow wildcard origins.
	origins := make([]string, 0, len(s.serverConfig.CorsOrigins))
	removedWildcard := false
	for _, origin := range s.serverConfig.CorsOrigins {
		if origin == "*" {
			removedWildcard = true
			continue
		}
		origins = append(origins, origin)
	}
	if removedWildcard {
		log := logs.NewLogger()
		log.Info("Ignoring wildcard origin for static assets", "origins", origins)
	}
	return wrapWithCORS(baseHandler, origins, false), nil
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
