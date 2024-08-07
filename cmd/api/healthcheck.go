package main

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

func (app *application) healthcheckhandler(c echo.Context) error {
	return c.String(
		http.StatusOK,
		fmt.Sprintf(
			"status: available\nenvironment: %s\nversion: %s",
			app.config.env,
			version,
		),
	)
}
