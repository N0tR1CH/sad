package data

import (
	"context"
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
	Name      string
	Email     string
	Password  password
	Activated bool
	Version   int
}

type UserModel struct {
	DB *sql.DB
}

func (um UserModel) Insert(user *User) error {
	ctx := context.Background()
	stmt, err := um.DB.PrepareContext(
		ctx, `
		INSERT INTO users (name, email, password_hash, activated)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at, version
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
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := stmt.QueryRowContext(ctx, args...).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Version,
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
		WHERE
			email=$1
		FROM
			users
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

func (um UserModel) Update(user *User) error {
	ctx := context.Background()
	stmt, err := um.DB.PrepareContext(
		ctx, `
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
			version
		`,
	)
	if err != nil {
		return fmt.Errorf("In UserModel#Update: %w", err)
	}
	defer stmt.Close()
	args := []any{
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
		&user.ID,
		&user.Version,
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := stmt.QueryRowContext(ctx, args...).Scan(&user.Version); err != nil {
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

type password struct {
	plaintest *string
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
	p = &password{
		plaintest: &clearPassword,
		hash:      hash,
	}
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
			return false, nil
		}
	}
	return true, nil
}
