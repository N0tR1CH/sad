package data

import (
	"database/sql"
	"time"
)

type Discussion struct {
	ID          int
	CreatedAt   time.Time
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

func (dm DiscussionModel) GetAll() ([]Discussion, error) {
	rows, err := dm.DB.Query("SELECT * FROM discussions")
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
