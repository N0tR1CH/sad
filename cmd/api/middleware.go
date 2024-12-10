package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/N0tR1CH/sad/rate_limiter"
	"github.com/N0tR1CH/sad/views/components"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
)

type middlewareErr string

const (
	errNotAuthenticated = middlewareErr("user is not authenticated")
)

func (me middlewareErr) Error() string {
	return string(me)
}

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

func (app *application) userIdExtraction(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := app.sessionManager.GetInt(c.Request().Context(), "userID")
		c.Set("userID", id)
		return next(c)
	}
}

func (app *application) authorize(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		app.logger.Info("app#authorize",
			"method", c.Request().Method,
			"path", c.Path(),
		)
		bytes, _ := json.Marshal(
			[]map[string]string{
				{
					"path":   c.Path(),
					"method": c.Request().Method,
				},
			},
		)
		permission := string(bytes)
		authorized, err := app.models.Users.Authorized(
			c.Get("userID").(int),
			permission,
		)
		if err != nil {
			app.logger.Error("app#authorize", "err", err.Error())
			return err
		}
		if !authorized {
			app.logger.Info("app#authorize", "userID", c.Get("userID").(int))
			app.logger.Info("app#authorize", "Path", c.Path())
			app.sessionManager.Put(
				c.Request().Context(),
				"alert",
				components.AlertProps{
					Title: "Not Authorized",
					Text:  "You are not authorized to do that!",
					Icon:  components.Error,
				},
			)
			c.Response().Header().Set("HX-Redirect", "/")
			return c.NoContent(http.StatusOK)
		}
		return next(c)
	}
}

func addHtmxToContext(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		_, ok := c.Request().Header[http.CanonicalHeaderKey("HX-Request")]
		c.Set("HTMX", ok)
		return next(c)
	}
}

func (app *application) middleware(e *echo.Echo) {
	e.Pre(middleware.RemoveTrailingSlashWithConfig(trailingSlashConfig))
	e.Use(middleware.RequestLoggerWithConfig(requestLoggerConfig(app.logger)))
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())
	e.Use(middleware.CORSWithConfig(corsConfig(app.config)))
	e.Use(rate_limiter.NewWithConfig(rateLimiterConfig()))
	e.Use(middleware.CSRF())
	e.Use(echo.WrapMiddleware(app.sessionManager.LoadAndSave))
	e.Use(app.userIdExtraction)
	e.Use(app.authorize)
	e.Use(addHtmxToContext)
}
