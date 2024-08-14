package data

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrRecordNotFound = errors.New("record not found")
)

type Models struct {
	Discussions interface {
		Insert(discussion *Discussion) error
		Get(id int64) (*Discussion, error)
		Update(discussion *Discussion) error
		Delete(id int64) error
	}
}

func NewModels(db *sql.DB) Models {
	return Models{
		Discussions: DiscussionModel{DB: db},
	}
}

type Discussion struct {
	ID          int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Url         string
	Title       string
	Description string
}
