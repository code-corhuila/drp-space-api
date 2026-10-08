package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/code-corhuila/drp-space-api/internal/app"
)

func RequireBearer(tokens app.TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := strings.TrimSpace(r.Header.Get("Authorization"))
			if header == "" {
				writeErr(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Token de autenticación requerido", nil)
				return
			}
			raw := header
			if len(header) >= 7 && strings.EqualFold(header[:7], "Bearer ") {
				raw = strings.TrimSpace(header[7:])
			} else {
				writeErr(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Token de autenticación requerido", nil)
				return
			}
			if raw == "" {
				writeErr(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Token de autenticación requerido", nil)
				return
			}
			sub, err := tokens.Parse(r.Context(), raw)
			if errors.Is(err, app.ErrJWKSUnavailable) {
				writeErr(w, r, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "No se pudo validar el token", nil)
				return
			}
			if err != nil || sub == "" {
				writeErr(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Token de autenticación requerido", nil)
				return
			}
			ctx := context.WithValue(r.Context(), subjectKey, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
