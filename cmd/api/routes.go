package main

import (
	"crypto/subtle"
	"net/http"

	"github.com/N0tR1CH/sad/cmd/web"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func (app *application) routes() http.Handler {
	r := echo.New()
	app.middleware(r)

	r.GET(
		"/healthcheck",
		app.healthcheckhandler,
		middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
			if subtle.ConstantTimeCompare([]byte(username), []byte("health")) == 1 &&
				subtle.ConstantTimeCompare([]byte(password), []byte("check")) == 1 {
				return true, nil
			}
			return false, nil
		}),
	)

	r.GET(
		"/static/*",
		echo.WrapHandler(
			http.StripPrefix(
				"/static/",
				http.FileServer(web.GetFileSystem(app.config.useOsFs, app.logger)),
			),
		),
	)

	return r
}
