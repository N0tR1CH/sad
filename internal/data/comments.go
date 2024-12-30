package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type Comment struct {
	ID           int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	UserId       int
	DiscussionId int
	Content      string
	U            User
	NumUpvotes   int
	ParentId     int
}

type Comments []Comment

type CommentModel struct {
	DB *sql.DB
}

func (cm CommentModel) Insert(c *Comment) error {
	var parentId sql.NullInt64
	if c.ParentId != 0 {
		parentId.Int64 = int64(c.ParentId)
		parentId.Valid = true
	}

	q := `
		INSERT INTO comments (user_id, discussion_id, content, parent_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at, user_id, discussion_id, content, parent_id
	`
	args := []any{&c.UserId, &c.DiscussionId, &c.Content, &parentId}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := cm.DB.QueryRowContext(ctx, q, args...).Scan(
		&c.ID,
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.UserId,
		&c.DiscussionId,
		&c.Content,
		&parentId,
	); err != nil {
		return fmt.Errorf(
			"inserting comment with "+
				"userId=%d "+
				"discussionId: %d "+
				"failed: %w",
			c.UserId,
			c.DiscussionId,
			err,
		)
	}
	if parentId.Valid {
		c.ParentId = int(parentId.Int64)
	}
	return nil
}

func (cm CommentModel) Upvote(userId, commentId int) error {
	q := "INSERT INTO upvotes (user_id, comment_id) VALUES($1, $2)"
	args := []any{&userId, &commentId}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := cm.DB.ExecContext(ctx, q, args...); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Message == `duplicate key value violates unique constraint "upvotes_user_id_comment_id_key"` {
			return fmt.Errorf("user can upvote comment only once: %w", ErrUniquenessViolation)
		}
		return fmt.Errorf("in commentModel#upvote: %w", err)
	}
	return nil
}

func (cm CommentModel) GetAllChildren(parentId, page int) (
	comms Comments,
	numCurrComms int,
	err error,
) {
	q := `
		SELECT
			c.id,
			c.created_at,
			c.updated_at,
			c.user_id,
			c.discussion_id,
			c.content,
			c.parent_id,
			u.name,
			u.avatar_src,
			COUNT(up.id)
		FROM comments c
			INNER JOIN users u ON c.user_id=u.id
			LEFT JOIN upvotes up ON up.comment_id=c.id
		WHERE c.parent_id=$1 OR $1=0
		GROUP BY c.id, u.id
		ORDER BY c.created_at DESC
		LIMIT 10
		OFFSET $2
	`
	offset := (page - 1) * 10
	args := []any{&parentId, &offset}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := cm.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"in CommentModel#GetAllChildren while querying: %w",
			err,
		)
	}
	defer func() {
		rcErr := rows.Close()
		if rcErr != nil && err == nil {
			err = rcErr
		}
	}()
	for rows.Next() {
		var c Comment
		if err := rows.Scan(
			&c.ID,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.UserId,
			&c.DiscussionId,
			&c.Content,
			&c.ParentId,
			&c.U.Name,
			&c.U.AvatarSrc,
			&c.NumUpvotes,
		); err != nil {
			return comms, 0, fmt.Errorf(
				"in CommentModel#GetAllChildren while while mapping fields: %w",
				err,
			)
		}
		comms = append(comms, c)
	}

	var commsCount int
	if err := cm.DB.QueryRow(
		"SELECT COUNT(*) FROM comments WHERE parent_id=$1",
		&parentId,
	).Scan(&commsCount); err != nil {
		return nil, 0, err
	}
	numCurrComms = commsCount - ((page-1)*10 + len(comms))
	return comms, numCurrComms, nil
}

func (cm CommentModel) GetAllWithUser(discussionId int, page int) (Comments, int, error) {
	q := `
		SELECT
			c.id,
			c.created_at,
			c.updated_at,
			c.user_id,
			c.discussion_id,
			c.content,
			u.name,
			u.avatar_src,
			COUNT(up.id)
		FROM comments c
			INNER JOIN users u ON c.user_id=u.id
			LEFT JOIN upvotes up ON up.comment_id=c.id
		WHERE (c.discussion_id=$1 OR $1=0) AND c.parent_id IS NULL
		GROUP BY c.id, u.id
		ORDER BY c.created_at DESC
		LIMIT 10
		OFFSET $2
	`
	offset := (page - 1) * 10
	args := []any{&discussionId, &offset}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := cm.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"in CommentModel#GetAll while querying: %w",
			err,
		)
	}
	defer func() {
		rcErr := rows.Close()
		if rcErr != nil && err == nil {
			err = rcErr
		}
	}()
	var comments Comments
	for rows.Next() {
		var c Comment
		if err := rows.Scan(
			&c.ID,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.UserId,
			&c.DiscussionId,
			&c.Content,
			&c.U.Name,
			&c.U.AvatarSrc,
			&c.NumUpvotes,
		); err != nil {
			return comments, 0, fmt.Errorf(
				"in CommentModel#Get while mapping fields: %w",
				err,
			)
		}
		comments = append(comments, c)
	}

	var commsCount int
	if err := cm.DB.QueryRow(
		"SELECT COUNT(*) FROM comments WHERE parent_id IS NULL AND discussion_id=$1",
		&discussionId,
	).Scan(&commsCount); err != nil {
		return nil, 0, err
	}
	currCommCount := commsCount - ((page-1)*10 + len(comments))
	return comments, currCommCount, nil
}
