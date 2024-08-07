package web

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
)

//go:embed assets
var staticFiles embed.FS

func GetFileSystem(useOs bool, logger *slog.Logger) http.FileSystem {
	if useOs {
		logger.Info("Using live mode")
		return http.FS(os.DirFS("assets"))
	}
	logger.Info("Using embed mode")
	fsys, err := fs.Sub(staticFiles, "assets")
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}
