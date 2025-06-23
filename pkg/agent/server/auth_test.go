package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/runmedev/runme/v3/pkg/agent/config"
	"github.com/runmedev/runme/v3/pkg/agent/iam"
)

type DenyAllChecker struct{}

func (d *DenyAllChecker) Check(principal string, role string) bool {
	return false
}

func (d *DenyAllChecker) GetPrincipal(idToken *jwt.Token) (string, error) {
	claims, _ := idToken.Claims.(jwt.MapClaims)
	email, _ := claims["email"].(string)
	return email, nil
}

func TestOIDC_UnauthenticatedRoutes_NoSession(t *testing.T) {
	// This test verifies that if there is no session token the user gets back an Unuauthorized error
	// Create test OIDC config
	oidcConfig := &config.OIDCConfig{
		Google: &config.GoogleOIDCConfig{
			ClientCredentialsFile: "../iam/testdata/google-client-dummy.json",
			DiscoveryURL:          "https://accounts.google.com/.well-known/openid-configuration",
		},
	}

	// Create server config
	serverConfig := &config.AssistantServerConfig{
		CorsOrigins: []string{"http://localhost:3000"},
		OIDC:        oidcConfig,
	}

	// Create OIDC instance for tests
	var oidc *iam.OIDC
	var err error
	oidc, err = iam.NewOIDC(oidcConfig)
	if err != nil {
		t.Fatalf("Failed to create OIDC instance: %v", err)
	}

	// Create auth mux
	mux, err := NewAuthMux(serverConfig, oidc)
	if err != nil {
		t.Fatalf("Failed to create auth mux: %v", err)
	}

	// Register auth routes
	if err := RegisterAuthRoutes(oidc, mux); err != nil {
		t.Fatalf("Failed to register auth routes: %v", err)
	}

	// Register test routes
	mux.Handle("/public", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	mux.HandleProtected("/protected", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), &DenyAllChecker{}, "test-role")

	// Test public route
	req := httptest.NewRequest("GET", "/public", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Test protected route
	req = httptest.NewRequest("GET", "/protected", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	msg, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(msg) != "Unauthorized: No valid session\n" {
		t.Errorf("Expected response body 'Unauthorized', got '%s'", msg)
	}
}

func TestOIDC_AccessForbidden(t *testing.T) {
	// This test verifies that if there is a session token but the user is denied by the IAMPolicy they get
	// a forbidden error
	idp, err := iam.NewTestIDP("../iam/testdata/google-client-dummy.json")
	if err != nil {
		t.Fatalf("Failed to create test IDP: %v", err)
	}

	// Create server config
	serverConfig := &config.AssistantServerConfig{
		CorsOrigins: []string{"http://localhost:3000"},
		OIDC:        idp.OIDCCfg,
	}

	// Create OIDC instance for tests
	var oidc *iam.OIDC
	if idp.OIDCCfg != nil {
		var err error
		oidc, err = iam.NewOIDC(idp.OIDCCfg)
		if err != nil {
			t.Fatalf("Failed to create OIDC instance: %v", err)
		}
	}

	// Create auth mux
	mux, err := NewAuthMux(serverConfig, oidc)
	if err != nil {
		t.Fatalf("Failed to create auth mux: %v", err)
	}
	// This is a bit of a hack to allow us to inject our test IDP.
	authMiddleWare, err := iam.NewAuthMiddlewareForOIDC(idp.OIDC)
	if err != nil {
		t.Fatalf("Failed to create auth middleware: %v", err)
	}
	mux.authMiddleware = authMiddleWare
	if err != nil {
		t.Fatalf("Failed to create auth mux: %v", err)
	}

	// Register auth routes
	if err := RegisterAuthRoutes(idp.OIDC, mux); err != nil {
		t.Fatalf("Failed to register auth routes: %v", err)
	}

	// Register test routes
	mux.Handle("/public", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// We use a DenyAll to make sure we get back unauthorized
	mux.HandleProtected("/protected", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), &DenyAllChecker{}, "test-role")

	signedToken, err := idp.GenerateToken(jwt.MapClaims{
		"iss":   "https://accounts.google.com",
		"aud":   "dummy-client-id",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"hd":    "example.com",
		"email": "john@acme.com",
	})
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	sessionCookie := &http.Cookie{
		Name:  iam.SessionCookieName,
		Value: signedToken,
	}

	// Test protected route
	req := httptest.NewRequest("GET", "/protected", nil)
	rec := httptest.NewRecorder()
	req.AddCookie(sessionCookie)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status code %d, got %d", http.StatusForbidden, rec.Code)
	}

	msg, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(msg) != "Forbidden: user john@acme.com doesn't have role test-role\n" {
		t.Errorf("Expected response body 'Unauthorized', got '%s'", msg)
	}
}

func TestOIDC_TokenHierarchy(t *testing.T) {
	idp, err := iam.NewTestIDP("../iam/testdata/google-client-dummy.json")
	if err != nil {
		t.Fatalf("Failed to create test IDP: %v", err)
	}

	// Create server config
	serverConfig := &config.AssistantServerConfig{
		CorsOrigins: []string{"http://localhost:3000"},
		OIDC:        idp.OIDCCfg,
	}

	// Create OIDC instance for tests
	var oidc *iam.OIDC
	if idp.OIDCCfg != nil {
		var err error
		oidc, err = iam.NewOIDC(idp.OIDCCfg)
		if err != nil {
			t.Fatalf("Failed to create OIDC instance: %v", err)
		}
	}

	// Create auth mux
	mux, err := NewAuthMux(serverConfig, oidc)
	if err != nil {
		t.Fatalf("Failed to create auth mux: %v", err)
	}
	// Inject our test OIDC
	authMiddleware, err := iam.NewAuthMiddlewareForOIDC(idp.OIDC)
	if err != nil {
		t.Fatalf("Failed to create auth middleware: %v", err)
	}
	mux.authMiddleware = authMiddleware

	// Register auth routes
	if err := RegisterAuthRoutes(idp.OIDC, mux); err != nil {
		t.Fatalf("Failed to register auth routes: %v", err)
	}

	// Register a protected route that returns 200 OK
	mux.HandleProtected("/protected", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}), &DenyAllChecker{}, "test-role")

	// Generate three tokens for clarity
	claims := jwt.MapClaims{
		"iss":   "https://accounts.google.com",
		"aud":   "dummy-client-id",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"hd":    "example.com",
		"email": "john@acme.com",
	}
	token, err := idp.GenerateToken(claims)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	const AuthedButForbidden = http.StatusForbidden

	t.Run("Header takes precedence over query and cookie", func(t *testing.T) {
		authParam := url.QueryEscape("authorization=Bearer invalid")
		req := httptest.NewRequest("GET", "/protected?"+authParam, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.AddCookie(&http.Cookie{Name: iam.SessionCookieName, Value: "invalid"})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != AuthedButForbidden {
			t.Errorf("Expected status %d, got %d", AuthedButForbidden, rec.Code)
		}
	})

	t.Run("Cookie is used if no header or query param", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{Name: iam.SessionCookieName, Value: token})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != AuthedButForbidden {
			t.Errorf("Expected status %d, got %d", AuthedButForbidden, rec.Code)
		}
	})

	t.Run("Unauthorized if no token anywhere", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}
