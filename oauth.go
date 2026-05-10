package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
)

const (
	methodNotAllowedMessage = "Method not allowed"
	headerContentType       = "Content-Type"
	contentTypeForm         = "application/x-www-form-urlencoded"
	contentTypeJSON         = "application/json"
)

// OAuthServer implements a minimal OAuth 2.0 authorization server
// with dynamic client registration (DCR) for local MCP use.
type OAuthServer struct {
	mu           sync.Mutex
	authCodes    map[string]*authCodeEntry
	accessTokens map[string]*accessTokenEntry
	clients      map[string]*clientEntry
	issuerURL    string
	mux          http.Handler // lazy-initialized
}

type authCodeEntry struct {
	clientID      string
	codeChallenge string
	redirectURI   string
	createdAt     time.Time
}

type accessTokenEntry struct {
	clientID string
	scopes   []string
	expires  time.Time
}

type clientEntry struct {
	clientID     string
	clientSecret string
	redirectURIs []string
	createdAt    time.Time
}

// NewOAuthServer creates a new OAuth server for the given issuer URL.
func NewOAuthServer(issuerURL string) *OAuthServer {
	return &OAuthServer{
		authCodes:    make(map[string]*authCodeEntry),
		accessTokens: make(map[string]*accessTokenEntry),
		clients:      make(map[string]*clientEntry),
		issuerURL:    issuerURL,
	}
}

func (s *OAuthServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.mux == nil {
		mux := http.NewServeMux()
		mux.HandleFunc("/.well-known/oauth-protected-resource", s.handleProtectedResourceMeta)
		mux.HandleFunc("/.well-known/oauth-authorization-server", s.handleAuthServerMeta)
		mux.HandleFunc("/register", s.handleRegistration)
		mux.HandleFunc("/authorize", s.handleAuthorize)
		mux.HandleFunc("/token", s.handleToken)
		s.mux = mux
	}

	s.mux.ServeHTTP(w, r)
}

// --- Protected Resource Metadata (RFC 9728) ---

type protectedResourceMeta struct {
	Resource               string   `json:"resource"`
	AuthorizationServers   []string `json:"authorization_servers"`
	BearerMethodsSupported []string `json:"bearer_methods_supported,omitempty"`
	ResourceName           string   `json:"resource_name,omitempty"`
}

func (s *OAuthServer) handleProtectedResourceMeta(w http.ResponseWriter, r *http.Request) {
	meta := protectedResourceMeta{
		Resource:               s.issuerURL,
		AuthorizationServers:   []string{s.issuerURL},
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "Scholar Aggregator MCP Server",
	}
	writeJSON(w, http.StatusOK, meta)
}

// --- Authorization Server Metadata (RFC 8414) ---

type authServerMeta struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint,omitempty"`
	ScopesSupported                   []string `json:"scopes_supported,omitempty"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported,omitempty"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported,omitempty"`
}

func (s *OAuthServer) handleAuthServerMeta(w http.ResponseWriter, r *http.Request) {
	meta := authServerMeta{
		Issuer:                            s.issuerURL,
		AuthorizationEndpoint:             s.issuerURL + "/authorize",
		TokenEndpoint:                     s.issuerURL + "/token",
		RegistrationEndpoint:              s.issuerURL + "/register",
		ScopesSupported:                   []string{"mcp"},
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_post", "none"},
		CodeChallengeMethodsSupported:     []string{"S256"},
	}
	writeJSON(w, http.StatusOK, meta)
}

// --- Dynamic Client Registration (RFC 7591) ---

type registrationRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
}

type registrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientSecretExpiresAt   int64    `json:"client_secret_expires_at,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

func (s *OAuthServer) handleRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, methodNotAllowedMessage, http.StatusMethodNotAllowed)
		return
	}

	var req registrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": err.Error()})
		return
	}
	if len(req.RedirectURIs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "redirect_uris is required"})
		return
	}

	clientID := "mcp-client-" + randomHex(16)
	clientSecret := randomHex(32)
	now := time.Now()

	s.mu.Lock()
	s.clients[clientID] = &clientEntry{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURIs: req.RedirectURIs,
		createdAt:    now,
	}
	s.mu.Unlock()

	log.Printf("[OAUTH] Registered client: %s (name: %s)", clientID, req.ClientName)

	resp := registrationResponse{
		ClientID:                clientID,
		ClientSecret:            clientSecret,
		ClientIDIssuedAt:        now.Unix(),
		ClientSecretExpiresAt:   now.Add(365 * 24 * time.Hour).Unix(),
		RedirectURIs:            req.RedirectURIs,
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
	}
	writeJSON(w, http.StatusCreated, resp)
}

// --- Authorization Endpoint ---

func (s *OAuthServer) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, methodNotAllowedMessage, http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	responseType := q.Get("response_type")
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	codeChallenge := q.Get("code_challenge")
	codeChallengeMethod := q.Get("code_challenge_method")
	state := q.Get("state")

	if responseType != "code" {
		redirectWithError(w, r, redirectURI, state, "unsupported_response_type", "only 'code' is supported")
		return
	}

	s.mu.Lock()
	client := s.clients[clientID]
	s.mu.Unlock()
	if client == nil {
		redirectWithError(w, r, redirectURI, state, "unauthorized_client", "unknown client")
		return
	}

	if codeChallengeMethod != "S256" || codeChallenge == "" {
		redirectWithError(w, r, redirectURI, state, "invalid_request", "PKCE S256 required")
		return
	}

	redirectURIOk := false
	for _, uri := range client.redirectURIs {
		if uri == redirectURI {
			redirectURIOk = true
			break
		}
	}
	if !redirectURIOk {
		redirectWithError(w, r, redirectURI, state, "invalid_request", "redirect_uri not registered")
		return
	}

	// Auto-approve for local dev
	authCode := "auth-code-" + randomHex(24)

	s.mu.Lock()
	s.authCodes[authCode] = &authCodeEntry{
		clientID:      clientID,
		codeChallenge: codeChallenge,
		redirectURI:   redirectURI,
		createdAt:     time.Now(),
	}
	s.mu.Unlock()

	log.Printf("[OAUTH] Issued auth code for client %s", clientID)

	redirectURL, _ := url.Parse(redirectURI)
	params := redirectURL.Query()
	params.Set("code", authCode)
	if state != "" {
		params.Set("state", state)
	}
	redirectURL.RawQuery = params.Encode()
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// --- Token Endpoint ---

type tokenRequest struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code,omitempty"`
	CodeVerifier string `json:"code_verifier,omitempty"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

func (s *OAuthServer) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, methodNotAllowedMessage, http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get(headerContentType)
	if !isSupportedTokenContentType(contentType) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "Content-Type must be application/x-www-form-urlencoded or application/json"})
		return
	}

	req, err := parseTokenRequest(r, contentType)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	status, clientErr := s.validateTokenClient(req)
	if clientErr != nil {
		writeJSON(w, status, map[string]string{"error": "invalid_client"})
		return
	}

	switch req.GrantType {
	case "authorization_code":
		s.handleAuthorizationCodeGrant(w, req)
	case "refresh_token":
		s.handleRefreshTokenGrant(w, req)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_grant_type"})
	}
}

func isSupportedTokenContentType(contentType string) bool {
	return strings.HasPrefix(contentType, contentTypeForm) || strings.HasPrefix(contentType, contentTypeJSON)
}

func parseTokenRequest(r *http.Request, contentType string) (tokenRequest, error) {
	if strings.HasPrefix(contentType, contentTypeJSON) {
		var req tokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return tokenRequest{}, err
		}
		return req, nil
	}

	if err := r.ParseForm(); err != nil {
		return tokenRequest{}, err
	}

	return tokenRequest{
		GrantType:    r.Form.Get("grant_type"),
		Code:         r.Form.Get("code"),
		CodeVerifier: r.Form.Get("code_verifier"),
		RedirectURI:  r.Form.Get("redirect_uri"),
		ClientID:     r.Form.Get("client_id"),
		ClientSecret: r.Form.Get("client_secret"),
		RefreshToken: r.Form.Get("refresh_token"),
	}, nil
}

func (s *OAuthServer) validateTokenClient(req tokenRequest) (int, error) {
	if req.ClientID == "" {
		return http.StatusOK, nil
	}

	s.mu.Lock()
	client := s.clients[req.ClientID]
	s.mu.Unlock()
	if client == nil {
		return http.StatusBadRequest, fmt.Errorf("unknown client")
	}
	if client.clientSecret != "" && req.ClientSecret != client.clientSecret {
		return http.StatusUnauthorized, fmt.Errorf("invalid client secret")
	}

	return http.StatusOK, nil
}

func (s *OAuthServer) handleAuthorizationCodeGrant(w http.ResponseWriter, req tokenRequest) {
	if req.Code == "" || req.CodeVerifier == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "code and code_verifier required"})
		return
	}

	s.mu.Lock()
	entry, ok := s.authCodes[req.Code]
	if ok {
		delete(s.authCodes, req.Code)
	}
	s.mu.Unlock()

	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "unknown or expired auth code"})
		return
	}

	if time.Since(entry.createdAt) > 5*time.Minute {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "auth code expired"})
		return
	}

	// PKCE verification
	hasher := sha256.New()
	hasher.Write([]byte(req.CodeVerifier))
	calculatedChallenge := base64.RawURLEncoding.EncodeToString(hasher.Sum(nil))
	if calculatedChallenge != entry.codeChallenge {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "code_verifier mismatch"})
		return
	}

	if req.RedirectURI != "" && req.RedirectURI != entry.redirectURI {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "redirect_uri mismatch"})
		return
	}

	accessToken := "mcp-token-" + randomHex(32)
	refreshToken := "mcp-refresh-" + randomHex(32)
	expires := time.Now().Add(1 * time.Hour)

	s.mu.Lock()
	s.accessTokens[accessToken] = &accessTokenEntry{
		clientID: entry.clientID,
		scopes:   []string{"mcp"},
		expires:  expires,
	}
	s.mu.Unlock()

	log.Printf("[OAUTH] Issued access token for client %s (expires: %s)", entry.clientID, expires.Format(time.RFC3339))

	writeJSON(w, http.StatusOK, tokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: refreshToken,
		Scope:        "mcp",
	})
}

func (s *OAuthServer) handleRefreshTokenGrant(w http.ResponseWriter, req tokenRequest) {
	// For simplicity, just issue a new access token
	accessToken := "mcp-token-" + randomHex(32)
	refreshToken := "mcp-refresh-" + randomHex(32)
	expires := time.Now().Add(1 * time.Hour)

	s.mu.Lock()
	s.accessTokens[accessToken] = &accessTokenEntry{
		clientID: req.ClientID,
		scopes:   []string{"mcp"},
		expires:  expires,
	}
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, tokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: refreshToken,
		Scope:        "mcp",
	})
}

// --- Token Verification ---

// TokenVerifier returns an auth.TokenVerifier that validates tokens from this OAuth server.
func (s *OAuthServer) TokenVerifier() auth.TokenVerifier {
	return func(ctx context.Context, token string) (*auth.TokenInfo, error) {
		s.mu.Lock()
		entry, ok := s.accessTokens[token]
		s.mu.Unlock()

		if !ok {
			return nil, fmt.Errorf("%w: unknown token", auth.ErrInvalidToken)
		}
		if time.Now().After(entry.expires) {
			return nil, fmt.Errorf("%w: token expired", auth.ErrInvalidToken)
		}

		return &auth.TokenInfo{
			Scopes:     entry.scopes,
			Expiration: entry.expires,
		}, nil
	}
}

// --- Bearer Middleware for MCP endpoint ---

func (s *OAuthServer) BearerMiddleware(mcpHandler http.Handler) http.Handler {
	verifier := s.TokenVerifier()
	return auth.RequireBearerToken(verifier, &auth.RequireBearerTokenOptions{
		ResourceMetadataURL: s.issuerURL + "/.well-known/oauth-protected-resource",
		Scopes:              []string{"mcp"},
	})(mcpHandler)
}

// --- Helpers ---

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("rand.Read failed: %v", err))
	}
	return hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func redirectWithError(w http.ResponseWriter, r *http.Request, redirectURI, state, errCode, desc string) {
	if redirectURI == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": errCode, "error_description": desc})
		return
	}
	u, _ := url.Parse(redirectURI)
	params := u.Query()
	params.Set("error", errCode)
	params.Set("error_description", desc)
	if state != "" {
		params.Set("state", state)
	}
	u.RawQuery = params.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}
