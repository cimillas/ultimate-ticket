package postgres

import (
	"context"
	"fmt"

	"github.com/cimillas/ultimate-ticket/services/api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepository struct {
	pool *pgxpool.Pool
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

func (r *AuthRepository) CreateUser(ctx context.Context, user domain.User, passwordHash string) error {
	const stmt = `
INSERT INTO users (id, username, email, password_hash, role, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.exec(ctx, stmt, user.ID, user.Username, user.Email, passwordHash, user.Role, user.CreatedAt, user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			switch uniqueConstraintName(err) {
			case "users_username_key":
				return domain.ErrUsernameTaken
			case "users_email_key":
				return domain.ErrEmailTaken
			}
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *AuthRepository) GetUserByUsername(ctx context.Context, username string) (domain.User, string, error) {
	const query = `
SELECT id, username, email, role, password_hash, created_at
FROM users
WHERE username = $1`
	var user domain.User
	var role string
	var hash string
	if err := r.queryRow(ctx, query, username).Scan(&user.ID, &user.Username, &user.Email, &role, &hash, &user.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return domain.User{}, "", domain.ErrUserNotFound
		}
		return domain.User{}, "", fmt.Errorf("get user by username: %w", err)
	}
	user.Role = domain.UserRole(role)
	return user, hash, nil
}

func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, string, error) {
	const query = `
SELECT id, username, email, role, password_hash, created_at
FROM users
WHERE email = $1`
	var user domain.User
	var role string
	var hash string
	if err := r.queryRow(ctx, query, email).Scan(&user.ID, &user.Username, &user.Email, &role, &hash, &user.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return domain.User{}, "", domain.ErrUserNotFound
		}
		return domain.User{}, "", fmt.Errorf("get user by email: %w", err)
	}
	user.Role = domain.UserRole(role)
	return user, hash, nil
}

func (r *AuthRepository) GetUserByID(ctx context.Context, userID string) (domain.User, error) {
	const query = `
SELECT id, username, email, role, created_at
FROM users
WHERE id = $1`
	var user domain.User
	var role string
	if err := r.queryRow(ctx, query, userID).Scan(&user.ID, &user.Username, &user.Email, &role, &user.CreatedAt); err != nil {
		if isInvalidUUID(err) {
			return domain.User{}, domain.ErrInvalidID
		}
		if err == pgx.ErrNoRows {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("get user by id: %w", err)
	}
	user.Role = domain.UserRole(role)
	return user, nil
}

func (r *AuthRepository) GetUserByIDWithPasswordHash(ctx context.Context, userID string) (domain.User, string, error) {
	const query = `
SELECT id, username, email, role, password_hash, created_at
FROM users
WHERE id = $1`
	var user domain.User
	var role string
	var hash string
	if err := r.queryRow(ctx, query, userID).Scan(&user.ID, &user.Username, &user.Email, &role, &hash, &user.CreatedAt); err != nil {
		if isInvalidUUID(err) {
			return domain.User{}, "", domain.ErrInvalidID
		}
		if err == pgx.ErrNoRows {
			return domain.User{}, "", domain.ErrUserNotFound
		}
		return domain.User{}, "", fmt.Errorf("get user by id: %w", err)
	}
	user.Role = domain.UserRole(role)
	return user, hash, nil
}

func (r *AuthRepository) CreateSession(ctx context.Context, session domain.Session) error {
	const stmt = `
INSERT INTO sessions (token_hash, user_id, expires_at, created_at, last_used_at)
VALUES ($1, $2, $3, $4, $5)`
	_, err := r.exec(ctx, stmt, session.TokenHash, session.UserID, session.ExpiresAt, session.CreatedAt, session.LastUsedAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *AuthRepository) GetSession(ctx context.Context, tokenHash string) (domain.Session, error) {
	const query = `
SELECT token_hash, user_id, expires_at, created_at, last_used_at
FROM sessions
WHERE token_hash = $1`
	var session domain.Session
	if err := r.queryRow(ctx, query, tokenHash).Scan(
		&session.TokenHash,
		&session.UserID,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.LastUsedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return domain.Session{}, domain.ErrSessionNotFound
		}
		return domain.Session{}, fmt.Errorf("get session: %w", err)
	}
	return session, nil
}

func (r *AuthRepository) UpdateSession(ctx context.Context, session domain.Session) error {
	const stmt = `
UPDATE sessions
SET expires_at = $2, last_used_at = $3
WHERE token_hash = $1`
	tag, err := r.exec(ctx, stmt, session.TokenHash, session.ExpiresAt, session.LastUsedAt)
	if err != nil {
		return fmt.Errorf("update session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrSessionNotFound
	}
	return nil
}

func (r *AuthRepository) DeleteSession(ctx context.Context, tokenHash string) error {
	const stmt = `DELETE FROM sessions WHERE token_hash = $1`
	if _, err := r.exec(ctx, stmt, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (r *AuthRepository) DeleteSessionsForUserExcept(ctx context.Context, userID, tokenHash string) error {
	const stmt = `DELETE FROM sessions WHERE user_id = $1 AND token_hash <> $2`
	if _, err := r.exec(ctx, stmt, userID, tokenHash); err != nil {
		return fmt.Errorf("delete sessions for user: %w", err)
	}
	return nil
}

func (r *AuthRepository) UpdateUserPassword(ctx context.Context, userID, passwordHash string) error {
	const stmt = `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`
	tag, err := r.exec(ctx, stmt, userID, passwordHash)
	if err != nil {
		if isInvalidUUID(err) {
			return domain.ErrInvalidID
		}
		return fmt.Errorf("update user password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *AuthRepository) ResetAuth(ctx context.Context, admin domain.User, passwordHash string) error {
	return withTx(ctx, r.pool, func(txCtx context.Context) error {
		if _, err := r.exec(txCtx, `DELETE FROM sessions`); err != nil {
			return fmt.Errorf("clear sessions: %w", err)
		}
		if _, err := r.exec(txCtx, `DELETE FROM users`); err != nil {
			return fmt.Errorf("clear users: %w", err)
		}
		if err := r.CreateUser(txCtx, admin, passwordHash); err != nil {
			return fmt.Errorf("create admin: %w", err)
		}
		return nil
	})
}

func (r *AuthRepository) exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if tx := txFromContext(ctx); tx != nil {
		return tx.Exec(ctx, sql, args...)
	}
	return r.pool.Exec(ctx, sql, args...)
}

func (r *AuthRepository) queryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if tx := txFromContext(ctx); tx != nil {
		return tx.QueryRow(ctx, sql, args...)
	}
	return r.pool.QueryRow(ctx, sql, args...)
}
