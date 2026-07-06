package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const apiTestConfig = `---
default:
  provider: gcp
  project: test-project
  env:
    TEST_VAR:
      value: hello
`

const testPort = 11911

func TestFindAvailablePort(t *testing.T) {
	port, err := findAvailablePort(11911, 10)
	require.NoError(t, err)
	require.GreaterOrEqual(t, port, 11911)
	require.LessOrEqual(t, port, 11920)
}

func TestAPIListEnvironments(t *testing.T) {
	configPath := writeAPIConfig(t)
	auth, handler := testAPIHandler(t, configPath)

	req := httptest.NewRequest(http.MethodGet, "/api/environments", nil)
	withLocalOrigin(req, testPort)
	req.Header.Set(csrfHeader, auth.token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Environments []struct {
			Name     string `json:"name"`
			Provider string `json:"provider"`
			Project  string `json:"project"`
		} `json:"environments"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.Len(t, body.Environments, 1)
	require.Equal(t, "default", body.Environments[0].Name)
}

func TestAPIRejectsMissingToken(t *testing.T) {
	configPath := writeAPIConfig(t)
	_, handler := testAPIHandler(t, configPath)

	req := httptest.NewRequest(http.MethodGet, "/api/environments", nil)
	withLocalOrigin(req, testPort)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAPIRejectsForeignOrigin(t *testing.T) {
	configPath := writeAPIConfig(t)
	auth, handler := testAPIHandler(t, configPath)

	req := httptest.NewRequest(http.MethodGet, "/api/environments", nil)
	req.Host = "127.0.0.1:11911"
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set(csrfHeader, auth.token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAuthTokenRequiresLocalOrigin(t *testing.T) {
	_, handler := testAPIHandler(t, writeAPIConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/api/auth/token", nil)
	req.Host = "127.0.0.1:11911"
	req.Header.Set("Origin", "http://evil.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAuthTokenWithSameOriginFetch(t *testing.T) {
	_, handler := testAPIHandler(t, writeAPIConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/api/auth/token", nil)
	req.Host = "127.0.0.1:11911"
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestSecurityMiddlewareRejectsRemoteHost(t *testing.T) {
	auth, err := newGUIAuth(testPort)
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "example.com"
	rec := httptest.NewRecorder()
	auth.middleware(mux).ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestLocalhostStaticHandlerSPA(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>ok</html>"), 0o644))

	req := httptest.NewRequest(http.MethodGet, "/missing-route", nil)
	rec := httptest.NewRecorder()
	localhostStaticHandler(dir).ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "ok")
}

func testAPIHandler(t *testing.T, configPath string) (*guiAuth, http.Handler) {
	t.Helper()
	auth, err := newGUIAuth(testPort)
	require.NoError(t, err)
	mux := http.NewServeMux()
	registerAPIRoutes(mux, configPath, auth)
	return auth, auth.middleware(mux)
}

func withLocalOrigin(req *http.Request, port int) {
	req.Host = "127.0.0.1:11911"
	req.Header.Set("Origin", "http://127.0.0.1:11911")
}

func writeAPIConfig(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "ws.yaml")
	require.NoError(t, os.WriteFile(p, []byte(apiTestConfig), 0o644))
	return p
}
