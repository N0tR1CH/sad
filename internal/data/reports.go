package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
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
	ReportedUser   User
}

type ReportModel struct {
	DB     *sql.DB
	logger *slog.Logger
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

func (rm ReportModel) AnyAfter(lastSeenId int) (bool, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()
	var areAny bool
	if err := rm.DB.QueryRowContext(
		ctx,
		`SELECT EXISTS(
			SELECT 1 FROM reports WHERE id > $1 FETCH FIRST 1 ROWS
		)`,
		&lastSeenId,
	).Scan(&areAny); err != nil {
		return false, fmt.Errorf(
			"in ReportModel#AnyAfter while querying: %w",
			err,
		)
	}
	return areAny, nil
}

func (rm ReportModel) GetAll(last_seen_id, limit int) ([]Report, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()
	args := []any{&last_seen_id}
	q := `
	SELECT
		r.id,
		r.created_at,
		r.updated_at,
		r.reported_user_id,
		COALESCE(r.discussion_id, 0) AS discussion_id,
		COALESCE(r.comment_id, 0) AS comment_id,
		r.reason,
		COALESCE(u.avatar_src, '') AS avatar_src,
		u.name AS username
	FROM reports r
		INNER JOIN users u ON r.reported_user_id = u.id AND u.banned = false
	WHERE r.id > $1
	ORDER BY r.id
	`
	var reports []Report
	if limit > 0 {
		q += " FETCH FIRST $2 ROWS ONLY"
		args = append(args, &limit)
		reports = make([]Report, 0, limit)
	}
	rows, err := rm.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("in ReportModel#GetAll: %w", err)
	}
	defer func() {
		closeErr := rows.Close()
		if err != nil {
			if closeErr != nil {
				rm.logger.Error(
					"in ReportModel#GetAll while closing rows",
					"err", err.Error(),
				)
			}
			return
		}
		err = closeErr
	}()
	for rows.Next() {
		var r Report
		if err := rows.Scan(
			&r.ID,
			&r.CreatedAt,
			&r.UpdatedAt,
			&r.ReportedUser.ID,
			&r.DiscussionID,
			&r.CommentID,
			&r.Reason,
			&r.ReportedUser.AvatarSrc,
			&r.ReportedUser.Name,
		); err != nil {
			return nil, fmt.Errorf(
				"in ReportModel#GetAll while scanning values: %w",
				err,
			)
		}
		reports = append(reports, r)
	}
	return reports, nil
}
