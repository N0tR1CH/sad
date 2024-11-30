package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/N0tR1CH/sad/views"
	"github.com/N0tR1CH/sad/views/pages"
	"github.com/labstack/echo/v4"
)

func (app *application) getRolesHandler(c echo.Context) error {
	roles, err := app.models.Roles.Roles(0)
	if err != nil {
		app.logger.Error(
			"in app#getRolesHandler",
			"err",
			err.Error(),
			"roles",
			roles,
		)
	}
	return views.Render(
		c,
		http.StatusOK,
		pages.RolesPage(
			pages.RolesPageProps{
				Rtpvms: pages.NewRolesTableViewModel(roles),
				Pfvm:   pages.NewPermissionFormViewModel(roles),
			},
		),
	)
}

func (app *application) deleteRolePermissionHandler(c echo.Context) error {
	var input struct {
		ID     string `param:"id" validate:"required,number"`
		Path   string `query:"path" validate:"required"`
		Method string `query:"method" validate:"required"`
	}
	if err := c.Bind(&input); err != nil {
		app.logger.Error(
			"app#deleteRolePermissionHandler",
			"err",
			err.Error(),
			"input",
			fmt.Sprintf("+%v", input),
		)
		return err
	}
	if err := c.Validate(&input); err != nil {
		app.logger.Error(
			"app#deleteRolePermissionHandler",
			"err",
			err.Error(),
			"input",
			fmt.Sprintf("+%v", input),
		)
	}
	iID, err := strconv.Atoi(input.ID)
	if err != nil {
		return err
	}
	if err := app.models.Roles.RemovePermission(
		iID,
		input.Path,
		input.Method,
	); err != nil {
		return err
	}
	return c.NoContent(http.StatusOK)
}

func (app *application) getRolePermissionsHandler(c echo.Context) error {
	var input struct {
		ID string `query:"roleId" validate:"required,number"`
	}
	if err := c.Bind(&input); err != nil {
		app.logger.Error(
			"app#getRolePermissionHandler",
			"err",
			err.Error(),
			"input",
			fmt.Sprintf("+%v", input),
		)
		return err
	}
	if err := c.Validate(&input); err != nil {
		app.logger.Error(
			"app#getRolePermissionHandler",
			"err",
			err.Error(),
			"input",
			fmt.Sprintf("+%v", input),
		)
		return err
	}

	id, err := strconv.Atoi(input.ID)
	if err != nil {
		return err
	}
	if left := c.QueryParam("left"); left == "true" {
		pl, err := app.models.Roles.PermissionsLeft(id, app.permissions)
		if err != nil {
			return err
		}
		return views.Render(c, http.StatusOK, pages.PermissionOptions(pl))
	}

	return nil
}

func (app *application) addRolePermissionsHandler(c echo.Context) error {
	var input struct {
		ID         string `form:"roleId" validate:"required,number"`
		Permission string `form:"permission" validate:"required,json"`
	}
	if err := c.Bind(&input); err != nil {
		app.logger.Error(
			"app#getRolePermissionHandler",
			"err",
			err.Error(),
			"input",
			fmt.Sprintf("+%v", input),
		)
		return err
	}
	if err := c.Validate(&input); err != nil {
		app.logger.Error(
			"app#getRolePermissionHandler",
			"err",
			err.Error(),
			"input",
			fmt.Sprintf("+%v", input),
		)
		return err
	}
	id, err := strconv.Atoi(input.ID)
	if err != nil {
		return err
	}
	if err := app.models.Roles.AddPermission(id, input.Permission); err != nil {
		return err
	}
	c.Response().Header().Set("HX-Location", "/roles")
	return c.NoContent(http.StatusOK)
}
