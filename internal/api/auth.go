package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"golang.org/x/crypto/bcrypt"
)

const sessionCookie = "firecracker_studio_session"
const sessionLifetime = 12 * time.Hour

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type sessionClaims struct {
	Username string `json:"username"`
	Expires  int64  `json:"expires"`
	Nonce    string `json:"nonce"`
}

func (s *Server) login(r *fastglue.Request) error {
	var req loginRequest
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendJSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "message": err.Error()})
	}
	if s.authUsername == "" || s.authPasswordHash == "" || req.Username != s.authUsername || bcrypt.CompareHashAndPassword([]byte(s.authPasswordHash), []byte(req.Password)) != nil {
		return r.SendJSON(http.StatusUnauthorized, map[string]string{"error": "invalid_credentials", "message": "username or password is incorrect"})
	}
	nonceBytes := make([]byte, 18)
	if _, err := rand.Read(nonceBytes); err != nil {
		return r.SendJSON(http.StatusInternalServerError, map[string]string{"error": "session_failed", "message": "could not create session"})
	}
	claims := sessionClaims{Username: s.authUsername, Expires: time.Now().Add(sessionLifetime).Unix(), Nonce: base64.RawURLEncoding.EncodeToString(nonceBytes)}
	payload, _ := json.Marshal(claims)
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	signature := s.signSession(encoded)
	cookie := fasthttp.AcquireCookie()
	cookie.SetKey(sessionCookie)
	cookie.SetValue(encoded + "." + signature)
	cookie.SetPath("/")
	cookie.SetHTTPOnly(true)
	cookie.SetSecure(s.publicHTTPS)
	cookie.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	cookie.SetExpire(time.Now().Add(sessionLifetime))
	r.RequestCtx.Response.Header.SetCookie(cookie)
	return r.SendJSON(http.StatusOK, map[string]any{"authenticated": true, "username": s.authUsername, "expiresAt": claims.Expires})
}

func (s *Server) authStatus(r *fastglue.Request) error {
	authenticated, username, expires := s.sessionFromRequest(r.RequestCtx)
	return r.SendJSON(http.StatusOK, map[string]any{"configured": s.authConfigured, "authenticated": authenticated, "username": username, "expiresAt": expires})
}

func (s *Server) logout(r *fastglue.Request) error {
	cookie := fasthttp.AcquireCookie()
	cookie.SetKey(sessionCookie)
	cookie.SetValue("")
	cookie.SetPath("/")
	cookie.SetHTTPOnly(true)
	cookie.SetExpire(time.Unix(0, 0))
	r.RequestCtx.Response.Header.SetCookie(cookie)
	return r.SendJSON(http.StatusOK, map[string]bool{"authenticated": false})
}

func (s *Server) sessionFromRequest(ctx *fasthttp.RequestCtx) (bool, string, int64) {
	value := string(ctx.Request.Header.Cookie(sessionCookie))
	parts := strings.Split(value, ".")
	if len(parts) != 2 || !hmac.Equal([]byte(parts[1]), []byte(s.signSession(parts[0]))) {
		return false, "", 0
	}
	decoded, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false, "", 0
	}
	var claims sessionClaims
	if err := json.Unmarshal(decoded, &claims); err != nil || claims.Expires <= time.Now().Unix() || claims.Username != s.authUsername {
		return false, "", 0
	}
	return true, claims.Username, claims.Expires
}

func (s *Server) signSession(payload string) string {
	mac := hmac.New(sha256.New, s.authKey)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) authError(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(http.StatusUnauthorized)
	ctx.Response.Header.SetContentType("application/json")
	ctx.SetBodyString(`{"error":"unauthorized","message":"sign in to Firecracker Studio"}`)
}

func authKey(passwordHash string) []byte {
	digest := sha256.Sum256([]byte(fmt.Sprintf("firecracker-studio:%s", passwordHash)))
	return digest[:]
}
