package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/N0tR1CH/sad/views"
	"github.com/N0tR1CH/sad/views/pages"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func (app *application) middleware(e *echo.Echo) {
	e.Pre(middleware.RemoveTrailingSlashWithConfig(
		middleware.TrailingSlashConfig{
			RedirectCode: http.StatusMovedPermanently,
		},
	))
	e.Use(
		middleware.RequestLoggerWithConfig(
			middleware.RequestLoggerConfig{
				LogStatus:   true,
				LogURI:      true,
				LogError:    true,
				HandleError: true,
				LogValuesFunc: func(
					c echo.Context,
					v middleware.RequestLoggerValues,
				) error {
					if v.Error == nil {
						app.logger.LogAttrs(
							context.Background(),
							slog.LevelInfo,
							"REQUEST",
							slog.String("uri", v.URI),
							slog.Int("status", v.Status),
						)
					} else {
						app.logger.LogAttrs(
							context.Background(),
							slog.LevelError,
							"REQUEST_ERROR",
							slog.String("uri", v.URI),
							slog.Int("status", v.Status),
							slog.String("err", v.Error.Error()),
						)
					}
					return nil
				},
			},
		),
	)
	e.Use(middleware.Recover())
	e.Use(
		middleware.CORSWithConfig(
			middleware.CORSConfig{
				AllowOrigins: []string{
					fmt.Sprintf("http://localhost:%d", app.config.port),
				},
				AllowMethods: []string{
					http.MethodGet,
					http.MethodPut,
					http.MethodPost,
					http.MethodDelete,
				},
			},
		),
	)

	e.RouteNotFound("/*", func(c echo.Context) error {
		return views.Render(c, http.StatusNotFound, pages.Page404())
	})
}
