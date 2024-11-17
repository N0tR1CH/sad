package main

import (
	"net/http"

	"github.com/N0tR1CH/sad/views"
	"github.com/N0tR1CH/sad/views/components"
	"github.com/labstack/echo/v4"
)

func (app *application) getCategoriesHandler(c echo.Context) error {
	categories, err := app.models.Categories.GetAll()
	if err != nil {
		return err
	}
	cps := make(components.CategoriesProps, len(categories))
	for i := range cps {
		cps[i].ID = categories[i].ID
		cps[i].Name = categories[i].Name
	}
	return views.Render(c, http.StatusOK, components.Categories(cps))
}
