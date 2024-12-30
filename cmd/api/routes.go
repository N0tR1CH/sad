package main

import (
	"net/http"
	"os"

	"github.com/N0tR1CH/sad/cmd/web"
	"github.com/N0tR1CH/sad/internal/data"
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

	r.GET("/", app.homeHandler)
	r.Static("/public", "cmd/web/public")
	r.GET("/static/*", echo.WrapHandler(staticFilesHandler))
	r.GET("/healthcheck", app.healthcheckhandler)
	r.GET("/login", app.loginHandler)
	r.GET("/register", func(c echo.Context) error {
		return c.Redirect(http.StatusTemporaryRedirect, "/login")
	})
	r.GET("/alert", app.flashMessageHandler)

	app.discussionsRoutes(r)
	app.usersRoutes(r)
	app.categoriesRoutes(r)
	app.rolesRoutes(r)

	r.GET("/routes", app.getRoutes(r))
	return r
}

func (app *application) getRoutes(e *echo.Echo) echo.HandlerFunc {
	routes := e.Routes()
	app.permissions = make([]data.Permission, 0, len(routes)+1)
	for _, route := range routes {
		app.permissions = append(
			app.permissions,
			data.Permission{
				Path:   route.Path,
				Method: route.Method,
			},
		)
	}
	app.permissions = append(
		app.permissions,
		data.Permission{Path: "/routes", Method: "GET"},
	)
	if err := app.models.Roles.AssignAdminAllPermissions(
		app.permissions,
	); err != nil {
		app.logger.Error("database problem", "err", err)
		os.Exit(exitFailure)
	}
	return func(c echo.Context) error {
		return views.Render(
			c,
			http.StatusOK,
			pages.Routes(
				pages.NewRoutesProps(app.permissions),
			),
		)
	}
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
	// Getting certain discussion
	g.GET("/:id", app.getDiscussionHandler)
	// Creating new discussion
	g.GET("/new", app.newDiscussionHandler)
	g.POST("/create", app.createDiscussionHandler)
	// Validating discussion fields
	g.GET("/title", app.validateDiscussionTitleHandler)
	g.GET("/description", app.validateDiscussionDescriptionHandler)
	g.GET("/url", app.validateDiscussionUrlHandler)
	// Generating discussion card preview
	g.GET("/preview", app.genDiscussionPreview)

	app.commentsRoutes(g)
}

func (app *application) commentsRoutes(e *echo.Group) {
	g := e.Group("/:discussionId/comments")

	g.RouteNotFound("/*", func(c echo.Context) error {
		return views.Render(c, http.StatusNotFound, pages.Page404())
	})

	g.GET("", app.getCommentsHandler)
	g.POST("/create", app.createCommentHandler)
	g.POST("/:id/upvote", app.upvoteCommentHandler)
	g.GET("/:id/reply", app.getCommentRepliesHandler)
	g.POST("/:id/report", app.reportCommentHandler)
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

	g.RouteNotFound("/*", func(c echo.Context) error {
		return views.Render(c, http.StatusNotFound, pages.Page404())
	})
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

	// GET /users/:id/avatar
	//
	// Params:
	// - id: integer
	g.GET("/:id/avatar", app.getUserAvatarHandler)

	// PUT /users/activated/:id/activated
	//
	// FormData:
	// - token: string
	//
	g.PUT("/:id/activated", app.updateUserActivationStatusHandler)
}

func (app *application) categoriesRoutes(e *echo.Echo) {
	g := e.Group("/categories")

	g.RouteNotFound("/*", func(c echo.Context) error {
		return views.Render(c, http.StatusNotFound, pages.Page404())
	})

	g.GET("", app.getCategoriesHandler)
}

func (app *application) rolesRoutes(e *echo.Echo) {
	g := e.Group("/roles")

	g.RouteNotFound("/*", func(c echo.Context) error {
		return views.Render(c, http.StatusNotFound, pages.Page404())
	})

	g.GET("", app.getRolesHandler)

	g.DELETE("/:id/permissions", app.deleteRolePermissionHandler)
	g.GET("/permissions", app.getRolePermissionsHandler)
	g.POST("/permissions", app.addRolePermissionsHandler)
}
