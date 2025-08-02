package iam

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/go-logr/zapr"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	agentv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
	streamv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/stream/v1"
	"github.com/runmedev/runme/v3/pkg/agent/config"
	"github.com/runmedev/runme/v3/pkg/agent/logs"
)

var (
	ErrPrincipalExtraction = errors.New("could not extract principal from token")
	ErrRoleDenied          = errors.New("user does not have the required role")
)

const (
	OIDCPathPrefix        = "/oidc"
	SessionCookieName     = "agent-session"
	SessionOAuthTokenName = "agent-oauth-token"
	stateLength           = 32
)

// OIDC handles OAuth2 authentication setup and management
type OIDC struct {
	config     *config.OIDCConfig
	oauth2     *oauth2.Config
	publicKeys map[string]*rsa.PublicKey
	discovery  *openIDDiscovery
	// stateManager manages the state for OAuth2 PKCE.
	// This assumes there is a single instants of the server so that the redirect from the OAuth2 provider
	// will hit the same server instance. If we had multiple instances of the server we would need to use a
	// distributed cache
	state    *stateManager
	provider OIDCProvider
}

// NewOIDC creates a new OIDC
func NewOIDC(cfg *config.OIDCConfig) (*OIDC, error) {
	if cfg == nil {
		return nil, nil
	}

	// Check that only one provider is configured
	if cfg.Google != nil && cfg.Generic != nil {
		return nil, errors.New("both Google and generic OIDC providers cannot be configured at the same time")
	}

	if cfg.Google == nil && cfg.Generic == nil {
		return nil, nil
	}

	var provider OIDCProvider
	if cfg.Google != nil {
		provider = NewGoogleProvider(cfg.Google)
	} else {
		provider = NewGenericProvider(cfg.Generic)
	}

	oauth2Config, err := provider.GetOAuth2Config()
	if err != nil {
		return nil, err
	}

	discoveryURL := provider.GetDiscoveryURL()

	// Fetch the OpenID configuration
	resp, err := http.Get(discoveryURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch OpenID configuration")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			zap.L().Error("failed to close response body", zap.Error(err))
		}
	}()

	var discovery openIDDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return nil, errors.Wrap(err, "failed to decode OpenID configuration")
	}

	// Update endpoints from discovery document
	oauth2Config.Endpoint = oauth2.Endpoint{
		AuthURL:  discovery.AuthURL,
		TokenURL: discovery.TokenURL,
	}

	// If the generic provider is configured with an issuer, use it
	if cfg.Generic != nil && cfg.Generic.Issuer != "" {
		discovery.Issuer = cfg.Generic.Issuer
	}

	// Initialize OIDC
	oidc := &OIDC{
		config:     cfg,
		oauth2:     oauth2Config,
		publicKeys: make(map[string]*rsa.PublicKey),
		discovery:  &discovery,
		state:      newStateManager(10 * time.Minute),
		provider:   provider,
	}

	// Download JWKS for signature verification
	if err := oidc.downloadJWKS(); err != nil {
		return nil, errors.Wrapf(err, "Failed to download JWKS")
	}

	// Start a goroutine to clean up expired states
	go func() {
		ticker := time.NewTicker(oidc.state.stateExpiration / 2)
		defer ticker.Stop()
		for range ticker.C {
			oidc.state.cleanupExpiredStates()
		}
	}()

	return oidc, nil
}

// DoClientExchange true if the token exchange happens on the client
func (o *OIDC) DoClientExchange() bool {
	return o.config.ClientExchange
}

