package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/runmedev/runme/v3/pkg/agent/logs"
)

const (
	jupyterServersRoute = "/v1/jupyter/servers"
	jupyterConfigDir    = "jupyter"
)

type jupyterProxyHandler struct {
	registry   *jupyterServerRegistry
	httpClient *http.Client
	upgrader   websocket.Upgrader
}

func newJupyterProxyHandler(configDir string) (*jupyterProxyHandler, error) {
	registry, err := newJupyterServerRegistry(configDir)
	if err != nil {
		return nil, err
	}

	return &jupyterProxyHandler{
		registry: registry,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(_ *http.Request) bool {
				return true
			},
		},
	}, nil
}

func (h *jupyterProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case path == jupyterServersRoute:
		if r.Method != http.MethodGet {
			writeHTTPError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.handleListServers(w, r)
		return
	case strings.HasPrefix(path, jupyterServersRoute+"/"):
		h.handleServerSubroute(w, r)
		return
	default:
		writeHTTPError(w, http.StatusNotFound, "not found")
		return
	}
}

func (h *jupyterProxyHandler) handleListServers(w http.ResponseWriter, r *http.Request) {
	records := h.registry.List()
	writeJSON(w, http.StatusOK, records)
}

func (h *jupyterProxyHandler) handleServerSubroute(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, jupyterServersRoute+"/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 {
		writeHTTPError(w, http.StatusNotFound, "not found")
		return
	}

	serverName, err := url.PathUnescape(parts[0])
	if err != nil || strings.TrimSpace(serverName) == "" {
		writeHTTPError(w, http.StatusBadRequest, "invalid server name")
		return
	}

	if parts[1] != "kernels" {
		writeHTTPError(w, http.StatusNotFound, "not found")
		return
	}

	switch {
	case len(parts) == 2:
		switch r.Method {
		case http.MethodGet, http.MethodPost:
			h.forwardHTTP(w, r, serverName, "/api/kernels")
		default:
			writeHTTPError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	case len(parts) == 3:
		kernelSegment, err := url.PathUnescape(parts[2])
		if err != nil || strings.TrimSpace(kernelSegment) == "" {
			writeHTTPError(w, http.StatusBadRequest, "invalid kernel id")
			return
		}
		switch {
		case strings.HasSuffix(kernelSegment, ":interrupt"):
			if r.Method != http.MethodPost {
				writeHTTPError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			kernelID := strings.TrimSuffix(kernelSegment, ":interrupt")
			if strings.TrimSpace(kernelID) == "" {
				writeHTTPError(w, http.StatusBadRequest, "invalid kernel id")
				return
			}
			h.forwardHTTP(w, r, serverName, "/api/kernels/"+url.PathEscape(kernelID)+"/interrupt")
			return
		case strings.HasSuffix(kernelSegment, ":restart"):
			if r.Method != http.MethodPost {
				writeHTTPError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			kernelID := strings.TrimSuffix(kernelSegment, ":restart")
			if strings.TrimSpace(kernelID) == "" {
				writeHTTPError(w, http.StatusBadRequest, "invalid kernel id")
				return
			}
			h.forwardHTTP(w, r, serverName, "/api/kernels/"+url.PathEscape(kernelID)+"/restart")
			return
		default:
			if r.Method != http.MethodDelete {
				writeHTTPError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			h.forwardHTTP(w, r, serverName, "/api/kernels/"+url.PathEscape(kernelSegment))
			return
		}
	case len(parts) == 4 && parts[3] == "channels":
		kernelID, err := url.PathUnescape(parts[2])
		if err != nil || strings.TrimSpace(kernelID) == "" {
			writeHTTPError(w, http.StatusBadRequest, "invalid kernel id")
			return
		}
		if r.Method != http.MethodGet {
			writeHTTPError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.forwardChannelsWebSocket(w, r, serverName, kernelID)
		return
	default:
		writeHTTPError(w, http.StatusNotFound, "not found")
		return
	}
}

func (h *jupyterProxyHandler) forwardHTTP(w http.ResponseWriter, r *http.Request, serverName, upstreamPath string) {
	ctx := r.Context()
	log := logs.FromContext(ctx)

	server, err := h.registry.Resolve(serverName)
	if err != nil {
		writeHTTPError(w, http.StatusNotFound, err.Error())
		return
	}
	if strings.TrimSpace(server.Token) == "" {
		writeHTTPError(w, http.StatusBadRequest, "server is missing jupyter token")
		return
	}

	upstreamURL, err := buildUpstreamURL(server.BaseURL, upstreamPath, r.URL.RawQuery)
	if err != nil {
		writeHTTPError(w, http.StatusInternalServerError, "failed to construct upstream url")
		return
	}
	setUpstreamAuthToken(upstreamURL, server.Token)

	var body io.Reader
	if r.Body != nil {
		body = r.Body
	}
	upstreamReq, err := http.NewRequestWithContext(ctx, r.Method, upstreamURL.String(), body)
	if err != nil {
		writeHTTPError(w, http.StatusInternalServerError, "failed to build upstream request")
		return
	}
	copyProxyRequestHeaders(upstreamReq.Header, r.Header)
	upstreamReq.Header.Set("Authorization", "token "+server.Token)

	resp, err := h.httpClient.Do(upstreamReq)
	if err != nil {
		log.Error(err, "Jupyter upstream request failed", "server", serverName, "url", upstreamURL.String())
		writeHTTPError(w, http.StatusBadGateway, "failed to contact jupyter server")
		return
	}
	defer resp.Body.Close()

	copyProxyResponseHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Error(err, "failed to copy jupyter response body")
	}
}

func (h *jupyterProxyHandler) forwardChannelsWebSocket(w http.ResponseWriter, r *http.Request, serverName, kernelID string) {
	ctx := r.Context()
	log := logs.FromContext(ctx)

	server, err := h.registry.Resolve(serverName)
	if err != nil {
		writeHTTPError(w, http.StatusNotFound, err.Error())
		return
	}
	if strings.TrimSpace(server.Token) == "" {
		writeHTTPError(w, http.StatusBadRequest, "server is missing jupyter token")
		return
	}

	upstreamURL, err := buildUpstreamURL(
		server.BaseURL,
		"/api/kernels/"+url.PathEscape(kernelID)+"/channels",
		sanitizeClientQueryForUpstream(r.URL.RawQuery),
	)
	if err != nil {
		writeHTTPError(w, http.StatusInternalServerError, "failed to construct upstream url")
		return
	}
	setUpstreamAuthToken(upstreamURL, server.Token)
	switch upstreamURL.Scheme {
	case "http":
		upstreamURL.Scheme = "ws"
	case "https":
		upstreamURL.Scheme = "wss"
	default:
		writeHTTPError(w, http.StatusInternalServerError, "invalid upstream url scheme")
		return
	}

	upstreamHeader := http.Header{}
	copyWebSocketDialHeaders(upstreamHeader, r.Header)
	upstreamHeader.Set("Authorization", "token "+server.Token)

	upstreamConn, upstreamResp, err := websocket.DefaultDialer.DialContext(ctx, upstreamURL.String(), upstreamHeader)
	if err != nil {
		statusCode := http.StatusBadGateway
		if upstreamResp != nil && upstreamResp.StatusCode > 0 {
			statusCode = upstreamResp.StatusCode
		}
		log.Error(err, "failed to connect to jupyter channels websocket", "server", serverName, "kernel_id", kernelID)
		writeHTTPError(w, statusCode, "failed to connect to jupyter kernel channels")
		return
	}
	defer upstreamConn.Close()

	clientConn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err, "failed to upgrade client websocket for jupyter channels")
		return
	}
	defer clientConn.Close()

	errCh := make(chan error, 2)
	go bridgeWebSocketMessages(clientConn, upstreamConn, "upstream_to_client", errCh)
	go bridgeWebSocketMessages(upstreamConn, clientConn, "client_to_upstream", errCh)

	if err := <-errCh; err != nil {
		logJupyterWebSocketBridgeClose(log, serverName, kernelID, err)
	}
}

type jupyterServerRegistry struct {
	configDir string
	mu        sync.RWMutex
	servers   map[string]jupyterServerRecord
}

type jupyterServerRecord struct {
	Name    string
	Runner  string
	BaseURL *url.URL
	Token   string
}

type jupyterServerFile struct {
	BaseURLSnake string `json:"base_url"`
	BaseURLCamel string `json:"baseUrl"`
	Token        string `json:"token"`
	Runner       string `json:"runner"`
}

type jupyterServerPublicRecord struct {
	Name     string `json:"name"`
	Runner   string `json:"runner"`
	BaseURL  string `json:"base_url"`
	HasToken bool   `json:"has_token"`
}

func newJupyterServerRegistry(configDir string) (*jupyterServerRegistry, error) {
	if strings.TrimSpace(configDir) == "" {
		return nil, errors.New("config directory is required")
	}
	registry := &jupyterServerRegistry{
		configDir: filepath.Join(configDir, jupyterConfigDir),
		servers:   make(map[string]jupyterServerRecord),
	}
	if err := registry.reload(); err != nil {
		return nil, err
	}
	return registry, nil
}

func (r *jupyterServerRegistry) List() []jupyterServerPublicRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	records := make([]jupyterServerPublicRecord, 0, len(r.servers))
	for _, server := range r.servers {
		records = append(records, jupyterServerPublicRecord{
			Name:     server.Name,
			Runner:   server.Runner,
			BaseURL:  server.BaseURL.String(),
			HasToken: strings.TrimSpace(server.Token) != "",
		})
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Name < records[j].Name
	})
	return records
}

