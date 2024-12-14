package data

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type Permission struct {
	Path   string `json:"path"`
	Method string `json:"method"`
}

type Permissions []Permission

func (p Permissions) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *Permissions) Scan(value any) error {
	switch value.(type) {
	case []byte:
		return json.Unmarshal(value.([]byte), &p)
	default:
		return errors.New("type assertion to []byte failed")
	}
}

type Role struct {
	ID          int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string
	Permissions Permissions
}

type RoleModel struct {
	DB *sql.DB
}

func (rm RoleModel) Roles(ID int) ([]Role, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	query := `
		SELECT id, created_at, updated_at, name, permissions
		FROM roles
		WHERE id=$1 OR $1=0
		ORDER BY id ASC
	`
	rows, err := rm.DB.QueryContext(ctx, query, &ID)
	if err != nil {
		return nil, fmt.Errorf("in RoleModel#Roles: %w", err)
	}
	defer func() {
		rcErr := rows.Close()
		if rcErr != nil && err == nil {
			err = rcErr
		}
	}()

	roles := make([]Role, 0)
	for rows.Next() {
		var r Role
		if err := rows.Scan(
			&r.ID,
			&r.CreatedAt,
			&r.UpdatedAt,
			&r.Name,
			&r.Permissions,
		); err != nil {
			return nil, fmt.Errorf("in RoleModel#Roles: %w", err)
		}
		roles = append(roles, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("in RoleModel#Roles: %w", err)
	}
	return roles, nil
}

func (rm RoleModel) RemovePermission(ID int, path, method string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	query := `
        UPDATE roles r
        SET permissions = (
            SELECT jsonb_agg(elem)
            FROM jsonb_array_elements(r.permissions) AS elem
            WHERE NOT (
                elem->>'path' = $1 AND
                elem->>'method' = $2
            )
        )
        WHERE id = $3
    `
	args := []any{&path, &method, &ID}
	res, err := rm.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("in RoleModel#RemovePermission: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("in RoleModel#RemovePermission: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf(
			"in RoleModel#RemovePermission: %w",
			errors.New("0 rows affected"),
		)
	}
	return nil
}

func (rm RoleModel) PermissionsLeft(
	ID int,
	allPermissions Permissions,
) (Permissions, error) {
	roles, err := rm.Roles(ID)
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, errors.New("no roles with such id")
	}
	permissions := roles[0].Permissions
	permissionsMap := make(map[Permission]struct{})
	for _, p := range permissions {
		permissionsMap[p] = struct{}{}
	}
	leftPermissions := make(Permissions, 0)
	for _, p := range allPermissions {
		if _, ok := permissionsMap[p]; !ok {
			leftPermissions = append(leftPermissions, p)
		}
	}
	return leftPermissions, nil
}

func (rm RoleModel) AddPermission(ID int, permission string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	query := `
		UPDATE roles
		SET permissions = permissions || $1::JSONB
		WHERE id=$2
	`
	args := []any{&permission, &ID}
	if _, err := rm.DB.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("in RoleModel#AddPermission: %w", err)
	}
	return nil
}

func (rm RoleModel) AssignAdminAllPermissions(allPermissions Permissions) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	bytes, err := json.Marshal(allPermissions)
	if err != nil {
		return err
	}
	json := string(bytes)
	query := "UPDATE roles SET permissions = $1::JSONB WHERE name='admin'"
	if _, err := rm.DB.ExecContext(ctx, query, &json); err != nil {
		return fmt.Errorf("in RoleModel#AssignAdminAllPermissions: %w", err)
	}
	return nil
}
