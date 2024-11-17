package data

import (
	"database/sql"
	"time"
)

type Category struct {
	ID        int
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string
}

type CategoryModel struct {
	DB *sql.DB
}

func (cm CategoryModel) GetAll() ([]Category, error) {
	var categories []Category
	query := "SELECT id, created_at, updated_at, name FROM categories"
	rows, err := cm.DB.Query(query)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt, &c.Name); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}
