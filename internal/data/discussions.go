package data

import "database/sql"

type DiscussionModel struct {
	DB *sql.DB
}

func (dm DiscussionModel) Insert(discussion *Discussion) error {
	query := `
		INSERT INTO discussions (url, title, description)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	queryArgs := []any{discussion.Url, discussion.Title, discussion.Description}
	return dm.DB.QueryRow(query, queryArgs...).Scan(
		&discussion.ID,
		&discussion.CreatedAt,
		&discussion.UpdatedAt,
	)
}

func (dm DiscussionModel) Get(id int64) (*Discussion, error) {
	return nil, nil
}

func (dm DiscussionModel) Update(discussion *Discussion) error {
	return nil
}

func (dm DiscussionModel) Delete(id int64) error {
	return nil
}
