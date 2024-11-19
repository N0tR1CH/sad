package data

import (
	"database/sql"
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
		INSERT INTO discussions (url, title, description, preview_src)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	queryArgs := []any{
		discussion.Url,
		discussion.Title,
		discussion.Description,
		discussion.PreviewSrc,
	}
	return dm.DB.QueryRow(query, queryArgs...).Scan(
		&discussion.ID,
		&discussion.CreatedAt,
		&discussion.UpdatedAt,
	)
}

func (dm DiscussionModel) Get(id int64) (*Discussion, error) {
	return nil, nil
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
