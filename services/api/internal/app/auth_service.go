package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/clock"
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
	"golang.org/x/crypto/argon2"
)

const (
	defaultSessionTTL   = time.Hour
	sessionTokenBytes   = 32
	argonTime           = 3
	argonMemory         = 64 * 1024
	argonThreads        = 2
	argonKeyLen         = 32
	argonSaltLen        = 16
	argonEncodedVersion = 19
)

type AuthRepository interface {
	CreateUser(ctx context.Context, user domain.User, passwordHash string) error
	GetUserByUsername(ctx context.Context, username string) (domain.User, string, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, string, error)
	GetUserByID(ctx context.Context, userID string) (domain.User, error)
	CreateSession(ctx context.Context, session domain.Session) error
	GetSession(ctx context.Context, tokenHash string) (domain.Session, error)
	UpdateSession(ctx context.Context, session domain.Session) error
	DeleteSession(ctx context.Context, tokenHash string) error
	ResetAuth(ctx context.Context, admin domain.User, passwordHash string) error
}

type AuthService struct {
	repo       AuthRepository
	clock      clock.Clock
	sessionTTL time.Duration
}

func NewAuthService(repo AuthRepository, clk clock.Clock, sessionTTL time.Duration) *AuthService {
	if sessionTTL <= 0 {
		sessionTTL = defaultSessionTTL
	}
	return &AuthService{
		repo:       repo,
		clock:      clk,
		sessionTTL: sessionTTL,
	}
}

type RegisterInput struct {
	Username string
	Email    string
	Password string
}

type LoginInput struct {
	Identifier string
	Password   string
}

type AuthResult struct {
	User         domain.User
	SessionToken string
	ExpiresAt    time.Time
}

type AuthSession struct {
	User      domain.User
	ExpiresAt time.Time
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (AuthResult, error) {
	username := strings.TrimSpace(in.Username)
	email := strings.TrimSpace(in.Email)
	if username == "" {
		return AuthResult{}, domain.ErrUsernameRequired
	}
	if strings.Contains(username, "@") {
		return AuthResult{}, domain.ErrUsernameInvalid
	}
	if email == "" {
		return AuthResult{}, domain.ErrEmailRequired
	}
	if in.Password == "" {
		return AuthResult{}, domain.ErrPasswordRequired
	}

	if _, _, err := s.repo.GetUserByUsername(ctx, username); err == nil {
		return AuthResult{}, domain.ErrUsernameTaken
	} else if !errors.Is(err, domain.ErrUserNotFound) {
		return AuthResult{}, err
	}
	if _, _, err := s.repo.GetUserByEmail(ctx, username); err == nil {
		return AuthResult{}, domain.ErrUsernameTaken
	} else if !errors.Is(err, domain.ErrUserNotFound) {
		return AuthResult{}, err
	}
	if _, _, err := s.repo.GetUserByEmail(ctx, email); err == nil {
		return AuthResult{}, domain.ErrEmailTaken
	} else if !errors.Is(err, domain.ErrUserNotFound) {
		return AuthResult{}, err
	}
	if _, _, err := s.repo.GetUserByUsername(ctx, email); err == nil {
		return AuthResult{}, domain.ErrEmailTaken
	} else if !errors.Is(err, domain.ErrUserNotFound) {
		return AuthResult{}, err
	}

	now := s.clock.Now()
	user := domain.User{
		ID:        newUUID(),
		Username:  username,
		Email:     email,
		Role:      domain.UserRoleUser,
		CreatedAt: now,
	}

	hash, err := hashPassword(in.Password)
	if err != nil {
		return AuthResult{}, err
	}
	if err := s.repo.CreateUser(ctx, user, hash); err != nil {
		return AuthResult{}, err
	}

	return s.createSession(ctx, user, now)
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (AuthResult, error) {
	identifier := strings.TrimSpace(in.Identifier)
	if identifier == "" || in.Password == "" {
		return AuthResult{}, domain.ErrInvalidCredentials
	}

	var (
		user domain.User
		hash string
		err  error
	)
	if strings.Contains(identifier, "@") {
		user, hash, err = s.repo.GetUserByEmail(ctx, identifier)
	} else {
		user, hash, err = s.repo.GetUserByUsername(ctx, identifier)
	}
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return AuthResult{}, domain.ErrInvalidCredentials
		}
		return AuthResult{}, err
	}

	ok, err := verifyPassword(in.Password, hash)
	if err != nil || !ok {
		return AuthResult{}, domain.ErrInvalidCredentials
	}

	now := s.clock.Now()
	return s.createSession(ctx, user, now)
}

