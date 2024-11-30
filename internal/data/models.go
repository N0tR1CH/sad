package data

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("record could not be updated")
)

type Models struct {
	Discussions interface {
		Insert(discussion *Discussion) error
		Get(id int64) (*Discussion, error)
		GetAll(category string, page int) ([]Discussion, error)
		Update(discussion *Discussion) error
		Delete(id int64) error
	}
	Users interface {
		Insert(user *User) error
		GetByEmail(email string) (*User, error)
		Update(user *User) error
		GetForToken(scope string, plainTextToken string) (*User, error)
		Exists(id int) (bool, error)
		AvatarSrcByID(id int) (string, error)
		Authorized(userID int, permission string) (bool, error)
	}
	Tokens interface {
		New(userID int, lifeTime time.Duration, tokenType TokenType) (*Token, error)
		Insert(t *Token) error
		DeleteAllForUser(scope string, userID int) error
	}
	Categories interface {
		GetAll() ([]Category, error)
	}
	Roles interface {
		Roles(ID int) ([]Role, error)
		RemovePermission(ID int, path, method string) error
		PermissionsLeft(
			ID int,
			allPermissions Permissions,
		) (Permissions, error)
		AddPermission(ID int, permission string) error
	}
}

func NewModels(db *sql.DB) Models {
	return Models{
		Discussions: DiscussionModel{DB: db},
		Users:       UserModel{DB: db},
		Tokens:      TokenModel{DB: db},
		Categories:  CategoryModel{DB: db},
		Roles:       RoleModel{DB: db},
	}
}
