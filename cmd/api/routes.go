package main

import (
	"net/http"

	"github.com/N0tR1CH/sad/cmd/web"
	"github.com/N0tR1CH/sad/views"
	"github.com/N0tR1CH/sad/views/components"
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
	r.GET("/login", app.loginHandler)
	r.GET("/register", func(c echo.Context) error {
		return c.Redirect(http.StatusTemporaryRedirect, "/login")
	})
	r.GET("/alert", app.flashMessageHandler)

	app.discussionsRoutes(r)
	app.usersRoutes(r)

	return r
}

func (app *application) homeHandler(c echo.Context) error {
	c.Set("success", struct{}{})
	return views.Render(
		c,
		http.StatusOK,
		pages.Home(),
	)
}

func (app *application) flashMessageHandler(c echo.Context) error {
	alert := app.sessionManager.Pop(c.Request().Context(), "alert")
	switch alert.(type) {
	case components.AlertProps:
		ap := components.AlertProps{
			Title: alert.(components.AlertProps).Title,
			Text:  alert.(components.AlertProps).Text,
			Icon:  alert.(components.AlertProps).Icon,
		}
		return views.Render(c, http.StatusOK, components.Alert(ap))
	default:
		return c.String(http.StatusOK, "")
	}
}

func (app *application) staticFilesHandler() http.Handler {
	fs := web.GetFileSystem(app.config.useOsFs, app.logger)
	fsServer := http.FileServer(fs)
	handler := http.StripPrefix("/static/", fsServer)
	return handler
}

func (app *application) discussionsRoutes(e *echo.Echo) {
	g := e.Group("/discussions", echo.WrapMiddleware(app.sessionManager.LoadAndSave))
	// Getting all discussions
	g.GET("", app.getDiscussionsHandler)
	// Creating new discussion
	g.GET("/new", app.newDiscussionHandler)
	g.POST("/create", app.createDiscussionHandler)
	// Validating discussion fields
	g.GET("/title", app.validateDiscussionTitleHandler)
	g.GET("/description", app.validateDiscussionDescriptionHandler)
	g.GET("/url", app.validateDiscussionUrlHandler)
	// Generating discussion card preview
	g.GET("/preview", app.genDiscussionPreview)
}

// Create users group and sets its middleware.
//
// It has two parameters: a pointer to application instance with pointer to echo instace.
// usersRoutes function create new group with the route and sets some middleware.
// On the created group it sets handlers.
//
// /users/*
func (app *application) usersRoutes(e *echo.Echo) {
	g := e.Group("/users", echo.WrapMiddleware(app.sessionManager.LoadAndSave))

	// POST /users/create
	g.POST("/create", app.createUserHandler)

	// POST /users/authenticate
	g.POST("/authenticate", app.authenticateUserHandler)

	// POST /users/:id/deauthenticate
	//
	// Params:
	// - id: integer
	g.POST("/:id/deauthenticate", app.deauthenticateUserHandler)

	// GET /users/validateEmail?email=[string]
	g.GET("/validateEmail", app.validateUserEmailHandler)

	// GET /users/validateUsername?username=[string]
	g.GET("/validateUsername", app.validateUserUsernameHandler)

	// GET /users/validatePassword?password=[string]
	g.GET("/validatePassword", app.validateUserPasswordHandler)

	// GET /users/activated/:id/activated?token=[string]
	g.GET("/:id/activated", app.getUserActivationSectionHandler)

	// PUT /users/activated/:id/activated
	//
	// FormData:
	// - token: string
	//
	g.PUT("/:id/activated", app.updateUserActivationStatusHandler)
}