func (s *AuthService) Authenticate(ctx context.Context, token string) (AuthSession, error) {
	if token == "" {
		return AuthSession{}, domain.ErrUnauthorized
	}
	hash := hashToken(token)
	session, err := s.repo.GetSession(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return AuthSession{}, domain.ErrUnauthorized
		}
		return AuthSession{}, err
	}

	now := s.clock.Now()
	if !session.ExpiresAt.After(now) {
		_ = s.repo.DeleteSession(ctx, hash)
		return AuthSession{}, domain.ErrUnauthorized
	}

	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		return AuthSession{}, domain.ErrUnauthorized
	}

	session.LastUsedAt = now
	session.ExpiresAt = now.Add(s.sessionTTL)
	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return AuthSession{}, err
	}

	return AuthSession{User: user, ExpiresAt: session.ExpiresAt}, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.repo.DeleteSession(ctx, hashToken(token))
}

func (s *AuthService) BootstrapAdmin(ctx context.Context, username, email, password string) error {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	if username == "" {
		return domain.ErrUsernameRequired
	}
	if email == "" {
		return domain.ErrEmailRequired
	}
	if password == "" {
		return domain.ErrPasswordRequired
	}

	user, _, err := s.repo.GetUserByUsername(ctx, username)
	if err == nil {
		if user.Role != domain.UserRoleAdmin {
			return domain.ErrUsernameTaken
		}
		return nil
	}
	if !errors.Is(err, domain.ErrUserNotFound) {
		return err
	}

	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	admin := domain.User{
		ID:        newUUID(),
		Username:  username,
		Email:     email,
		Role:      domain.UserRoleAdmin,
		CreatedAt: s.clock.Now(),
	}
	return s.repo.CreateUser(ctx, admin, hash)
}

func (s *AuthService) ResetAuth(ctx context.Context, username, email, password string) error {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	if username == "" {
		return domain.ErrUsernameRequired
	}
	if email == "" {
		return domain.ErrEmailRequired
	}
	if password == "" {
		return domain.ErrPasswordRequired
	}

	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	admin := domain.User{
		ID:        newUUID(),
		Username:  username,
		Email:     email,
		Role:      domain.UserRoleAdmin,
		CreatedAt: s.clock.Now(),
	}
	return s.repo.ResetAuth(ctx, admin, hash)
}

func (s *AuthService) createSession(ctx context.Context, user domain.User, now time.Time) (AuthResult, error) {
	token, err := generateToken()
	if err != nil {
		return AuthResult{}, err
	}
	expiresAt := now.Add(s.sessionTTL)
	session := domain.Session{
		TokenHash:  hashToken(token),
		UserID:     user.ID,
		ExpiresAt:  expiresAt,
		CreatedAt:  now,
		LastUsedAt: now,
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return AuthResult{}, err
	}
	return AuthResult{User: user, SessionToken: token, ExpiresAt: expiresAt}, nil
}

func generateToken() (string, error) {
	buf := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	saltEncoded := base64.RawStdEncoding.EncodeToString(salt)
	hashEncoded := base64.RawStdEncoding.EncodeToString(hash)
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argonEncodedVersion, argonMemory, argonTime, argonThreads, saltEncoded, hashEncoded)
	return encoded, nil
}

func verifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, err
	}
	if version != argonEncodedVersion {
		return false, errors.New("incompatible hash version")
	}
	var memory uint32
	var timeParam uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeParam, &threads); err != nil {
		return false, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	hash := argon2.IDKey([]byte(password), salt, timeParam, memory, threads, uint32(len(expectedHash)))
	if subtle.ConstantTimeCompare(hash, expectedHash) == 1 {
		return true, nil
	}
	return false, nil
}
