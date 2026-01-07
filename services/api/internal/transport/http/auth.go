package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/app"
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

type AuthSession = app.AuthSession

type AuthService interface {
	Register(ctx context.Context, in app.RegisterInput) (app.AuthResult, error)
	Login(ctx context.Context, in app.LoginInput) (app.AuthResult, error)
	Authenticate(ctx context.Context, token string) (app.AuthSession, error)
	Logout(ctx context.Context, token string) error
}

type SessionAuthenticator interface {
	Authenticate(ctx context.Context, token string) (app.AuthSession, error)
}

type SessionCookieConfig struct {
	Name     string
	Path     string
	SameSite http.SameSite
	Secure   bool
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type authUserResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type authResponse struct {
	User      authUserResponse `json:"user"`
	ExpiresAt time.Time        `json:"expires_at"`
}

func HandleRegister(svc AuthService, cookieCfg SessionCookieConfig, allowRegister bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, codeMethodNotAllowed, "method not allowed")
			return
		}
		if !allowRegister {
			writeAuthError(w, domain.ErrRegistrationDisabled)
			return
		}

		var req registerRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, codeInvalidRequestBody, "invalid request body")
			return
		}

		result, err := svc.Register(r.Context(), app.RegisterInput{
			Username: req.Username,
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			writeAuthError(w, err)
			return
		}

		setSessionCookie(w, cookieCfg, result.SessionToken, result.ExpiresAt)
		writeAuthResponse(w, result.User, result.ExpiresAt)
	}
}

func HandleLogin(svc AuthService, cookieCfg SessionCookieConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, codeMethodNotAllowed, "method not allowed")
			return
		}

		var req loginRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, codeInvalidRequestBody, "invalid request body")
			return
		}

		result, err := svc.Login(r.Context(), app.LoginInput{
			Identifier: req.Identifier,
			Password:   req.Password,
		})
		if err != nil {
			writeAuthError(w, err)
			return
		}

		setSessionCookie(w, cookieCfg, result.SessionToken, result.ExpiresAt)
		writeAuthResponse(w, result.User, result.ExpiresAt)
	}
}

func HandleLogout(svc AuthService, cookieCfg SessionCookieConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, codeMethodNotAllowed, "method not allowed")
			return
		}

		token, _ := sessionTokenFromRequest(r, cookieCfg.Name)
		if err := svc.Logout(r.Context(), token); err != nil {
			writeError(w, http.StatusInternalServerError, codeInternalError, "internal error")
			return
		}

		clearSessionCookie(w, cookieCfg)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

func HandleMe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, ok := AuthFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, codeUnauthorized, domain.ErrUnauthorized.Error())
			return
		}
		writeAuthResponse(w, session.User, session.ExpiresAt)
	}
}

func writeAuthResponse(w http.ResponseWriter, user domain.User, expiresAt time.Time) {
	resp := authResponse{
		User: authUserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     string(user.Role),
		},
		ExpiresAt: expiresAt,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch err {
	case domain.ErrUsernameRequired:
		writeError(w, http.StatusBadRequest, codeUsernameRequired, err.Error())
	case domain.ErrUsernameInvalid:
		writeError(w, http.StatusBadRequest, codeUsernameInvalid, err.Error())
	case domain.ErrEmailRequired:
		writeError(w, http.StatusBadRequest, codeEmailRequired, err.Error())
	case domain.ErrPasswordRequired:
		writeError(w, http.StatusBadRequest, codePasswordRequired, err.Error())
	case domain.ErrUsernameTaken:
		writeError(w, http.StatusConflict, codeUsernameTaken, err.Error())
	case domain.ErrEmailTaken:
		writeError(w, http.StatusConflict, codeEmailTaken, err.Error())
	case domain.ErrInvalidCredentials:
		writeError(w, http.StatusUnauthorized, codeInvalidCredentials, err.Error())
	case domain.ErrUnauthorized:
		writeError(w, http.StatusUnauthorized, codeUnauthorized, err.Error())
	case domain.ErrRegistrationDisabled:
		writeError(w, http.StatusForbidden, codeRegistrationDisabled, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, codeInternalError, "internal error")
	}
}

func sessionTokenFromRequest(r *http.Request, name string) (string, bool) {
	if name == "" {
		return "", false
	}
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", false
	}
	if cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}

func setSessionCookie(w http.ResponseWriter, cfg SessionCookieConfig, token string, expiresAt time.Time) {
	if cfg.Name == "" {
		return
	}
	maxAge := int(time.Until(expiresAt).Seconds())
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.Name,
		Value:    token,
		Path:     cfg.Path,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
		Expires:  expiresAt,
		MaxAge:   maxAge,
	})
}

func clearSessionCookie(w http.ResponseWriter, cfg SessionCookieConfig) {
	if cfg.Name == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.Name,
		Value:    "",
		Path:     cfg.Path,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}