// downloadJWKS downloads the JSON Web Key Set (JWKS) from Google's OAuth2 provider.
// It fetches the public keys used to verify JWT signatures, decodes them from the
// JWK format, and stores them in the OIDC instance's publicKeys map indexed by key ID.
// This allows the application to verify tokens offline without contacting Google's servers
// for each verification request.
func (o *OIDC) downloadJWKS() error {
	// Fetch the JWKS from the URI specified in the discovery document
	resp, err := http.Get(o.discovery.JWKSURI)
	if err != nil {
		return errors.Wrapf(err, "Failed to fetch JWKS from %s", o.discovery.JWKSURI)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			zap.L().Error("failed to close response body", zap.Error(err))
		}
	}()

	// Parse the JWKS into our structured format
	var jwks jwks
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return errors.Wrapf(err, "Failed to parse JWKS response")
	}

	// Convert each key to RSA public key and store in the map
	for _, key := range jwks.Keys {
		// Convert the modulus and exponent from base64url to *rsa.PublicKey
		n, err := base64.RawURLEncoding.DecodeString(key.N)
		if err != nil {
			return errors.Wrap(err, "failed to decode modulus")
		}

		e, err := base64.RawURLEncoding.DecodeString(key.E)
		if err != nil {
			return errors.Wrap(err, "failed to decode exponent")
		}

		// Convert the modulus to a big integer
		modulus := new(big.Int).SetBytes(n)

		// Convert the exponent to an integer
		var exponent int
		if len(e) < 4 {
			for i := range e {
				exponent = exponent<<8 + int(e[i])
			}
		} else {
			return errors.New("exponent too large")
		}

		// Create the RSA public key
		publicKey := &rsa.PublicKey{
			N: modulus,
			E: exponent,
		}

		// Store the public key in the map using the kid as the key
		o.publicKeys[key.Kid] = publicKey
	}

	return nil
}

// verifyBearerToken verifies the JWT token in the bearer token and returns whether it's valid and any error encountered
func (o *OIDC) verifyBearerToken(bearerToken string) (*jwt.Token, error) {
	if !strings.HasPrefix(strings.ToLower(bearerToken), "bearer ") {
		return nil, errors.New("bearer token must start with 'Bearer '")
	}

	// Remove prefix case-insensitive, trim whitespace
	idToken := strings.TrimPrefix(bearerToken, bearerToken[:7])
	idToken = strings.TrimSpace(idToken)

	return o.verifyToken(idToken)
}

// verifyToken verifies the JWT token and returns whether it's valid and any error encountered
// result is nil if its an invalid token and non-nil if its valid
func (o *OIDC) verifyToken(idToken string) (*jwt.Token, error) {
	if o == nil {
		return nil, errors.New("OIDC is not configured")
	}

	// Verify the token signature using JWKS
	token, err := jwt.Parse(idToken, func(token *jwt.Token) (any, error) {
		// Verify the signing method is what we expect
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get the key ID from the token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("kid header not found in token")
		}

		// Get the public key from our map
		publicKey, ok := o.publicKeys[kid]
		if !ok {
			return nil, errors.New("unable to find appropriate key")
		}

		return publicKey, nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token signature: %v", err)
	}

	// Get the claims from the token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("failed to get claims from token")
	}

	// Verify expiration
	exp, err := claims.GetExpirationTime()
	if err != nil {
		return nil, fmt.Errorf("failed to get expiration from claims: %v", err)
	}
	if time.Now().After(exp.Time) {
		return nil, errors.New("token expired")
	}

	// Verify issuer
	iss, err := claims.GetIssuer()
	if err != nil || iss != o.discovery.Issuer {
		return nil, fmt.Errorf("invalid token issuer: got %v, expected %v", iss, o.discovery.Issuer)
	}

	// Verify audience matches our client ID
	aud, err := claims.GetAudience()
	if err != nil || len(aud) == 0 || aud[0] != o.oauth2.ClientID {
		return nil, fmt.Errorf("invalid token audience: got %v, expected %v", aud, o.oauth2.ClientID)
	}

	return token, nil
}

// NewAuthMiddleware creates a middleware that enforces OIDC authentication
func NewAuthMiddleware(oidc *OIDC) (func(http.Handler) http.Handler, error) {
	return NewAuthMiddlewareForOIDC(oidc)
}

