package app

import (
	"context"
	"testing"
	"time"

	"github.com/cimillas/ultimate-ticket/services/api/internal/clock"
	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
)

func TestAuthService_Register_ValidatesInput(t *testing.T) {
	repo := newFakeAuthRepo()
	svc := NewAuthService(repo, clock.NewFixed(time.Now()), time.Hour)

	_, err := svc.Register(context.Background(), RegisterInput{})
	if err != domain.ErrUsernameRequired {
		t.Fatalf("expected ErrUsernameRequired, got %v", err)
	}

	_, err = svc.Register(context.Background(), RegisterInput{Username: "user"})
	if err != domain.ErrEmailRequired {
		t.Fatalf("expected ErrEmailRequired, got %v", err)
	}

	_, err = svc.Register(context.Background(), RegisterInput{Username: "user", Email: "user@example.com"})
	if err != domain.ErrPasswordRequired {
		t.Fatalf("expected ErrPasswordRequired, got %v", err)
	}

	_, err = svc.Register(context.Background(), RegisterInput{
		Username: "user@example.com",
		Email:    "user@example.com",
		Password: "secret",
	})
	if err != domain.ErrUsernameInvalid {
		t.Fatalf("expected ErrUsernameInvalid, got %v", err)
	}
}

func TestAuthService_Register_CreatesUserAndSession(t *testing.T) {
	now := time.Date(2025, 2, 10, 10, 0, 0, 0, time.UTC)
	repo := newFakeAuthRepo()
	svc := NewAuthService(repo, clock.NewFixed(now), time.Hour)

	res, err := svc.Register(context.Background(), RegisterInput{
		Username: "user",
		Email:    "user@example.com",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.User.ID == "" {
		t.Fatalf("expected user ID to be set")
	}
	if res.User.Role != domain.UserRoleUser {
		t.Fatalf("expected role user, got %s", res.User.Role)
	}
	if res.SessionToken == "" {
		t.Fatalf("expected session token to be set")
	}
	if res.ExpiresAt.Before(now.Add(59 * time.Minute)) {
		t.Fatalf("expected expires_at ~1h, got %v", res.ExpiresAt)
	}
	if repo.lastCreatedUser.Username != "user" {
		t.Fatalf("expected user persisted")
	}
	if repo.lastSession.UserID != res.User.ID {
		t.Fatalf("expected session for user, got %s", repo.lastSession.UserID)
	}
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	repo := newFakeAuthRepo()
	svc := NewAuthService(repo, clock.NewFixed(time.Now()), time.Hour)

	_, err := svc.Login(context.Background(), LoginInput{
		Identifier: "user",
		Password:   "secret",
	})
	if err != domain.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_CreatesSession(t *testing.T) {
	now := time.Date(2025, 2, 10, 10, 0, 0, 0, time.UTC)
	repo := newFakeAuthRepo()
	svc := NewAuthService(repo, clock.NewFixed(now), time.Hour)

	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	repo.users["user"] = storedUser{
		User: domain.User{
			ID:       "user-1",
			Username: "user",
			Email:    "user@example.com",
			Role:     domain.UserRoleUser,
		},
		PasswordHash: hash,
	}

	res, err := svc.Login(context.Background(), LoginInput{
		Identifier: "user",
		Password:   "secret",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.SessionToken == "" {
		t.Fatalf("expected session token")
	}
	if repo.lastSession.UserID != "user-1" {
		t.Fatalf("expected session for user-1, got %s", repo.lastSession.UserID)
	}
}

func TestAuthService_Login_WithEmailIdentifier(t *testing.T) {
	now := time.Date(2025, 2, 10, 10, 0, 0, 0, time.UTC)
	repo := newFakeAuthRepo()
	svc := NewAuthService(repo, clock.NewFixed(now), time.Hour)

	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	repo.users["user"] = storedUser{
		User: domain.User{
			ID:       "user-1",
			Username: "user",
			Email:    "user@example.com",
			Role:     domain.UserRoleUser,
		},
		PasswordHash: hash,
	}

	res, err := svc.Login(context.Background(), LoginInput{
		Identifier: "user@example.com",
		Password:   "secret",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.User.ID != "user-1" {
		t.Fatalf("expected user-1, got %s", res.User.ID)
	}
}

func TestAuthService_Authenticate_RefreshesSession(t *testing.T) {
	now := time.Date(2025, 2, 10, 10, 0, 0, 0, time.UTC)
	repo := newFakeAuthRepo()
	svc := NewAuthService(repo, clock.NewFixed(now), time.Hour)

	repo.users["user"] = storedUser{
		User: domain.User{
			ID:       "user-1",
			Username: "user",
			Email:    "user@example.com",
			Role:     domain.UserRoleUser,
		},
		PasswordHash: "hash",
	}

	token := "token"
	tokenHash := hashToken(token)
	repo.sessions[tokenHash] = storedSession{
		Session: domain.Session{
			TokenHash:  tokenHash,
			UserID:     "user-1",
			ExpiresAt:  now.Add(10 * time.Minute),
			CreatedAt:  now.Add(-10 * time.Minute),
			LastUsedAt: now.Add(-5 * time.Minute),
		},
	}

	res, err := svc.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.User.ID != "user-1" {
		t.Fatalf("expected user-1, got %s", res.User.ID)
	}
	if res.ExpiresAt.Before(now.Add(59 * time.Minute)) {
		t.Fatalf("expected refreshed expiry, got %v", res.ExpiresAt)
	}
	if repo.lastUpdatedSession.TokenHash != tokenHash {
		t.Fatalf("expected session update")
	}
}

func TestAuthService_BootstrapAdmin_CreatesAdmin(t *testing.T) {
	repo := newFakeAuthRepo()
	svc := NewAuthService(repo, clock.NewFixed(time.Now()), time.Hour)

	if err := svc.BootstrapAdmin(context.Background(), "admin", "admin@example.com", "admin"); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	if repo.lastCreatedUser.Role != domain.UserRoleAdmin {
		t.Fatalf("expected admin role, got %s", repo.lastCreatedUser.Role)
	}
}

func TestAuthService_BootstrapAdmin_NoOpWhenAdminExists(t *testing.T) {
	repo := newFakeAuthRepo()
	repo.users["admin"] = storedUser{
		User: domain.User{
			ID:       "admin-1",
			Username: "admin",
			Email:    "admin@example.com",
			Role:     domain.UserRoleAdmin,
		},
		PasswordHash: "hash",
	}
	svc := NewAuthService(repo, clock.NewFixed(time.Now()), time.Hour)

	if err := svc.BootstrapAdmin(context.Background(), "admin", "admin@example.com", "admin"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastCreatedUser.ID != "" {
		t.Fatalf("expected no new admin to be created")
	}
}

func TestAuthService_BootstrapAdmin_ReturnsErrorWhenUserExists(t *testing.T) {
	repo := newFakeAuthRepo()
	repo.users["admin"] = storedUser{
		User: domain.User{
			ID:       "user-1",
			Username: "admin",
			Email:    "user@example.com",
			Role:     domain.UserRoleUser,
		},
		PasswordHash: "hash",
	}
	svc := NewAuthService(repo, clock.NewFixed(time.Now()), time.Hour)

	if err := svc.BootstrapAdmin(context.Background(), "admin", "admin@example.com", "admin"); err != domain.ErrUsernameTaken {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}
}

func TestAuthService_ResetAuth_ClearsUsersAndSessions(t *testing.T) {
	repo := newFakeAuthRepo()
	repo.users["user"] = storedUser{
		User: domain.User{ID: "user-1", Username: "user", Email: "user@example.com", Role: domain.UserRoleUser},
	}
	repo.sessions["token"] = storedSession{Session: domain.Session{TokenHash: "token", UserID: "user-1"}}
	svc := NewAuthService(repo, clock.NewFixed(time.Now()), time.Hour)

	if err := svc.ResetAuth(context.Background(), "admin", "admin@example.com", "admin"); err != nil {
		t.Fatalf("reset auth: %v", err)
	}
	if len(repo.sessions) != 0 {
		t.Fatalf("expected sessions to be cleared")
	}
	admin, ok := repo.users["admin"]
	if !ok {
		t.Fatalf("expected admin to be created")
	}
	if admin.Role != domain.UserRoleAdmin {
		t.Fatalf("expected admin role, got %s", admin.Role)
	}
	if len(repo.users) != 1 {
		t.Fatalf("expected only admin user, got %d users", len(repo.users))
	}
}

func TestAuthService_Register_RejectsUsernameMatchingEmail(t *testing.T) {
	repo := newFakeAuthRepo()
	repo.users["existing"] = storedUser{
		User: domain.User{ID: "user-1", Username: "existing", Email: "alias", Role: domain.UserRoleUser},
	}
	svc := NewAuthService(repo, clock.NewFixed(time.Now()), time.Hour)

	_, err := svc.Register(context.Background(), RegisterInput{
		Username: "alias",
		Email:    "new@example.com",
		Password: "secret",
	})
	if err != domain.ErrUsernameTaken {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}
}

func TestAuthService_Register_RejectsEmailMatchingUsername(t *testing.T) {
	repo := newFakeAuthRepo()
	repo.users["existing"] = storedUser{
		User: domain.User{ID: "user-1", Username: "existing", Email: "user@example.com", Role: domain.UserRoleUser},
	}
	svc := NewAuthService(repo, clock.NewFixed(time.Now()), time.Hour)

	_, err := svc.Register(context.Background(), RegisterInput{
		Username: "new-user",
		Email:    "existing",
		Password: "secret",
	})
	if err != domain.ErrEmailTaken {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestAuthService_ChangePassword(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 2, 10, 10, 0, 0, 0, time.UTC)

	t.Run("updates password and clears other sessions", func(t *testing.T) {
		repo := newFakeAuthRepo()
		svc := NewAuthService(repo, clock.NewFixed(now), time.Hour)

		oldHash, err := hashPassword("old-secret")
		if err != nil {
			t.Fatalf("hash password: %v", err)
		}
		repo.users["user"] = storedUser{
			User: domain.User{
				ID:       "user-1",
				Username: "user",
				Email:    "user@example.com",
				Role:     domain.UserRoleUser,
			},
			PasswordHash: oldHash,
		}
		keepHash := hashToken("keep-token")
		dropHash := hashToken("drop-token")
		repo.sessions[keepHash] = storedSession{Session: domain.Session{TokenHash: keepHash, UserID: "user-1"}}
		repo.sessions[dropHash] = storedSession{Session: domain.Session{TokenHash: dropHash, UserID: "user-1"}}

		err = svc.ChangePassword(context.Background(), ChangePasswordInput{
			UserID:          "user-1",
			CurrentPassword: "old-secret",
			NewPassword:     "new-secret",
			SessionToken:    "keep-token",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if repo.lastPasswordUserID != "user-1" {
			t.Fatalf("expected password update for user-1, got %s", repo.lastPasswordUserID)
		}
		if repo.lastPasswordHash == "" || repo.lastPasswordHash == oldHash {
			t.Fatalf("expected new password hash to be set")
		}
		if _, ok := repo.sessions[dropHash]; ok {
			t.Fatalf("expected other sessions cleared")
		}
		if _, ok := repo.sessions[keepHash]; !ok {
			t.Fatalf("expected current session to remain")
		}
	})

	t.Run("invalid current password returns error", func(t *testing.T) {
		repo := newFakeAuthRepo()
		svc := NewAuthService(repo, clock.NewFixed(now), time.Hour)

		oldHash, err := hashPassword("old-secret")
		if err != nil {
			t.Fatalf("hash password: %v", err)
		}
		repo.users["user"] = storedUser{
			User: domain.User{
				ID:       "user-1",
				Username: "user",
				Email:    "user@example.com",
				Role:     domain.UserRoleUser,
			},
			PasswordHash: oldHash,
		}

		err = svc.ChangePassword(context.Background(), ChangePasswordInput{
			UserID:          "user-1",
			CurrentPassword: "wrong",
			NewPassword:     "new-secret",
			SessionToken:    "keep-token",
		})
		if err != domain.ErrInvalidCredentials {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
		if repo.lastPasswordUserID != "" {
			t.Fatalf("expected password not updated")
		}
	})

	t.Run("missing passwords return error", func(t *testing.T) {
		repo := newFakeAuthRepo()
		svc := NewAuthService(repo, clock.NewFixed(now), time.Hour)

		err := svc.ChangePassword(context.Background(), ChangePasswordInput{
			UserID:       "user-1",
			SessionToken: "keep-token",
		})
		if err != domain.ErrPasswordRequired {
			t.Fatalf("expected ErrPasswordRequired, got %v", err)
		}
	})

	t.Run("missing user returns error", func(t *testing.T) {
		repo := newFakeAuthRepo()
		svc := NewAuthService(repo, clock.NewFixed(now), time.Hour)

		err := svc.ChangePassword(context.Background(), ChangePasswordInput{
			UserID:          "",
			CurrentPassword: "old-secret",
			NewPassword:     "new-secret",
			SessionToken:    "keep-token",
		})
		if err != domain.ErrUnauthorized {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})
}

type storedUser struct {
	domain.User
	PasswordHash string
}

type storedSession struct {
	domain.Session
}

type fakeAuthRepo struct {
	users              map[string]storedUser
	sessions           map[string]storedSession
	lastCreatedUser    domain.User
	lastSession        domain.Session
	lastUpdatedSession domain.Session
	lastPasswordUserID string
	lastPasswordHash   string
	lastSessionUserID  string
	lastSessionToken   string
}

func newFakeAuthRepo() *fakeAuthRepo {
	return &fakeAuthRepo{
		users:    make(map[string]storedUser),
		sessions: make(map[string]storedSession),
	}
}

func (f *fakeAuthRepo) CreateUser(_ context.Context, user domain.User, passwordHash string) error {
	if _, exists := f.users[user.Username]; exists {
		return domain.ErrUsernameTaken
	}
	for _, existing := range f.users {
		if existing.Email == user.Email {
			return domain.ErrEmailTaken
		}
	}
	f.users[user.Username] = storedUser{User: user, PasswordHash: passwordHash}
	f.lastCreatedUser = user
	return nil
}

func (f *fakeAuthRepo) GetUserByUsername(_ context.Context, username string) (domain.User, string, error) {
	user, ok := f.users[username]
	if !ok {
		return domain.User{}, "", domain.ErrUserNotFound
	}
	return user.User, user.PasswordHash, nil
}

func (f *fakeAuthRepo) GetUserByEmail(_ context.Context, email string) (domain.User, string, error) {
	for _, user := range f.users {
		if user.Email == email {
			return user.User, user.PasswordHash, nil
		}
	}
	return domain.User{}, "", domain.ErrUserNotFound
}

func (f *fakeAuthRepo) GetUserByID(_ context.Context, userID string) (domain.User, error) {
	for _, user := range f.users {
		if user.ID == userID {
			return user.User, nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

func (f *fakeAuthRepo) GetUserByIDWithPasswordHash(_ context.Context, userID string) (domain.User, string, error) {
	for _, user := range f.users {
		if user.ID == userID {
			return user.User, user.PasswordHash, nil
		}
	}
	return domain.User{}, "", domain.ErrUserNotFound
}

func (f *fakeAuthRepo) CreateSession(_ context.Context, session domain.Session) error {
	f.sessions[session.TokenHash] = storedSession{Session: session}
	f.lastSession = session
	return nil
}

func (f *fakeAuthRepo) GetSession(_ context.Context, tokenHash string) (domain.Session, error) {
	session, ok := f.sessions[tokenHash]
	if !ok {
		return domain.Session{}, domain.ErrSessionNotFound
	}
	return session.Session, nil
}

func (f *fakeAuthRepo) UpdateSession(_ context.Context, session domain.Session) error {
	f.sessions[session.TokenHash] = storedSession{Session: session}
	f.lastUpdatedSession = session
	return nil
}

func (f *fakeAuthRepo) DeleteSession(_ context.Context, tokenHash string) error {
	delete(f.sessions, tokenHash)
	return nil
}

func (f *fakeAuthRepo) DeleteSessionsForUserExcept(_ context.Context, userID, tokenHash string) error {
	f.lastSessionUserID = userID
	f.lastSessionToken = tokenHash
	for key, session := range f.sessions {
		if session.UserID == userID && key != tokenHash {
			delete(f.sessions, key)
		}
	}
	return nil
}

func (f *fakeAuthRepo) UpdateUserPassword(_ context.Context, userID, passwordHash string) error {
	for name, user := range f.users {
		if user.ID == userID {
			user.PasswordHash = passwordHash
			f.users[name] = user
			f.lastPasswordUserID = userID
			f.lastPasswordHash = passwordHash
			return nil
		}
	}
	return domain.ErrUserNotFound
}

func (f *fakeAuthRepo) ResetAuth(_ context.Context, admin domain.User, passwordHash string) error {
	f.users = make(map[string]storedUser)
	f.sessions = make(map[string]storedSession)
	f.users[admin.Username] = storedUser{User: admin, PasswordHash: passwordHash}
	f.lastCreatedUser = admin
	return nil
}
