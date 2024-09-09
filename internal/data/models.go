package data

import (
	"database/sql"
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("record could not be updated")
)

type Models struct {
	Discussions interface {
		Insert(discussion *Discussion) error
		Get(id int64) (*Discussion, error)
		GetAll() ([]Discussion, error)
		Update(discussion *Discussion) error
		Delete(id int64) error
	}
	Users interface {
		Insert(user *User) error
		GetByEmail(email string) (*User, error)
		Update(user *User) error
	}
}

func NewModels(db *sql.DB) Models {
	return Models{
		Discussions: DiscussionModel{DB: db},
		Users:       UserModel{DB: db},
	}
}
