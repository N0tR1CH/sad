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

func (cm CommentModel) GetAllWithUser(discussionId int) (Comments, error) {
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
	`
	args := []any{&discussionId}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := cm.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf(
			"in CommentModel#GetAll while querying: %w",
			err,
		)
	}
	defer rows.Close()
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
			return comments, fmt.Errorf(
				"in CommentModel#Get while mapping fields: %w",
				err,
			)
		}
		comments = append(comments, c)
	}
	return comments, nil
}
