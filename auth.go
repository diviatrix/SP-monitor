package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
)

var sessions = struct {
	sync.RWMutex
	tokens map[string]string // token -> username
}{tokens: map[string]string{}}

func newToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte("fallbacktoken"))
	}
	return hex.EncodeToString(b)
}

func respondJSON(w http.ResponseWriter, v any) { respondJSONCode(w, http.StatusOK, v) }

func respondJSONCode(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func getSessionToken(r *http.Request) string {
	c, err := r.Cookie("session")
	if err != nil {
		return ""
	}
	return c.Value
}

func authUser(r *http.Request) string {
	tok := getSessionToken(r)
	if tok == "" {
		return ""
	}
	sessions.RLock()
	defer sessions.RUnlock()
	return sessions.tokens[tok]
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return xr
	}
	ip := r.RemoteAddr
	if i := strings.LastIndex(ip, ":"); i > 0 {
		return ip[:i]
	}
	return ip
}
