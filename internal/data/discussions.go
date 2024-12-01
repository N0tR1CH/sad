package data

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Discussion struct {
	ID          int
	CreatedAt   time.Time
	CategoryID  int
	UpdatedAt   time.Time
	Url         string
	Title       string
	Description string
	PreviewSrc  string
}

type DiscussionModel struct {
	DB *sql.DB
}

func (dm DiscussionModel) Insert(discussion *Discussion) error {
	query := `
		INSERT INTO discussions (url, title, description, preview_src, category_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at, category_id
	`
	queryArgs := []any{
		discussion.Url,
		discussion.Title,
		discussion.Description,
		discussion.PreviewSrc,
		discussion.CategoryID,
	}
	return dm.DB.QueryRow(query, queryArgs...).Scan(
		&discussion.ID,
		&discussion.CreatedAt,
		&discussion.UpdatedAt,
		&discussion.CategoryID,
	)
}

func (dm DiscussionModel) Get(id int64) (*Discussion, error) {
	var d Discussion
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	query := `
		SELECT
			id,
			created_at,
			updated_at,
			url,
			title,
			description,
			preview_src,
			category_id
		FROM
			discussions
		WHERE id=$1
	`
	if err := dm.DB.QueryRowContext(ctx, query, &id).Scan(
		&d.ID,
		&d.CreatedAt,
		&d.UpdatedAt,
		&d.Url,
		&d.Title,
		&d.Description,
		&d.PreviewSrc,
		&d.CategoryID,
	); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &d, nil
}

func (dm DiscussionModel) GetAll(category string, page int) ([]Discussion, error) {
	query := `
	SELECT
		d.id,
		d.created_at,
		d.updated_at,
		d.url,
		d.title,
		d.description,
		d.preview_src,
		d.category_id
	FROM
		discussions d
		JOIN categories c ON c.id=d.category_id
	WHERE (LOWER(c.name)=LOWER($1) OR $1='')
	ORDER BY
		d.created_at DESC
	LIMIT 9
	OFFSET $2
	`
	offset := (page - 1) * 9
	rows, err := dm.DB.Query(query, &category, &offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var discussions []Discussion

	for rows.Next() {
		var discussion Discussion
		if err := rows.Scan(
			&discussion.ID,
			&discussion.CreatedAt,
			&discussion.UpdatedAt,
			&discussion.Url,
			&discussion.Title,
			&discussion.Description,
			&discussion.PreviewSrc,
			&discussion.CategoryID,
		); err != nil {
			return discussions, err
		}
		discussions = append(discussions, discussion)
	}
	if err := rows.Err(); err != nil {
		return discussions, err
	}

	return discussions, nil
}

func (dm DiscussionModel) Update(discussion *Discussion) error {
	return nil
}

func (dm DiscussionModel) Delete(id int64) error {
	return nil
}
