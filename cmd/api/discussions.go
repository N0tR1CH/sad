package main

import (
	"fmt"
	"net/http"

	"github.com/N0tR1CH/sad/internal/data"
	"github.com/N0tR1CH/sad/views"
	"github.com/N0tR1CH/sad/views/components"
	"github.com/N0tR1CH/sad/views/pages"
	"github.com/labstack/echo/v4"
)

func (app *application) newDiscussionHandler(c echo.Context) error {
	url := c.QueryParam("url")
	_, HTMX := c.Request().Header[http.CanonicalHeaderKey("HX-Request")]
	if HTMX {
		return views.Render(c, http.StatusOK, components.DiscussionForm(url))
	}
	return views.Render(c, http.StatusOK, pages.NewDiscussionPage(url))
}

func (app *application) createDiscussionHandler(c echo.Context) error {
	var input struct {
		Title       string `form:"title" validate:"required,max=130"`
		Description string `form:"description" validate:"required,max=4000"`
		Url         string `form:"url" validate:"required"`
	}

	if err := c.Bind(&input); err != nil {
		return c.String(http.StatusBadRequest, "bad request")
	}
	if err := c.Validate(&input); err != nil {
		return err
	}
	if err := app.models.Discussions.Insert(
		&data.Discussion{
			Title:       input.Title,
			Url:         input.Url,
			Description: input.Description,
		},
	); err != nil {
		return err
	}

	return c.String(http.StatusOK, fmt.Sprintf("%v", input))
}
