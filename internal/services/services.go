package services

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Services struct {
	ChromeDp interface {
		GenScreenshot(c echo.Context, url string) (string, error)
	}
}

func NewServices(logger *slog.Logger) Services {
	return Services{
		ChromeDp: ChromeDpService{
			logger,
		},
	}
}

type ChromeDpService struct {
	logger *slog.Logger
}

func (cds ChromeDpService) GenScreenshot(c echo.Context, url string) (string, error) {
	ctx, cancel := chromedp.NewContext(c.Request().Context())
	defer cancel()

	var buf []byte

	if err := chromedp.Run(ctx, mobileScreenshot(url, &buf)); err != nil {
		cds.logger.Error("CHROMEDPSERVICE", "err", err)
		return "", err
	}

	resPath := fmt.Sprintf("/public/%s.png", uuid.NewString())
	if err := os.WriteFile(fmt.Sprintf("cmd/web%s", resPath), buf, 0o644); err != nil {
		cds.logger.Error("CHROMEDPSERVICE", "err", err)
		return "", err
	}

	return resPath, nil
}

func mobileScreenshot(urlstr string, res *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate(urlstr),
		chromedp.EmulateViewport(
			375,
			667,
			chromedp.EmulateOrientation(
				emulation.OrientationTypePortraitPrimary,
				0,
			),
		),
		chromedp.CaptureScreenshot(res),
	}
}
