package main

import (
	"net/http"

	"github.com/N0tR1CH/sad/cmd/web"
	"github.com/N0tR1CH/sad/views"
	"github.com/N0tR1CH/sad/views/pages"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

func (app *application) routes() http.Handler {
	r := echo.New()

	r.Validator = NewCustomValidator(
		validator.New(
			validator.WithRequiredStructEnabled(),
		),
	)

	app.middleware(r)

	staticFilesHandler := app.staticFilesHandler()

	r.Static("/public", "cmd/web/public")
	r.GET("/static/*", echo.WrapHandler(staticFilesHandler))
	r.GET("/healthcheck", app.healthcheckhandler)
	r.GET("/", app.homeHandler)

	app.discussionsRoutes(r)

	return r
}

func (app *application) homeHandler(c echo.Context) error {
	return views.Render(c, http.StatusOK, pages.Home())
}

func (app *application) staticFilesHandler() http.Handler {
	fs := web.GetFileSystem(app.config.useOsFs, app.logger)
	fsServer := http.FileServer(fs)
	handler := http.StripPrefix("/static/", fsServer)
	return handler
}

func (app *application) discussionsRoutes(e *echo.Echo) {
	g := e.Group("/discussions")
	// Creating new discussion
	g.GET("/new", app.newDiscussionHandler)
	g.POST("", app.createDiscussionHandler)
	// Validating discussion fields
	g.GET("/title", app.validateDiscussionTitleHandler)
	g.GET("/description", app.validateDiscussionDescriptionHandler)
	g.GET("/url", app.validateDiscussionUrlHandler)
}
