package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var distFiles embed.FS

// StaticFS 返回前端静态文件的 http.FileSystem
func StaticFS() (http.FileSystem, error) {
	sub, err := fs.Sub(distFiles, "dist")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}

// DistFS 返回前端静态文件的 fs.FS
func DistFS() (fs.FS, error) {
	return fs.Sub(distFiles, "dist")
}

