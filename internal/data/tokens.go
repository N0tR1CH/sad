package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"errors"
	"fmt"
	"time"
)

const (
	TokenTypeActivation     = "activation"
	TokenTypeAuthentication = "authentication"
)

type (
	void      struct{}
	TokenType string
	Token     struct {
		PlainText string
		Hash      []byte
		UserID    int
		ExpiredAt time.Time
		TokenType TokenType
	}
	TokenModel struct {
		DB *sql.DB
	}
)

var (
	member        void
	tokenTypesSet map[TokenType]void = map[TokenType]void{
		TokenTypeAuthentication: member,
		TokenTypeActivation:     member,
	}
)

func (t TokenType) validate() error {
	if _, ok := tokenTypesSet[t]; ok {
		return nil
	}
	return errors.New("Invalid token type")
}

func genToken(userID int, lifeTime time.Duration, tokenType TokenType) (*Token, error) {
	if err := tokenType.validate(); err != nil {
		return nil, err
	}

	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, err
	}

	plainText := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(plainText))
	expiredAt := time.Now().Add(lifeTime)
	t := &Token{
		PlainText: plainText,
		Hash:      hash[:],
		UserID:    userID,
		ExpiredAt: expiredAt,
		TokenType: tokenType,
	}
	return t, nil
}

func (tm TokenModel) New(userID int, lifeTime time.Duration, tokenType TokenType) (*Token, error) {
	t, err := genToken(userID, lifeTime, tokenType)
	if err != nil {
		return nil, fmt.Errorf("In TokenModel#New while generating token: %w", err)
	}
	if err := tm.Insert(t); err != nil {
		return nil, fmt.Errorf("In TokenModel#New while inserting token: %w", err)
	}
	return t, nil
}

func (tm TokenModel) Insert(t *Token) error {
	query := `
		INSERT INTO tokens (hash, user_id, expired_at, token_type)
		VALUES ($1, $2, $3, $4)`
	args := []any{t.Hash, t.UserID, t.ExpiredAt, string(t.TokenType)}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := tm.DB.ExecContext(ctx, query, args...); err != nil {
		return err
	}

	return nil
}

func (tm TokenModel) DeleteAllForUser(scope string, userID int) error {
	query := `DELETE FROM tokens WHERE token_type = $1 AND user_id=$2`
	args := []any{scope, userID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := tm.DB.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	return nil
}
