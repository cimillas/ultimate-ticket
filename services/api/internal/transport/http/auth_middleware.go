package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

type authContextKey struct{}

func WithAuth(ctx context.Context, session AuthSession) context.Context {
	return context.WithValue(ctx, authContextKey{}, session)
}

func AuthFromContext(ctx context.Context) (AuthSession, bool) {
	session, ok := ctx.Value(authContextKey{}).(AuthSession)
	return session, ok
}

func RequireAuth(svc SessionAuthenticator, cookieCfg SessionCookieConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := sessionTokenFromRequest(r, cookieCfg.Name)
		if !ok {
			writeError(w, http.StatusUnauthorized, codeUnauthorized, domain.ErrUnauthorized.Error())
			return
		}

		session, err := svc.Authenticate(r.Context(), token)
		if err != nil {
			if errors.Is(err, domain.ErrUnauthorized) {
				writeError(w, http.StatusUnauthorized, codeUnauthorized, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, codeInternalError, "internal error")
			return
		}

		setSessionCookie(w, cookieCfg, token, session.ExpiresAt)
		next.ServeHTTP(w, r.WithContext(WithAuth(r.Context(), session)))
	})
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := AuthFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, codeUnauthorized, domain.ErrUnauthorized.Error())
			return
		}
		if session.User.Role != domain.UserRoleAdmin {
			writeError(w, http.StatusForbidden, codeForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}
