package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/N0tR1CH/sad/rate_limiter"
	"github.com/N0tR1CH/sad/views"
	"github.com/N0tR1CH/sad/views/components"
	"github.com/N0tR1CH/sad/views/pages"
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
		allowOrigins := []string(nil)
		switch appConfig.env {
		case "development":
			allowOrigins = append(
				allowOrigins,
				fmt.Sprintf("https://localhost:%d", appConfig.port),
			)
		case "production":
			allowOrigins = append(
				allowOrigins,
				"https://damian-richter-soft.works",
				"https://www.damian-richter-soft.works",
			)
		}
		return middleware.CORSConfig{
			AllowOrigins: allowOrigins,
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

	rateLimiterConfig = func(env string) rate_limiter.Config {
		client := redis.NewClient(&redis.Options{
			Addr: func() string {
				if env == "development" {
					return "localhost:6379"
				}
				return "redis:6379"
			}(),
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
	sessMan := app.sessionManager
	logger := app.logger
	notFoundPath := func(path string) bool {
		if path[len(path)-1] == '*' {
			return true
		}
		return false
	}
	return func(c echo.Context) error {
		var (
			path   string = c.Path()
			method string = c.Request().Method
			userID int    = c.Get("userID").(int)
		)
		if banned, err := app.models.Users.Banned(userID); err != nil || banned {
			switch {
			case err != nil:
				return fmt.Errorf("in app#authorize: %w", err)
			case banned:
				sessMan.Put(
					c.Request().Context(),
					"alert",
					components.AlertProps{
						Title: "You are banned!",
						Text:  "Go play somewhere else :)",
						Icon:  components.Error,
					},
				)
				return c.Redirect(http.StatusTemporaryRedirect, "/")
			}
		}
		// Not found paths are echo concept and they
		// do not require permission checks
		if notFoundPath(path) {
			return next(c)
		}
		app.logger.Info("app#authorize", "method", method, "path", path)
		bytes, err := json.Marshal(
			[]map[string]string{{"path": path, "method": method}},
		)
		if err != nil {
			return c.Redirect(http.StatusTemporaryRedirect, "/")
		}
		permission := string(bytes)
		if authorized, err := app.models.Users.Authorized(
			userID,
			permission,
		); err != nil || !authorized {
			switch {
			case err != nil:
				return fmt.Errorf("in app#authorize: %w", err)
			case !authorized:
				logger.Info("app#authorize", "userID", userID)
				logger.Info("app#authorize", "Path", path)
				sessMan.Put(
					c.Request().Context(),
					"alert",
					components.AlertProps{
						Title: "Not Authorized",
						Text:  "You are not authorized to do that!",
						Icon:  components.Error,
					},
				)
				return c.Redirect(http.StatusTemporaryRedirect, "/")
			}
		}
		return next(c)
	}
}

func addHtmxToContext(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		_, ok := c.Request().Header[http.CanonicalHeaderKey("HX-Request")]
		c.Set("HTMX", ok)
		_, ok = c.Request().Header[http.CanonicalHeaderKey("HX-Boosted")]
		c.Set("Boosted", ok)
		return next(c)
	}
}

func (app *application) middleware(e *echo.Echo) {
	e.Pre(middleware.RemoveTrailingSlashWithConfig(trailingSlashConfig))
	e.Use(middleware.RequestLoggerWithConfig(requestLoggerConfig(app.logger)))
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())
	e.Use(middleware.CORSWithConfig(corsConfig(app.config)))
	e.Use(rate_limiter.NewWithConfig(rateLimiterConfig(app.config.env)))
	e.Use(middleware.CSRF())
	e.Use(echo.WrapMiddleware(app.sessionManager.LoadAndSave))
	e.Use(app.userIdExtraction)
	e.Use(app.authorize)
	e.Use(addHtmxToContext)
	e.RouteNotFound("/*", func(c echo.Context) error {
		return views.Render(c, http.StatusNotFound, pages.Page404())
	})
}
