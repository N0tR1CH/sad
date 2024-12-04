package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
	"github.com/h2non/bimg"
)

type Services struct {
	ChromeDp interface {
		GenScreenshot(url string) (string, error)
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

func (cds ChromeDpService) GenScreenshot(url string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	var buf []byte

	if err := chromedp.Run(ctx, mobileScreenshot(url, &buf)); err != nil {
		cds.logger.Error("CHROMEDPSERVICE", "err", err)
		return "", err
	}

	resPath := fmt.Sprintf("/public/%s.webp", uuid.NewString())
	filepath := fmt.Sprintf("cmd/web%s", resPath)
	if err := os.WriteFile(filepath, buf, 0o644); err != nil {
		cds.logger.Error("CHROMEDPSERVICE", "err", err)
		return "", err
	}

	buffer, err := bimg.Read(filepath)
	if err != nil {
		cds.logger.Error("CHROMEDPSERVICE", "err", err)
		return "", err
	}

	imgConvertedToWebp, err := bimg.NewImage(buffer).Convert(bimg.WEBP)
	if err != nil {
		cds.logger.Error("CHROMEDPSERVICE", "err", err)
		return "", err
	}

	imgCompressed, err := bimg.NewImage(imgConvertedToWebp).Process(
		bimg.Options{Quality: 1},
	)
	if err != nil {
		cds.logger.Error("CHROMEDPSERVICE", "err", err)
		return "", err
	}

	bimg.Write(filepath, imgCompressed)

	return resPath, nil
}

func mobileScreenshot(urlstr string, res *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate(urlstr),
		chromedp.EmulateViewport(
			1920,
			1080,
			chromedp.EmulateOrientation(
				emulation.OrientationTypePortraitPrimary,
				0,
			),
		),
		chromedp.CaptureScreenshot(res),
	}
}
