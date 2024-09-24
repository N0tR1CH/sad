package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/N0tR1CH/sad/rate_limiter"
	"github.com/N0tR1CH/sad/views"
	"github.com/N0tR1CH/sad/views/pages"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
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

	corsConfig = func(appConfig *config) middleware.CORSConfig {
		return middleware.CORSConfig{
			AllowOrigins: []string{
				// TODO: Handle cors urls
				func() string {
					if appConfig.env == "development" {
						return fmt.Sprintf("https://localhost:%d", appConfig.port)
					}

					return fmt.Sprintf("https://localhost:%d", appConfig.port)
				}(),
			},
			AllowHeaders: []string{
				echo.HeaderOrigin,
				echo.HeaderContentType,
				echo.HeaderAccept,
			},
			AllowMethods: []string{
				http.MethodGet,
				http.MethodPut,
				http.MethodPost,
				http.MethodDelete,
			},
		}
	}

	rateLimiterConfig = func() rate_limiter.Config {
		client := redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		})
		ctx := context.Background()
		_ = client.FlushDB(ctx).Err()
		return rate_limiter.Config{
			Rediser:   client,
			Max:       100,
			Burst:     200,
			Period:    10 * time.Second,
			Algorithm: rate_limiter.SlidingWindowAlgorithm,
		}
	}
)

func DefaultSkipper(echo.Context) bool {
	return false
}

func (app *application) middleware(e *echo.Echo) {
	e.Pre(middleware.RemoveTrailingSlashWithConfig(trailingSlashConfig))
	e.Use(middleware.RequestLoggerWithConfig(requestLoggerConfig(app.logger)))
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(corsConfig(app.config)))
	e.Use(rate_limiter.NewWithConfig(rateLimiterConfig()))
	e.RouteNotFound("/*", func(c echo.Context) error {
		return views.Render(c, http.StatusNotFound, pages.Page404())
	})
}
