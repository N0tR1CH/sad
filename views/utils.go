package views

import (
	"io"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"golang.org/x/net/context"
)

func Render(c echo.Context, statusCode int, t templ.Component) error {
	buf := templ.GetBuffer()
	defer templ.ReleaseBuffer(buf)

	ctx := c.Request().Context()
	if _, ok := c.Get("activationSuccess").(struct{}); ok {
		ctx = context.WithValue(ctx, "activationSuccess", struct{}{})
	}
	if token, ok := c.Get("csrf").(string); ok {
		ctx = context.WithValue(ctx, "csrf", token)
	}
	if id, ok := c.Get("userID").(int); ok {
		ctx = context.WithValue(ctx, "userID", id)
	}
	if err := t.Render(ctx, buf); err != nil {
		return err
	}

	return c.HTML(statusCode, buf.String())
}

func Unsafe(html string) templ.Component {
	cf := func(ctx context.Context, w io.Writer) (err error) {
		_, err = io.WriteString(w, html)
		return
	}
	return templ.ComponentFunc(cf)
}
