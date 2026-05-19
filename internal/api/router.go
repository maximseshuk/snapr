package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/maximseshuk/snapr/internal/backup"
	"github.com/maximseshuk/snapr/internal/i18n"
	"github.com/maximseshuk/snapr/internal/logger"
)

func MountAPI(r chi.Router, manager *backup.Manager, authMiddleware *AuthMiddleware, sinks *logger.FileSinks, version string) {
	humaCfg := buildHumaConfig(version)
	humaAPI := humachi.New(r, humaCfg)
	humaAPI.UseMiddleware(languageHumaMiddleware)
	authMiddleware.applyTo(humaAPI)

	RegisterOperations(humaAPI, manager, authMiddleware, sinks, version)
}

func BuildOpenAPI(version string) ([]byte, error) {
	r := chi.NewRouter()
	humaCfg := buildHumaConfig(version)
	humaAPI := humachi.New(r, humaCfg)
	authMiddleware := &AuthMiddleware{}
	RegisterOperations(humaAPI, nil, authMiddleware, nil, version)
	return humaAPI.OpenAPI().YAML()
}

func BuildOpenAPIJSON(version string) ([]byte, error) {
	r := chi.NewRouter()
	humaCfg := buildHumaConfig(version)
	humaAPI := humachi.New(r, humaCfg)
	authMiddleware := &AuthMiddleware{}
	RegisterOperations(humaAPI, nil, authMiddleware, nil, version)
	return humaAPI.OpenAPI().MarshalJSON()
}

func languageHumaMiddleware(hctx huma.Context, next func(huma.Context)) {
	lang := i18n.DetectLanguage(hctx.Header("Accept-Language"))
	hctx = huma.WithValue(hctx, langCtxKey, lang)
	next(hctx)
}

func buildHumaConfig(version string) huma.Config {
	cfg := huma.DefaultConfig("snapr API", version)
	cfg.DocsPath = "/api/v1/docs"
	cfg.OpenAPIPath = "/api/v1/openapi"
	cfg.DocsRenderer = huma.DocsRendererScalar
	cfg.Info.Description = "snapr is a self-hosted backup service. This document describes its JSON HTTP API."
	cfg.CreateHooks = nil
	return cfg
}

func (am *AuthMiddleware) applyTo(api huma.API) {
	api.UseMiddleware(func(hctx huma.Context, next func(huma.Context)) {
		op := hctx.Operation()
		if op == nil || !requiresSession(op) {
			next(hctx)
			return
		}
		if am.authConfig() == nil || !am.authConfig().Enabled {
			next(hctx)
			return
		}
		token := cookieValue(hctx.Header("Cookie"), sessionCookieName)
		if token == "" {
			am.writeUnauthorized(hctx)
			return
		}
		claims, ok := am.parseJWT(token)
		if !ok {
			am.writeUnauthorized(hctx)
			return
		}
		if time.Until(claims.ExpiresAt.Time) < tokenRefreshThreshold*time.Minute {
			if newToken, err := am.createJWT(claims.Username); err == nil {
				c := am.buildCookie(newToken, int(am.getTokenExpiration().Seconds()))
				hctx.AppendHeader("Set-Cookie", c.String())
				am.logger.Debug().Str("username", claims.Username).Msg("Token refreshed automatically")
			}
		}
		next(hctx)
	})
}

func requiresSession(op *huma.Operation) bool {
	for _, sec := range op.Security {
		if _, ok := sec["session"]; ok {
			return true
		}
	}
	return false
}

func (am *AuthMiddleware) writeUnauthorized(hctx huma.Context) {
	lang := "en"
	if v := hctx.Context().Value(langCtxKey); v != nil {
		if s, ok := v.(string); ok && s != "" {
			lang = s
		}
	}
	hctx.SetHeader("Content-Type", "application/json; charset=utf-8")
	hctx.SetStatus(http.StatusUnauthorized)
	body, _ := json.Marshal(map[string]string{"error": i18n.T(lang, "error.unauthorized")})
	_, _ = hctx.BodyWriter().Write(body)
}
