package server

import (
	"net/http"

	"github.com/runmedev/runme/v3/pkg/agent/logs"
)

// SetOriginHeader is middleware that copies the Origin header from the request to the response
// This is necessary when using AllowAllOrigins because the browser will complain if the response header
// is the "*" and not the same origin as on the request. The cors handler in the connect library doesn't do
// this by default.
func SetOriginHeader(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := logs.FromContext(r.Context())
		// Always set the Access-Control-Allow-Origin header to the request's Origin header
		// Not just if its an Options request.
		// It looks like the CORS handler
		// (https://github.com/rs/cors/blob/1084d89a16921942356d1c831fbe523426cf836e/cors.go#L323) checks origin
		// and sets the response headers even if its not an OPTIONS request. So we need to always set the
		// Access-Control-Allow-Origin header to the request's Origin header.
		w.Header()["Access-Control-Allow-Origin"] = r.Header["Origin"]
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			log.V(logs.Debug).Info("Setting Access-Control-Allow-Origin header", "origin", r.Header["Origin"])
			// http.StatusNoContent is used for preflight requests
			w.WriteHeader(http.StatusNoContent)
		} else {
			log.V(logs.Debug).Info("Calling next handler", "method", r.Method, "url", r.URL.String())
			h.ServeHTTP(w, r)
		}
	})
}
