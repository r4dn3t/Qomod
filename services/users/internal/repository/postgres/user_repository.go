package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"qomod/services/users/internal/domain"
	"qomod/services/users/internal/usecase"
)

// UserRepository implements usecase.UserRepository using pgxpool.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository returns a new instance of UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create inserts a new user record.
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (
			id, cellphone_number, first_name, last_name, birthdate, 
			email, password_hash, status, phone_verified, email_verified, 
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.pool.Exec(
		ctx, query,
		u.ID, u.CellphoneNumber, u.FirstName, u.LastName, u.Birthdate,
		u.Email, u.PasswordHash, u.Status, u.PhoneVerified, u.EmailVerified,
		u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return r.mapError(err)
	}

	return nil
}

// GetByID fetches a user by unique identifier.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, cellphone_number, first_name, last_name, birthdate,
		       email, password_hash, status, phone_verified, email_verified,
		       last_login_at, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1
	`
	return r.scanUser(r.pool.QueryRow(ctx, query, id))
}

// GetByCellphone fetches an active user by phone number.
func (r *UserRepository) GetByCellphone(ctx context.Context, cellphone string) (*domain.User, error) {
	query := `
		SELECT id, cellphone_number, first_name, last_name, birthdate,
		       email, password_hash, status, phone_verified, email_verified,
		       last_login_at, created_at, updated_at, deleted_at
		FROM users
		WHERE cellphone_number = $1 AND deleted_at IS NULL
	`
	return r.scanUser(r.pool.QueryRow(ctx, query, cellphone))
}

// GetByEmail fetches an active user by email address.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, cellphone_number, first_name, last_name, birthdate,
		       email, password_hash, status, phone_verified, email_verified,
		       last_login_at, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`
	return r.scanUser(r.pool.QueryRow(ctx, query, email))
}

// UpdateProfile updates the profile fields of an active user.
func (r *UserRepository) UpdateProfile(ctx context.Context, u *domain.User) error {
	query := `
		UPDATE users 
		SET first_name = $1, last_name = $2, birthdate = $3, email = $4, updated_at = $5
		WHERE id = $6 AND deleted_at IS NULL
	`
	res, err := r.pool.Exec(ctx, query, u.FirstName, u.LastName, u.Birthdate, u.Email, u.UpdatedAt, u.ID)
	if err != nil {
		return r.mapError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// UpdatePassword sets a new password hash for the user.
func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `
		UPDATE users 
		SET password_hash = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`
	now := time.Now().UTC()
	res, err := r.pool.Exec(ctx, query, passwordHash, now, id)
	if err != nil {
		return r.mapError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// UpdateStatus changes the status enum of a user.
func (r *UserRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.Status) error {
	query := `
		UPDATE users 
		SET status = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`
	now := time.Now().UTC()
	res, err := r.pool.Exec(ctx, query, status, now, id)
	if err != nil {
		return r.mapError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// SoftDelete sets deleted_at timestamp and updates status to 'deleted'.
func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users 
		SET deleted_at = $1, status = $2, updated_at = $1
		WHERE id = $3 AND deleted_at IS NULL
	`
	now := time.Now().UTC()
	res, err := r.pool.Exec(ctx, query, now, domain.StatusDeleted, id)
	if err != nil {
		return r.mapError(err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// List queries users with pagination and optional status filter.
func (r *UserRepository) List(ctx context.Context, filter usecase.ListFilter) ([]*domain.User, int64, error) {
	var (
		conditions []string
		args       []any
		argIdx     = 1
	)

	conditions = append(conditions, "deleted_at IS NULL")

	if filter.Status != nil && *filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT id, cellphone_number, first_name, last_name, birthdate,
		       email, password_hash, status, phone_verified, email_verified,
		       last_login_at, created_at, updated_at, deleted_at,
		       COUNT(*) OVER() AS total_count
		FROM users
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	var totalCount int64

	for rows.Next() {
		var u domain.User
		err := rows.Scan(
			&u.ID, &u.CellphoneNumber, &u.FirstName, &u.LastName, &u.Birthdate,
			&u.Email, &u.PasswordHash, &u.Status, &u.PhoneVerified, &u.EmailVerified,
			&u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
			&totalCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, &u)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error reading user rows: %w", err)
	}

	return users, totalCount, nil
}

// scanUser is a helper scanning single row into domain.User.
func (r *UserRepository) scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(
		&u.ID, &u.CellphoneNumber, &u.FirstName, &u.LastName, &u.Birthdate,
		&u.Email, &u.PasswordHash, &u.Status, &u.PhoneVerified, &u.EmailVerified,
		&u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
	return &u, nil
}

// mapError translates database specific driver errors into domain errors.
func (r *UserRepository) mapError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Postgres 23505 = unique_violation
		if pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "cellphone") || strings.Contains(pgErr.Detail, "cellphone_number") {
				return domain.ErrPhoneAlreadyExists
			}
			if strings.Contains(pgErr.ConstraintName, "email") || strings.Contains(pgErr.Detail, "email") {
				return domain.ErrEmailAlreadyExists
			}
		}
	}
	return err
}