func (r *jupyterServerRegistry) Resolve(name string) (jupyterServerRecord, error) {
	if strings.TrimSpace(name) == "" {
		return jupyterServerRecord{}, errors.New("server name is required")
	}

	r.mu.RLock()
	server, ok := r.servers[name]
	r.mu.RUnlock()
	if ok {
		return server, nil
	}

	if err := r.reload(); err != nil {
		return jupyterServerRecord{}, err
	}

	r.mu.RLock()
	server, ok = r.servers[name]
	r.mu.RUnlock()
	if !ok {
		return jupyterServerRecord{}, fmt.Errorf("jupyter server %q not found", name)
	}
	return server, nil
}

func (r *jupyterServerRegistry) reload() error {
	entries, err := os.ReadDir(r.configDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			r.mu.Lock()
			r.servers = make(map[string]jupyterServerRecord)
			r.mu.Unlock()
			return nil
		}
		return err
	}

	next := make(map[string]jupyterServerRecord)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		serverName := strings.TrimSuffix(entry.Name(), ".json")
		fullPath := filepath.Join(r.configDir, entry.Name())
		server, err := loadJupyterServerFile(serverName, fullPath)
		if err != nil {
			zap.L().Warn(
				"Skipping invalid jupyter server config",
				zap.String("path", fullPath),
				zap.Error(err),
			)
			continue
		}
		next[serverName] = server
	}

	r.mu.Lock()
	r.servers = next
	r.mu.Unlock()
	return nil
}

