package gui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateOriginAndToken(t *testing.T) {
	auth, err := newGUIAuth(11911)
	require.NoError(t, err)

	okReq := httptest.NewRequest(http.MethodGet, "/api/environments", nil)
	okReq.Header.Set("Origin", "http://127.0.0.1:11911")
	okReq.Header.Set(csrfHeader, auth.token)
	require.True(t, auth.validateOrigin(okReq))
	require.True(t, auth.validateToken(okReq))

	badOrigin := httptest.NewRequest(http.MethodGet, "/api/environments", nil)
	badOrigin.Header.Set("Origin", "http://evil.example")
	badOrigin.Header.Set(csrfHeader, auth.token)
	require.False(t, auth.validateOrigin(badOrigin))

	badToken := httptest.NewRequest(http.MethodGet, "/api/environments", nil)
	badToken.Header.Set("Origin", "http://127.0.0.1:11911")
	badToken.Header.Set(csrfHeader, "wrong")
	require.False(t, auth.validateToken(badToken))

	sameOrigin := httptest.NewRequest(http.MethodGet, "/api/auth/token", nil)
	sameOrigin.Host = "127.0.0.1:11911"
	sameOrigin.Header.Set("Sec-Fetch-Site", "same-origin")
	require.True(t, auth.validateOrigin(sameOrigin))

	crossSite := httptest.NewRequest(http.MethodGet, "/api/auth/token", nil)
	crossSite.Host = "127.0.0.1:11911"
	crossSite.Header.Set("Sec-Fetch-Site", "cross-site")
	require.False(t, auth.validateOrigin(crossSite))
}

func TestSecurityHeaders(t *testing.T) {
	auth, err := newGUIAuth(11911)
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "127.0.0.1:11911"
	rec := httptest.NewRecorder()
	auth.middleware(mux).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Security-Policy"), "default-src 'self'")
	require.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	require.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
}
