package server

import (
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/runmedev/runme/v3/pkg/agent/logs"
)

func loadStaticAssets(staticAssets string) (fs.FS, error) {
	if strings.TrimSpace(staticAssets) == "" {
		return nil, nil
	}

	info, err := os.Stat(staticAssets)
	if err != nil {
		return nil, errors.Wrap(err, "failed to inspect static assets directory")
	}
	if !info.IsDir() {
		return nil, errors.Errorf("static assets path %q is not a directory", staticAssets)
	}

	indexPath := filepath.Join(staticAssets, "index.html")
	index, err := os.Open(indexPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open static assets index.html")
	}
	defer index.Close()

	indexInfo, err := index.Stat()
	if err != nil {
		return nil, errors.Wrap(err, "failed to inspect static assets index.html")
	}
	if !indexInfo.Mode().IsRegular() {
		return nil, errors.Errorf("static assets index %q is not a regular file", indexPath)
	}

	return os.DirFS(staticAssets), nil
}

func singlePageAppHandler(assetsFS fs.FS, origins []string) http.Handler {
	fileServer := http.FileServer(http.FS(assetsFS))

	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		indexRequest := r.Clone(r.Context())
		indexURL := *r.URL
		indexURL.Path = "/"
		indexURL.RawPath = ""
		indexRequest.URL = &indexURL
		fileServer.ServeHTTP(w, indexRequest)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assetPath := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if assetPath == "" || assetPath == "." || assetPath == "index.html" {
			serveIndex(w, r)
			return
		}

		if _, err := fs.Stat(assetsFS, assetPath); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				http.Error(w, "failed to inspect static asset", http.StatusInternalServerError)
				return
			}
			serveIndex(w, r)
			return
		}

		fileServer.ServeHTTP(w, r)
	})

	staticOrigins := make([]string, 0, len(origins))
	for _, origin := range origins {
		if origin == "*" {
			logs.NewLogger().Info("Ignoring wildcard CORS origin for static assets")
			continue
		}
		staticOrigins = append(staticOrigins, origin)
	}
	return wrapWithCORS(handler, staticOrigins, false)
}