func loadJupyterServerFile(name, path string) (jupyterServerRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return jupyterServerRecord{}, err
	}

	var payload jupyterServerFile
	if err := json.Unmarshal(raw, &payload); err != nil {
		return jupyterServerRecord{}, err
	}

	baseURLRaw := strings.TrimSpace(payload.BaseURLSnake)
	if baseURLRaw == "" {
		baseURLRaw = strings.TrimSpace(payload.BaseURLCamel)
	}
	if baseURLRaw == "" {
		return jupyterServerRecord{}, errors.New("base_url is required")
	}

	parsed, err := url.Parse(baseURLRaw)
	if err != nil {
		return jupyterServerRecord{}, errors.New("base_url is invalid")
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return jupyterServerRecord{}, errors.New("base_url must include scheme and host")
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	if !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}

	return jupyterServerRecord{
		Name:    name,
		Runner:  strings.TrimSpace(payload.Runner),
		BaseURL: parsed,
		Token:   strings.TrimSpace(payload.Token),
	}, nil
}

func buildUpstreamURL(base *url.URL, upstreamPath, rawQuery string) (*url.URL, error) {
	if base == nil {
		return nil, errors.New("base url is required")
	}
	if strings.TrimSpace(upstreamPath) == "" {
		return nil, errors.New("upstream path is required")
	}
	ref := &url.URL{
		Path:     upstreamPath,
		RawQuery: rawQuery,
	}
	return base.ResolveReference(ref), nil
}

func sanitizeClientQueryForUpstream(rawQuery string) string {
	if strings.TrimSpace(rawQuery) == "" {
		return ""
	}

	queryValues, err := url.ParseQuery(rawQuery)
	if err != nil {
		return rawQuery
	}

	for key := range queryValues {
		if strings.EqualFold(strings.TrimSpace(key), "authorization") {
			delete(queryValues, key)
		}
	}

	return queryValues.Encode()
}

