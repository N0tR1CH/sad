package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type Report struct {
	ID             int
	CreatedAt      time.Time
	UpdatedAt      time.Time
	UserID         int
	ReportedUserID int
	DiscussionID   int
	CommentID      int
	Reason         string
}

type ReportModel struct {
	DB *sql.DB
}

func (rm ReportModel) Insert(r *Report) error {
	var (
		discussionID sql.NullInt64
		commentID    sql.NullInt64
	)
	if r.DiscussionID != 0 {
		discussionID.Int64 = int64(r.DiscussionID)
		discussionID.Valid = true
	}
	if r.CommentID != 0 {
		commentID.Int64 = int64(r.CommentID)
		commentID.Valid = true
	}
	q := `
		INSERT INTO reports (
			user_id,
			reported_user_id,
			discussion_id,
			comment_id,
			reason
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	args := []any{
		&r.UserID,
		&r.ReportedUserID,
		discussionID,
		commentID,
		&r.Reason,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rm.DB.QueryRowContext(ctx, q, args...).Scan(
		&r.ID,
		&r.CreatedAt,
		&r.UpdatedAt,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Message == `duplicate key value violates unique constraint "reports_user_id_reported_user_id_comment_id_key"` {
			return fmt.Errorf(
				"user cannot be reported twice for the same thing: %w",
				ErrUniquenessViolation,
			)
		}
		return fmt.Errorf("in ReportModel#Insert: %w", err)
	}
	return nil
}
