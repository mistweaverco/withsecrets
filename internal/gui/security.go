package gui

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

const csrfHeader = "X-WS-GUI-Token"

type guiAuth struct {
	token string
	port  int
}

func newGUIAuth(port int) (*guiAuth, error) {
	token, err := generateToken(32)
	if err != nil {
		return nil, err
	}
	return &guiAuth{token: token, port: port}, nil
}

func generateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (a *guiAuth) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := strings.Split(r.Host, ":")[0]
		if host != "127.0.0.1" && host != "localhost" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		setSecurityHeaders(w)
		next.ServeHTTP(w, r)
	})
}

func setSecurityHeaders(w http.ResponseWriter) {
	// SvelteKit bootstraps via a small inline module loader in index.html.
	// Hashes change each build, so script-src allows 'unsafe-inline' for same-origin SPA only.
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self'; font-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
}

func (a *guiAuth) handleToken(w http.ResponseWriter, r *http.Request) {
	if !a.validateOrigin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": a.token})
}

func (a *guiAuth) apiGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.validateOrigin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if !a.validateToken(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *guiAuth) validateOrigin(r *http.Request) bool {
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		return a.isAllowedOrigin(origin)
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		return a.isAllowedReferer(referer)
	}
	// Same-origin fetches may omit Origin/Referer (especially with Referrer-Policy).
	// Modern browsers set Sec-Fetch-Site; cross-origin attacks use "cross-site".
	if strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")) == "same-origin" {
		return a.isAllowedHost(r.Host)
	}
	return false
}

func (a *guiAuth) isAllowedHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	hostName, portStr, hasPort := strings.Cut(host, ":")
	if hostName != "127.0.0.1" && hostName != "localhost" {
		return false
	}
	if !hasPort {
		return false
	}
	return portStr == fmt.Sprintf("%d", a.port)
}

func (a *guiAuth) isAllowedOrigin(origin string) bool {
	lower := strings.ToLower(origin)
	port := fmt.Sprintf(":%d", a.port)
	return strings.HasPrefix(lower, "http://127.0.0.1"+port) ||
		strings.HasPrefix(lower, "http://localhost"+port)
}

func (a *guiAuth) isAllowedReferer(referer string) bool {
	lower := strings.ToLower(referer)
	port := fmt.Sprintf(":%d", a.port)
	return strings.HasPrefix(lower, "http://127.0.0.1"+port+"/") ||
		strings.HasPrefix(lower, "http://localhost"+port+"/")
}

func (a *guiAuth) validateToken(r *http.Request) bool {
	got := strings.TrimSpace(r.Header.Get(csrfHeader))
	if got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(a.token)) == 1
}