func setUpstreamAuthToken(upstreamURL *url.URL, token string) {
	if upstreamURL == nil {
		return
	}
	trimmedToken := strings.TrimSpace(token)
	if trimmedToken == "" {
		return
	}
	query := upstreamURL.Query()
	query.Set("token", trimmedToken)
	upstreamURL.RawQuery = query.Encode()
}

func copyProxyRequestHeaders(dst, src http.Header) {
	for k, values := range src {
		if isHopByHopHeader(k) || isRestrictedProxyRequestHeader(k) {
			continue
		}
		for _, value := range values {
			dst.Add(k, value)
		}
	}
}

func copyProxyResponseHeaders(dst, src http.Header) {
	for k, values := range src {
		if isHopByHopHeader(k) || isCORSResponseHeader(k) {
			continue
		}
		for _, value := range values {
			dst.Add(k, value)
		}
	}
}

func copyWebSocketDialHeaders(dst, src http.Header) {
	const secWebSocketProtocol = "Sec-WebSocket-Protocol"
	for _, value := range src.Values(secWebSocketProtocol) {
		dst.Add(secWebSocketProtocol, value)
	}
}

func isHopByHopHeader(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "connection",
		"keep-alive",
		"proxy-authenticate",
		"proxy-authorization",
		"te",
		"trailers",
		"transfer-encoding",
		"upgrade":
		return true
	default:
		return false
	}
}

func isCORSResponseHeader(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "access-control-allow-origin",
		"access-control-allow-credentials",
		"access-control-allow-methods",
		"access-control-allow-headers",
		"access-control-expose-headers",
		"access-control-max-age":
		return true
	default:
		return false
	}
}

func isRestrictedProxyRequestHeader(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "authorization",
		"origin",
		"referer",
		"cookie",
		"x-xsrftoken":
		return true
	default:
		return false
	}
}

type webSocketBridgeError struct {
	err         error
	direction   string
	operation   string
	srcRemote   string
	dstRemote   string
	messageType int
}

func (e *webSocketBridgeError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *webSocketBridgeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func remoteAddr(conn *websocket.Conn) string {
	if conn == nil || conn.UnderlyingConn() == nil {
		return ""
	}
	return conn.UnderlyingConn().RemoteAddr().String()
}

func logJupyterWebSocketBridgeClose(log logr.Logger, serverName, kernelID string, err error) {
	if err == nil {
		return
	}

	fields := []any{
		"server", serverName,
		"kernel_id", kernelID,
		"error", err.Error(),
	}

	var bridgeErr *webSocketBridgeError
	if errors.As(err, &bridgeErr) {
		fields = append(
			fields,
			"direction", bridgeErr.direction,
			"operation", bridgeErr.operation,
			"src_remote_addr", bridgeErr.srcRemote,
			"dst_remote_addr", bridgeErr.dstRemote,
		)
		if bridgeErr.messageType > 0 {
			fields = append(fields, "message_type", bridgeErr.messageType)
		}
	}

	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) {
		fields = append(
			fields,
			"is_websocket_close", true,
			"close_code", closeErr.Code,
			"close_text", closeErr.Text,
		)
	} else {
		fields = append(fields, "is_websocket_close", false)
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		fields = append(
			fields,
			"is_timeout", netErr.Timeout(),
			"is_temporary", netErr.Temporary(),
		)
	}

	log.Info("jupyter channels websocket bridge closed", fields...)
}

func bridgeWebSocketMessages(dst, src *websocket.Conn, direction string, errCh chan<- error) {
	for {
		msgType, payload, err := src.ReadMessage()
		if err != nil {
			errCh <- &webSocketBridgeError{
				err:       err,
				direction: direction,
				operation: "read",
				srcRemote: remoteAddr(src),
				dstRemote: remoteAddr(dst),
			}
			return
		}
		if err := dst.WriteMessage(msgType, payload); err != nil {
			errCh <- &webSocketBridgeError{
				err:         err,
				direction:   direction,
				operation:   "write",
				srcRemote:   remoteAddr(src),
				dstRemote:   remoteAddr(dst),
				messageType: msgType,
			}
			return
		}
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		writeHTTPError(w, http.StatusInternalServerError, "failed to serialize response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(data)
}

func writeHTTPError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]any{
		"error": message,
	})
}

var _ http.Handler = (*jupyterProxyHandler)(nil)
