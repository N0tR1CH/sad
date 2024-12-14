package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Comment struct {
	ID           int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	UserId       int
	DiscussionId int
	Content      string
	U            User
}

type Comments []Comment

type CommentModel struct {
	DB *sql.DB
}

func (cm CommentModel) Insert(c *Comment) error {
	q := `
		INSERT INTO comments (user_id, discussion_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at, user_id, discussion_id, content
	`
	args := []any{&c.UserId, &c.DiscussionId, &c.Content}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := cm.DB.QueryRowContext(ctx, q, args...).Scan(
		&c.ID,
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.UserId,
		&c.DiscussionId,
		&c.Content,
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
	return nil
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
			u.avatar_src
		FROM comments c
			INNER JOIN users u ON c.user_id=u.id
		WHERE discussion_id=$1 OR $1=0
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
		"SELECT COUNT(*) FROM comments WHERE discussion_id=$1",
		&discussionId,
	).Scan(&commsCount); err != nil {
		return nil, 0, err
	}
	currCommCount := commsCount - ((page-1)*10 + len(comments))
	return comments, currCommCount, nil
}
