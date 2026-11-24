package adminpage

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/zeromicro/go-zero/rest"
)

const routePrefix = "/plugins/telegram_chat_bot/admin"

//go:embed all:assets/dist
var public embed.FS

func RegisterHandlers(server *rest.Server) {
	dist, err := fs.Sub(public, "assets/dist")
	if err != nil {
		panic(err)
	}
	staticFiles := http.StripPrefix(routePrefix, http.FileServer(http.FS(dist)))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		staticFiles.ServeHTTP(w, r)
	})
	server.AddRoute(rest.Route{Method: http.MethodGet, Path: routePrefix + "/", Handler: handler.ServeHTTP})
	if err := fs.WalkDir(dist, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || name == "index.html" {
			return nil
		}
		server.AddRoute(rest.Route{Method: http.MethodGet, Path: routePrefix + "/" + name, Handler: handler.ServeHTTP})
		return nil
	}); err != nil {
		panic(err)
	}
}
