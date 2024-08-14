package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/N0tR1CH/sad/internal/data"
	"github.com/N0tR1CH/sad/views"
	"github.com/N0tR1CH/sad/views/components"
	"github.com/N0tR1CH/sad/views/pages"
	"github.com/go-playground/validator/v10"
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
		Url         string `form:"url" validate:"required,url"`
	}

	if err := c.Bind(&input); err != nil {
		return c.String(http.StatusBadRequest, "bad request")
	}

	if err := c.Validate(&input); err != nil {
		errs := make([]string, 0, len(err.(validator.ValidationErrors)))
		for _, ve := range err.(validator.ValidationErrors) {
			switch ve.Tag() {
			case "required":
				errs = append(
					errs,
					fmt.Sprintf("Field '%s' can't be blank.", ve.Field()),
				)
			case "max":
				errs = append(
					errs,
					fmt.Sprintf(
						"Field '%s' maximum length is %s characters.",
						ve.Field(),
						ve.Param(),
					),
				)
			default:
				errs = append(errs, ve.Error())
			}
		}
		return c.String(http.StatusBadRequest, strings.Join(errs, "\n"))
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
