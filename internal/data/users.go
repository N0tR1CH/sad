package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicatedEmail = errors.New("duplicated email")
)

const bcryptCost = 12

type User struct {
	ID        int
	CreatedAt time.Time
	UpdatedAt time.Time
	AvatarSrc string
	Name      string
	Email     string
	Password  password
	Activated bool
	Version   int
	RoleID    int
}

type UserModel struct {
	DB *sql.DB
}

func (um UserModel) GetForToken(scope string, plainTextToken string) (*User, error) {
	var u User
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	query := `
		SELECT
			u.id,
			u.created_at,
			u.updated_at,
			u.name,
			u.email,
			u.password_hash,
			u.activated,
			u.version
		FROM
			users u
			INNER JOIN tokens t ON u.id=t.user_id
		WHERE
			t.hash = $1
			AND t.token_type = $2
			AND t.expired_at > $3`
	hash := sha256.Sum256([]byte(plainTextToken))
	args := []any{hash[:], scope, time.Now()}
	if err := um.DB.QueryRowContext(ctx, query, args...).Scan(
		&u.ID,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.Name,
		&u.Email,
		&u.Password.hash,
		&u.Activated,
		&u.Version,
	); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &u, nil
}

func (um UserModel) Insert(user *User) error {
	ctx := context.Background()
	stmt, err := um.DB.PrepareContext(
		ctx, `
		INSERT INTO users (name, email, password_hash, activated, avatar_src)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at, version, avatar_src
		`,
	)
	if err != nil {
		return fmt.Errorf("In UserModel#Insert: %w", err)
	}
	defer stmt.Close()
	args := []any{
		user.Name,
		user.Email,
		user.Password.hash,
		user.Activated,
		user.AvatarSrc,
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := stmt.QueryRowContext(ctx, args...).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Version,
		&user.AvatarSrc,
	); err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicatedEmail
		default:
			return err
		}
	}
	return nil
}

func (um UserModel) Exists(id int) (bool, error) {
	var exists bool
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	row := um.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT id FROM users WHERE id=$1)", id)

	if err := row.Scan(&exists); err != nil {
		return exists, err
	}
	return exists, nil
}

func (um UserModel) GetByEmail(email string) (*User, error) {
	ctx := context.Background()
	stmt, err := um.DB.PrepareContext(
		ctx, `
		SELECT
			id,
			created_at,
			updated_at,
			name,
			email,
			password_hash,
			activated,
			version
		FROM
			users
		WHERE
			email=$1
		`,
	)
	if err != nil {
		return nil, fmt.Errorf("In UserModel#GetByEmail: %w", err)
	}
	defer stmt.Close()
	args := []any{email}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var user User
	if err := stmt.QueryRowContext(ctx, args...).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
	); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (um UserModel) AvatarSrcByID(id int) (string, error) {
	var src string
	query := "SELECT COALESCE(avatar_src, '') FROM users WHERE id=$1"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := um.DB.QueryRowContext(ctx, query, &id).Scan(&src); err != nil {
		return "", err
	}
	return src, nil
}

func (um UserModel) Update(user *User) error {
	query := `
		UPDATE
			users
		SET
			updated_at = current_timestamp,
			name = $1,
			email = $2,
			password_hash = $3,
			activated = $4,
			version = version + 1
		WHERE
			id = $5 AND version = $6
		RETURNING
			version`
	args := []any{
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.ID,
		&user.Version,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := um.DB.QueryRowContext(ctx, query, args...).Scan(&user.Version); err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicatedEmail
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil
}

func (um UserModel) Authorized(userID int, permission string) (bool, error) {
	var (
		authorized bool
		args       []any
		query      string
	)

	if userID == 0 {
		query = `
			SELECT EXISTS (
				SELECT 1
				FROM roles r
				WHERE r.name='guest' AND permissions @> $1::jsonb
			);
		`
		args = append(args, &permission)
	} else {
		query = `
			SELECT EXISTS (
				SELECT 1
				FROM roles r
				JOIN users u ON r.id = u.role_id
				WHERE u.id=$1 AND permissions @> $2::jsonb
		)`
		args = append(args, &userID, &permission)
	}

	if err := um.DB.QueryRow(
		query,
		args...,
	).Scan(&authorized); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return false, nil
		default:
			return false, err
		}
	}
	return authorized, nil
}

type password struct {
	plaintext *string
	hash      []byte
}

func (p *password) Set(clearPassword string) error {
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(clearPassword),
		bcryptCost,
	)
	if err != nil {
		return err
	}
	p.plaintext = &clearPassword
	p.hash = hash
	return nil
}

func (p *password) Match(clearPassword string) (bool, error) {
	if err := bcrypt.CompareHashAndPassword(
		p.hash,
		[]byte(clearPassword),
	); err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}
