package server

import (
	"fmt"
	"net/http"

	connectcors "connectrpc.com/cors"

	"github.com/rs/cors"

	"github.com/runmedev/runme/v3/pkg/agent/iam"
	"github.com/runmedev/runme/v3/pkg/agent/logs"

	"github.com/pkg/errors"

	"github.com/runmedev/runme/v3/pkg/agent/config"
)

// AuthMux wraps http.ServeMux to add protected route handling
type AuthMux struct {
	mux            *http.ServeMux
	authMiddleware func(http.Handler) http.Handler
	serverConfig   *config.AssistantServerConfig
}

// NewAuthMux creates a new AuthMux
func NewAuthMux(serverConfig *config.AssistantServerConfig, oidc *iam.OIDC) (*AuthMux, error) {
	mux := http.NewServeMux()

	// Create auth middleware if OIDC is configured
	var authMiddleware func(http.Handler) http.Handler
	if oidc != nil {
		middleware, err := iam.NewAuthMiddleware(oidc)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create auth middleware")
		}
		authMiddleware = middleware
	} else {
		// No-op middleware if OIDC is not configured
		authMiddleware = func(next http.Handler) http.Handler {
			return next
		}
	}

	return &AuthMux{
		mux:            mux,
		authMiddleware: authMiddleware,
		serverConfig:   serverConfig,
	}, nil
}

// Handle registers a handler for the given pattern
func (p *AuthMux) Handle(pattern string, handler http.Handler) {
	p.mux.Handle(pattern, handler)
}

// HandleFunc registers a handler function for the given pattern
func (p *AuthMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	p.mux.HandleFunc(pattern, handler)
}

// HandleProtected registers a protected handler for the given pattern
func (p *AuthMux) HandleProtected(pattern string, handler http.Handler, checker iam.Checker, role string) {
	// We need to create a new Auth middleware that will apply the IAM checks specific to this handler
	iamChecker := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logs.FromContext(r.Context())

			idToken, err := iam.GetIDToken(r.Context())
			// Nil token is not fatal until authz check
			if err != nil {
				log.Info("Unauthenticated: ", "error", err)
			}

			principal, err := checker.GetPrincipal(idToken)
			if err != nil {
				log.Info("Unauthorized: ", "error", err)
				http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}

			if !checker.Check(principal, role) {
				log.Info("Unauthorized", "user", principal, "role", role)
				http.Error(w, fmt.Sprintf("Forbidden: user %s doesn't have role %s", principal, role), http.StatusForbidden)
				return
			}

			// Get the IDToken from the context
			next.ServeHTTP(w, r)
		})
	}
	log := logs.NewLogger()
	// Create a chain: cors > check the IDToken is valid -> Apply AuthZ -> call the handler
	// CORS needs to come first because it will terminate the request chain on OPTIONS requests
	// OPTIONS requests won't carry authorization headers so we can't do authorization first
	handler = p.authMiddleware(iamChecker(handler))
	// Apply CORS if origins are configured
	// This is modeled on cors.AllowAll() but we can't use that because we need to allow credentials
	corsOptions := cors.Options{
		AllowedOrigins: p.serverConfig.CorsOrigins,
		// Allow all methods.
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		// Allow all headers.
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   connectcors.ExposedHeaders(),
		AllowCredentials: true,
		MaxAge:           7200, // 2 hours in seconds
	}
	if len(p.serverConfig.CorsOrigins) > 0 {
		log.Info("Adding CORS support", "AllowedOrigins", corsOptions.AllowedOrigins, "AllowCredentials", corsOptions.AllowCredentials, "AllowedMethods", corsOptions.AllowedMethods, "AllowedHeaders", corsOptions.AllowedHeaders, "ExposedHeaders", corsOptions.ExposedHeaders)

		if p.serverConfig.CorsOrigins[0] == "*" {
			log.Info("Allowing all origins; enabling SetOriginHeader middleware")
			// We need to set the origin header to the request's origin
			// To do that we need to set the passthrough option to true so that the handler will invoke our middleware
			// after calling the cors handler
			corsOptions.OptionsPassthrough = true
			corsOptions.Debug = true
			c := cors.New(corsOptions)
			handler = c.Handler(SetOriginHeader(handler))
		} else {
			c := cors.New(corsOptions)
			handler = c.Handler(handler)
		}

	}

	p.mux.Handle(pattern, handler)
}

// HandleProtectedFunc registers a protected handler function for the given pattern
func (p *AuthMux) HandleProtectedFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	p.mux.Handle(pattern, p.authMiddleware(http.HandlerFunc(handler)))
}

// ServeHTTP implements http.Handler
func (p *AuthMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mux.ServeHTTP(w, r)
}

// RegisterAuthRoutes registers the OAuth2 authentication routes
func RegisterAuthRoutes(oidc *iam.OIDC, mux *AuthMux) error {
	if oidc == nil {
		return nil
	}

	// Register OAuth2 endpoints
	mux.HandleFunc(iam.OIDCPathPrefix+"/login", oidc.LoginHandler)
	mux.HandleFunc(iam.OIDCPathPrefix+"/callback", oidc.CallbackHandler)
	mux.HandleFunc(iam.OIDCPathPrefix+"/logout", oidc.LogoutHandler)
	mux.HandleFunc("/logout", oidc.LogoutHandler)

	return nil
}
