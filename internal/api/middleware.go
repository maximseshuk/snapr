package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/maximseshuk/snapr/internal/backup"
	"github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/i18n"
)

const (
	sessionCookieName      = "snapr_session"
	defaultSessionDuration = 30 // minutes
	tokenRefreshThreshold  = 5  // minutes; refresh when less than this remains
)

type AuthMiddleware struct {
	manager  *backup.Manager
	logger   zerolog.Logger
	fallback []byte // generated random secret used when server.secret is empty
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func NewAuthMiddleware(manager *backup.Manager) *AuthMiddleware {
	logger := log.With().Str("component", "auth").Logger()

	fallback := make([]byte, 32)
	if _, err := rand.Read(fallback); err != nil {
		logger.Fatal().Err(err).Msg("Failed to generate fallback secret")
	}

	return &AuthMiddleware{
		manager:  manager,
		logger:   logger,
		fallback: fallback,
	}
}

type loginInput struct {
	RemoteAddr string `header:"X-Forwarded-For"`
	RealIP     string `header:"X-Real-IP"`
	AcceptLang string `header:"Accept-Language"`
	Body       struct {
		Username string `json:"username" required:"true" minLength:"1"`
		Password string `json:"password" required:"true" minLength:"1"`
	}
}

type successOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      struct {
		Success bool `json:"success"`
	}
}

type checkAuthOutput struct {
	Body struct {
		Authenticated bool `json:"authenticated"`
		AuthEnabled   bool `json:"authEnabled"`
	}
}

type sessionInput struct {
	Cookie string `header:"Cookie"`
}

// registerAuth wires login/logout/check onto the huma API. These endpoints stay
// reachable even when server.auth.enabled is false so the UI can detect no-auth
// mode via /auth/check.
func (am *AuthMiddleware) registerAuth(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/login",
		Summary:     "Issue a session cookie",
		Tags:        []string{"Auth"},
	}, func(_ context.Context, in *loginInput) (*successOutput, error) {
		lang := i18n.DetectLanguage(in.AcceptLang)
		if !am.validateCredentials(in.Body.Username, in.Body.Password) {
			am.logger.Warn().
				Str("username", in.Body.Username).
				Str("ip", firstNonEmpty(in.RealIP, in.RemoteAddr)).
				Msg("Failed login attempt")
			return nil, huma.Error401Unauthorized(i18n.T(lang, "error.invalid_credentials"))
		}
		token, err := am.createJWT(in.Body.Username)
		if err != nil {
			am.logger.Error().Err(err).Msg("Failed to create JWT token")
			return nil, huma.Error500InternalServerError("Failed to create session")
		}
		am.logger.Info().
			Str("username", in.Body.Username).
			Str("ip", firstNonEmpty(in.RealIP, in.RemoteAddr)).
			Msg("Successful login")
		out := &successOutput{
			SetCookie: am.buildCookie(token, int(am.getTokenExpiration().Seconds())),
		}
		out.Body.Success = true
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "logout",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/logout",
		Summary:     "Clear the session cookie",
		Tags:        []string{"Auth"},
	}, func(_ context.Context, _ *struct{}) (*successOutput, error) {
		out := &successOutput{SetCookie: am.buildCookie("", -1)}
		out.Body.Success = true
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "checkAuth",
		Method:      http.MethodGet,
		Path:        "/api/v1/auth/check",
		Summary:     "Report current auth state",
		Tags:        []string{"Auth"},
	}, func(_ context.Context, in *sessionInput) (*checkAuthOutput, error) {
		out := &checkAuthOutput{}
		if am.authConfig() == nil || !am.authConfig().Enabled {
			out.Body.Authenticated = true
			out.Body.AuthEnabled = false
			return out, nil
		}
		out.Body.AuthEnabled = true
		if token := cookieValue(in.Cookie, sessionCookieName); token != "" {
			if _, ok := am.parseJWT(token); ok {
				out.Body.Authenticated = true
			}
		}
		return out, nil
	})
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func cookieValue(header, name string) string {
	for _, part := range strings.Split(header, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 && kv[0] == name {
			return kv[1]
		}
	}
	return ""
}

func (am *AuthMiddleware) authConfig() *config.AuthConfig {
	return am.manager.Config().Server.Auth
}

// Empty server.secret → random per-process key, sessions die on restart.
func (am *AuthMiddleware) jwtSecret() []byte {
	if s := am.manager.Config().Server.Secret; s != "" {
		return []byte(s)
	}
	return am.fallback
}

func (am *AuthMiddleware) getTokenExpiration() time.Duration {
	if am.authConfig().TokenExpiration > 0 {
		return time.Duration(am.authConfig().TokenExpiration) * time.Minute
	}
	return defaultSessionDuration * time.Minute
}

func (am *AuthMiddleware) getCookieSecure() bool {
	if am.authConfig().Cookies != nil {
		return am.authConfig().Cookies.Secure
	}
	return false
}

func (am *AuthMiddleware) getCookieSameSite() http.SameSite {
	if am.authConfig().Cookies != nil && am.authConfig().Cookies.SameSite != "" {
		switch strings.ToLower(am.authConfig().Cookies.SameSite) {
		case "strict":
			return http.SameSiteStrictMode
		case "none":
			return http.SameSiteNoneMode
		case "lax":
			return http.SameSiteLaxMode
		default:
			return http.SameSiteLaxMode
		}
	}
	return http.SameSiteLaxMode
}

func (am *AuthMiddleware) getCookieDomain() string {
	if am.authConfig().Cookies != nil {
		return am.authConfig().Cookies.Domain
	}
	return ""
}

func (am *AuthMiddleware) buildCookie(value string, maxAge int) http.Cookie {
	return http.Cookie{ //nolint:gosec // Secure/HttpOnly/SameSite set via getters below
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		Domain:   am.getCookieDomain(),
		MaxAge:   maxAge,
		Secure:   am.getCookieSecure(),
		HttpOnly: true,
		SameSite: am.getCookieSameSite(),
	}
}

func (am *AuthMiddleware) validateCredentials(username, password string) bool {
	usernameMatch := subtle.ConstantTimeCompare(
		[]byte(username),
		[]byte(am.authConfig().Username),
	) == 1

	passwordMatch := subtle.ConstantTimeCompare(
		[]byte(password),
		[]byte(am.authConfig().Password),
	) == 1

	return usernameMatch && passwordMatch
}

func (am *AuthMiddleware) createJWT(username string) (string, error) {
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(am.getTokenExpiration())),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "snapr",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(am.jwtSecret())
}

func (am *AuthMiddleware) parseJWT(tokenString string) (*Claims, bool) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return am.jwtSecret(), nil
	})

	if err != nil {
		return nil, false
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, false
	}

	if claims.Username != am.authConfig().Username {
		return nil, false
	}

	return claims, true
}
