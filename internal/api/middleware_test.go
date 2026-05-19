package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maximseshuk/snapr/internal/backup"
	"github.com/maximseshuk/snapr/internal/config"
)

func newAuthMW(t *testing.T, auth *config.AuthConfig, secret string) *AuthMiddleware {
	t.Helper()
	cfg := &config.Config{Server: config.ServerConfig{Auth: auth, Secret: secret}}
	mgr := backup.NewManager(cfg)
	return NewAuthMiddleware(mgr)
}

func TestCookieValue(t *testing.T) {
	cases := []struct {
		header, name, want string
	}{
		{"snapr_session=abc.def", "snapr_session", "abc.def"},
		{"first=1; snapr_session=tok; other=2", "snapr_session", "tok"},
		{"  snapr_session=tok  ", "snapr_session", "tok"},
		{"foo=bar", "snapr_session", ""},
		{"", "snapr_session", ""},
		{"snapr_session", "snapr_session", ""},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, cookieValue(c.header, c.name), "cookieValue(%q)", c.header)
	}
}

func TestFirstNonEmpty(t *testing.T) {
	assert.Equal(t, "a", firstNonEmpty("a", "b"))
	assert.Equal(t, "b", firstNonEmpty("", "b"))
	assert.Equal(t, "", firstNonEmpty("", ""))
}

func TestValidateCredentials(t *testing.T) {
	am := newAuthMW(t, &config.AuthConfig{Enabled: true, Username: "admin", Password: "s3cret"}, "key")

	assert.True(t, am.validateCredentials("admin", "s3cret"))
	assert.False(t, am.validateCredentials("admin", "wrong"))
	assert.False(t, am.validateCredentials("alice", "s3cret"))
	assert.False(t, am.validateCredentials("", ""))
}

func TestJWTRoundtrip(t *testing.T) {
	am := newAuthMW(t, &config.AuthConfig{Enabled: true, Username: "admin", Password: "p"}, "secretkey")

	token, err := am.createJWT("admin")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, ok := am.parseJWT(token)
	require.True(t, ok)
	assert.Equal(t, "admin", claims.Username)
}

func TestJWTRejectsWrongUsername(t *testing.T) {
	am := newAuthMW(t, &config.AuthConfig{Enabled: true, Username: "admin", Password: "p"}, "k")
	token, _ := am.createJWT("eve")
	_, ok := am.parseJWT(token)
	assert.False(t, ok, "claims for unknown user must be rejected")
}

func TestJWTRejectsWrongSecret(t *testing.T) {
	a := newAuthMW(t, &config.AuthConfig{Enabled: true, Username: "u", Password: "p"}, "secret-A")
	b := newAuthMW(t, &config.AuthConfig{Enabled: true, Username: "u", Password: "p"}, "secret-B")
	tok, _ := a.createJWT("u")
	_, ok := b.parseJWT(tok)
	assert.False(t, ok)
}

func TestJWTRejectsExpired(t *testing.T) {
	am := newAuthMW(t, &config.AuthConfig{Enabled: true, Username: "u", Password: "p"}, "k")
	claims := &Claims{
		Username: "u",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(am.jwtSecret())
	require.NoError(t, err)
	_, ok := am.parseJWT(token)
	assert.False(t, ok, "expired token must be rejected")
}

func TestJWTRejectsWrongAlg(t *testing.T) {
	am := newAuthMW(t, &config.AuthConfig{Enabled: true, Username: "u", Password: "p"}, "k")
	// Token with "none" alg should be rejected by signing-method check.
	token := jwt.NewWithClaims(jwt.SigningMethodNone, &Claims{Username: "u"})
	s, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	_, ok := am.parseJWT(s)
	assert.False(t, ok)
}

func TestTokenExpirationDefault(t *testing.T) {
	am := newAuthMW(t, &config.AuthConfig{Enabled: true, Username: "u", Password: "p"}, "k")
	assert.Equal(t, time.Duration(defaultSessionDuration)*time.Minute, am.getTokenExpiration())
}

func TestTokenExpirationCustom(t *testing.T) {
	am := newAuthMW(t, &config.AuthConfig{Enabled: true, Username: "u", Password: "p", TokenExpiration: 60}, "k")
	assert.Equal(t, 60*time.Minute, am.getTokenExpiration())
}

func TestCookieSameSite(t *testing.T) {
	cases := map[string]http.SameSite{
		"strict": http.SameSiteStrictMode,
		"STRICT": http.SameSiteStrictMode,
		"none":   http.SameSiteNoneMode,
		"lax":    http.SameSiteLaxMode,
		"weird":  http.SameSiteLaxMode,
		"":       http.SameSiteLaxMode,
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			am := newAuthMW(t, &config.AuthConfig{
				Enabled: true, Username: "u", Password: "p",
				Cookies: &config.CookieConfig{SameSite: in},
			}, "k")
			assert.Equal(t, want, am.getCookieSameSite())
		})
	}
}

func TestCookieSameSite_NilCookies(t *testing.T) {
	am := newAuthMW(t, &config.AuthConfig{Enabled: true, Username: "u", Password: "p"}, "k")
	assert.Equal(t, http.SameSiteLaxMode, am.getCookieSameSite())
}

func TestBuildCookie(t *testing.T) {
	am := newAuthMW(t, &config.AuthConfig{
		Enabled: true, Username: "u", Password: "p",
		Cookies: &config.CookieConfig{Secure: true, SameSite: "strict", Domain: "example.com"},
	}, "k")

	c := am.buildCookie("val", 600)
	assert.Equal(t, sessionCookieName, c.Name)
	assert.Equal(t, "val", c.Value)
	assert.Equal(t, "/", c.Path)
	assert.Equal(t, "example.com", c.Domain)
	assert.Equal(t, 600, c.MaxAge)
	assert.True(t, c.Secure)
	assert.True(t, c.HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, c.SameSite)
}

func TestBuildCookie_Logout(t *testing.T) {
	am := newAuthMW(t, &config.AuthConfig{Enabled: true, Username: "u", Password: "p"}, "k")
	c := am.buildCookie("", -1)
	assert.Equal(t, "", c.Value)
	assert.Equal(t, -1, c.MaxAge, "MaxAge=-1 clears the cookie")
}

func TestJwtSecret_FallbackWhenEmpty(t *testing.T) {
	am := newAuthMW(t, &config.AuthConfig{Enabled: true, Username: "u", Password: "p"}, "")
	s := am.jwtSecret()
	assert.Len(t, s, 32, "fallback secret is 32 random bytes")

	am2 := newAuthMW(t, &config.AuthConfig{Enabled: true, Username: "u", Password: "p"}, "explicit")
	assert.Equal(t, []byte("explicit"), am2.jwtSecret())
}