// NewAuthMiddlewareForOIDC allows our test to inject a mock OIDC
func NewAuthMiddlewareForOIDC(oidc *OIDC) (func(http.Handler) http.Handler, error) {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := zapr.NewLogger(zap.L())

			// Skip authentication for login page and OAuth2 endpoints
			if r.URL.Path == "/login" || strings.HasPrefix(r.URL.Path, OIDCPathPrefix+"/") {
				next.ServeHTTP(w, r)
				return
			}

			var token *jwt.Token
			var err error

			// Prefer bearer token over session cookie
			bearerToken := r.Header.Get("Authorization")
			if strings.HasPrefix(strings.ToLower(bearerToken), "bearer ") {
				token, err = oidc.verifyBearerToken(bearerToken)
			} else {
				// Fallback to session cookie
				cookie, cookieErr := r.Cookie(SessionCookieName)
				if cookieErr != nil {
					// No session cookie or bearer token, return 401 Unauthorized
					log.Error(cookieErr, "No session cookie or bearer token found")
					http.Error(w, "Unauthorized: No valid session", http.StatusUnauthorized)
					return
				}
				token, err = oidc.verifyToken(cookie.Value)
			}

			if token == nil {
				log.Error(err, "Token validation failed")
				// Return HTTP 401 and let the client handle the redirect to the login page
				http.Error(w, "Unauthorized: Invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := ContextWithIDToken(r.Context(), token)

			// Token is valid, proceed with the request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}, nil
}

// loginHandler handles the OAuth2 login flow
func (o *OIDC) LoginHandler(w http.ResponseWriter, r *http.Request) {
	log := logs.FromContext(r.Context())
	state, err := o.state.generateState()
	if err != nil {
		log.Error(err, "Failed to generate state")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	url := o.oauth2.AuthCodeURL(state.state, oauth2.S256ChallengeOption(state.verifier))
	if o.config.ForceApproval {
		url = o.oauth2.AuthCodeURL(state.state, oauth2.ApprovalForce)
	}
	log.Info("Redirecting to OAuth2 login", "url", url)
	http.Redirect(w, r, url, http.StatusFound)
}

// callbackHandler handles the OAuth2 callback
func (o *OIDC) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	log := zapr.NewLogger(zap.L())

	// Verify state
	stateKey := r.URL.Query().Get("state")
	verifier, ok := o.state.validateState(stateKey)
	if !ok {
		log.Error(nil, "Invalid state parameter")
		redirectWithError(w, r, "invalid_state", "Invalid state parameter")
		return
	}

	// Exchange code for token
	verifierOpt := oauth2.VerifierOption(verifier)

	code := r.URL.Query().Get("code")
	token, err := o.oauth2.Exchange(r.Context(), code, verifierOpt)
	if err != nil {
		log.Error(err, "Failed to exchange code for token")
		redirectWithError(w, r, "token_exchange_failed", "Failed to exchange code for token")
		return
	}

	// Create an OAuthToken protobuf message
	// This will be used to allow the client to potentially refresh the token.
	tokenPB := &agentv1.OAuthToken{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       timestamppb.New(token.Expiry),
		ExpiresIn:    token.ExpiresIn,
	}

	tokenPBJson, err := protojson.Marshal(tokenPB)
	if err != nil {
		log.Error(err, "Failed to marshal OAuthToken to JSON")
		redirectWithError(w, r, "token_marshal_failed", "Failed to marshal token")
		return
	}

	// Get the ID token from the response
	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		log.Error(nil, "No ID token in response")
		redirectWithError(w, r, "no_id_token", "No ID token in response")
		return
	}

	// We need to properly escape the token for use in a cookie
	// PathEscape properly encodes nested spaces.
	tokenEscaped := url.PathEscape(string(tokenPBJson))

	// Set the session cookie with the ID token
	http.SetCookie(w, &http.Cookie{
		Name:     SessionOAuthTokenName,
		Value:    tokenEscaped,
		Path:     "/",
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    idToken,
		Path:     "/",
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	// Redirect to the home page
	http.Redirect(w, r, "/", http.StatusFound)
}

// redirectWithError redirects to the login page with error information
func redirectWithError(w http.ResponseWriter, r *http.Request, errorCode, errorDescription string) {
	// Get any existing error parameters from the request
	existingError := r.URL.Query().Get("error")
	existingDescription := r.URL.Query().Get("error_description")

	// Use existing error parameters if they exist, otherwise use the provided ones
	if existingError == "" {
		existingError = errorCode
	}
	if existingDescription == "" {
		existingDescription = errorDescription
	}

	// Build the redirect URL with error parameters
	redirectURL := fmt.Sprintf("/login?error=%s&error_description=%s",
		url.QueryEscape(existingError),
		url.QueryEscape(existingDescription))

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// LogoutHandler handles the OAuth2 logout
func (o *OIDC) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Clear the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	// Redirect to the home page
	http.Redirect(w, r, "/", http.StatusFound)
}

type stateEntry struct {
	expiresAt time.Time
	// verifier is used for PKCE (Proof Key for Code Exchange)
	verifier string
	// state is the unique state string to be passed in the query argument
	state string
}

type stateManager struct {
	stateExpiration time.Duration
	states          map[string]stateEntry
	mu              sync.RWMutex
}

func newStateManager(stateExpiration time.Duration) *stateManager {
	return &stateManager{
		stateExpiration: stateExpiration,
		states:          make(map[string]stateEntry),
	}
}

// generateState generates a new cryptographically secure random state
func (sm *stateManager) generateState() (stateEntry, error) {
	b := make([]byte, stateLength)
	if _, err := rand.Read(b); err != nil {
		return stateEntry{}, errors.Wrap(err, "failed to generate random state")
	}
	state := base64.URLEncoding.EncodeToString(b)

	verifier, err := generateCodeVerifier(64)
	if err != nil {
		return stateEntry{}, errors.Wrap(err, "failed to generate code verifier")
	}
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.states[state] = stateEntry{
		state:     state,
		expiresAt: time.Now().Add(sm.stateExpiration),
		// Generate a code verifier for PKCE
		verifier: verifier,
	}

	return sm.states[state], nil
}

// validateState checks if a state is valid and removes it if it is
// returns a verifier if the state is valid, or false if it is not
func (sm *stateManager) validateState(state string) (string, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	entry, exists := sm.states[state]
	if !exists {
		return "", false
	}

	// Remove the state regardless of validity
	delete(sm.states, state)

	// Check if the state has expired
	return entry.verifier, time.Now().Before(entry.expiresAt)
}

// cleanupExpiredStates removes expired states from the map
func (sm *stateManager) cleanupExpiredStates() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for state, entry := range sm.states {
		if now.After(entry.expiresAt) {
			delete(sm.states, state)
		}
	}
}

// jwksKey represents a single key in the JWKS
type jwksKey struct {
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// jwks represents the JSON Web Key Set
type jwks struct {
	Keys []jwksKey `json:"keys"`
}

// openIDDiscovery is the struct for parsing the discovery document
type openIDDiscovery struct {
	Issuer                           string   `json:"issuer"`
	AuthURL                          string   `json:"authorization_endpoint"`
	TokenURL                         string   `json:"token_endpoint"`
	JWKSURI                          string   `json:"jwks_uri"`
	UserInfoURL                      string   `json:"userinfo_endpoint"`
	ScopesSupported                  []string `json:"scopes_supported"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
}

// OIDCProvider defines the interface for OIDC providers
type OIDCProvider interface {
	// GetOAuth2Config returns the OAuth2 configuration
	GetOAuth2Config() (*oauth2.Config, error)
	// GetDiscoveryURL returns the OpenID Connect discovery URL
	GetDiscoveryURL() string
}

// GoogleProvider implements OIDCProvider for Google OAuth2
type GoogleProvider struct {
	config *config.GoogleOIDCConfig
}

func NewGoogleProvider(cfg *config.GoogleOIDCConfig) *GoogleProvider {
	return &GoogleProvider{config: cfg}
}

func (p *GoogleProvider) GetOAuth2Config() (*oauth2.Config, error) {
	bytes, err := os.ReadFile(p.config.ClientCredentialsFile)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read client credentials file")
	}

	config, err := google.ConfigFromJSON(bytes, "openid", "email")
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create OAuth2 config")
	}
	return config, nil
}

func (p *GoogleProvider) GetDiscoveryURL() string {
	return p.config.GetDiscoveryURL()
}

func (p *GoogleProvider) ValidateDomainClaims(claims jwt.MapClaims, allowedDomains []string) error {
	hd, ok := claims["hd"].(string)
	if !ok {
		return errors.New("missing hosted domain claim")
	}
	if hd != "" {
		if !slices.Contains(allowedDomains, hd) {
			return fmt.Errorf("hosted domain %v not in allowed domains", hd)
		}
	} else {
		return errors.New("missing hosted domain claim")
	}
	return nil
}

// GenericProvider implements OIDCProvider for generic OIDC providers
type GenericProvider struct {
	config *config.GenericOIDCConfig
}

func NewGenericProvider(cfg *config.GenericOIDCConfig) *GenericProvider {
	return &GenericProvider{config: cfg}
}

func (p *GenericProvider) GetOAuth2Config() (*oauth2.Config, error) {
	return &oauth2.Config{
		ClientID:     p.config.ClientID,
		ClientSecret: p.config.ClientSecret,
		RedirectURL:  p.config.RedirectURL,
		Scopes:       p.config.Scopes,
	}, nil
}

func (p *GenericProvider) GetDiscoveryURL() string {
	return p.config.GetDiscoveryURL()
}

func (p *GenericProvider) ValidateDomainClaims(claims jwt.MapClaims, allowedDomains []string) error {
	email, ok := claims["email"].(string)
	if !ok {
		return errors.New("missing email claim")
	}

	if email == "" {
		return errors.New("empty email claim")
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return errors.New("invalid email format")
	}

	emailDomain := parts[1]
	if !slices.Contains(allowedDomains, emailDomain) {
		return fmt.Errorf("email domain not in allowed domains: %s", email)
	}

	return nil
}

// todo(sebastian): use everywhere?
// AuthContext encapsulates the Auth/IAM context for handlers.
type AuthContext struct {
	OIDC    *OIDC
	Checker Checker
	Role    string
}

func (a *AuthContext) AuthorizeRequest(ctx context.Context, req *streamv1.WebsocketRequest) error {
	log := logs.FromContextWithTrace(ctx)

	// Nil token is not fatal until authz denies access
	idToken, err := a.OIDC.verifyBearerToken(req.GetAuthorization())
	if err != nil {
		log.Info("Unauthenticated: ", "error", err)
		// TODO(jlewi): Should we be returning here?
	}

	principal, err := a.Checker.GetPrincipal(idToken)
	if err != nil {
		log.Error(err, "Could not extract principal from token")
		return ErrPrincipalExtraction
	}
	if a.Checker != nil {
		if ok := a.Checker.Check(principal, a.Role); !ok {
			log.Info("User does not have the required role", "principal", principal)
			return ErrRoleDenied
		}
	}
	return nil
}

// TestIDP is an IDP that we can use for testing.
// It can produce OIDC signed OIDC tokens that we can use to verify auth is working.

type TestIDP struct {
	privateKey *rsa.PrivateKey
	OIDC       *OIDC
	OIDCCfg    *config.OIDCConfig
}

func NewTestIDP(clientCredentialsFile string) (*TestIDP, error) {
	// Create a test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to generate RSA key")
	}

	// Create a test OIDC instance
	cfg := &config.OIDCConfig{
		Google: &config.GoogleOIDCConfig{
			ClientCredentialsFile: clientCredentialsFile,
			DiscoveryURL:          "https://accounts.google.com/.well-known/openid-configuration",
		},
	}
	oidc, err := NewOIDC(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create OIDC instance")
	}

	// Store the public key
	oidc.publicKeys["test-key"] = &privateKey.PublicKey

	return &TestIDP{
		privateKey: privateKey,
		OIDC:       oidc,
		OIDCCfg:    cfg,
	}, nil
}

func (idp *TestIDP) GenerateToken(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key"
	signedToken, err := token.SignedString(idp.privateKey)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to sign token")
	}

	return signedToken, nil
}

func generateCodeVerifier(length int) (string, error) {
	if length < 43 || length > 128 {
		return "", errors.New("code_verifier must be between 43 and 128 characters")
	}

	// Use URL-safe characters per RFC 7636
	// We'll generate random bytes and base64url encode them
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	verifier := base64.RawURLEncoding.EncodeToString(bytes)
	// Trim or pad to exact length if needed
	if len(verifier) > length {
		verifier = verifier[:length]
	}
	return verifier, nil
}
