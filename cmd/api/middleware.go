package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/N0tR1CH/sad/views"
	"github.com/N0tR1CH/sad/views/pages"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

var (
	trailingSlashConfig = middleware.TrailingSlashConfig{
		RedirectCode: http.StatusMovedPermanently,
	}

	requestLoggerConfig = func(logger *slog.Logger) middleware.RequestLoggerConfig {
		return middleware.RequestLoggerConfig{
			LogStatus:   true,
			LogURI:      true,
			LogError:    true,
			HandleError: true,
			LogValuesFunc: func(
				c echo.Context,
				v middleware.RequestLoggerValues,
			) error {
				if v.Error == nil {
					logger.LogAttrs(
						context.Background(),
						slog.LevelInfo,
						"REQUEST",
						slog.String("uri", v.URI),
						slog.Int("status", v.Status),
					)
				} else {
					logger.LogAttrs(
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
		}
	}

	corsConfig = func(port int) middleware.CORSConfig {
		return middleware.CORSConfig{
			AllowOrigins: []string{
				fmt.Sprintf("http://localhost:%d", port),
			},
			AllowMethods: []string{
				http.MethodGet,
				http.MethodPut,
				http.MethodPost,
				http.MethodDelete,
			},
		}
	}

	rateLimiterConfig = middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(10),
				Burst:     30,
				ExpiresIn: 3 * time.Minute,
			},
		),
		IdentifierExtractor: func(c echo.Context) (string, error) {
			id := c.RealIP()
			return id, nil
		},
		ErrorHandler: func(c echo.Context, err error) error {
			return c.JSON(http.StatusForbidden, nil)
		},
		DenyHandler: func(c echo.Context, identifier string, err error) error {
			return c.JSON(http.StatusTooManyRequests, nil)
		},
	}
)

func DefaultSkipper(echo.Context) bool {
	return false
}

func (app *application) middleware(e *echo.Echo) {
	e.Pre(middleware.RemoveTrailingSlashWithConfig(trailingSlashConfig))
	e.Use(middleware.RequestLoggerWithConfig(requestLoggerConfig(app.logger)))
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(corsConfig(app.config.port)))
	e.Use(middleware.RateLimiterWithConfig(rateLimiterConfig))
	e.RouteNotFound("/*", func(c echo.Context) error {
		return views.Render(c, http.StatusNotFound, pages.Page404())
	})
}
