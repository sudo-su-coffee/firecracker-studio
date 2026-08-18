package web

import (
	"embed"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/valyala/fasthttp"
)

//go:embed dist/* dist/assets/*
var assets embed.FS

// Handler serves the embedded Vue application and delegates API requests to
// the Go Firecracker runtime API. The browser never talks to Firecracker's
// Unix sockets directly.
func Handler(apiHandler func(*fasthttp.RequestCtx)) func(*fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		requestPath := string(ctx.Path())
		if strings.HasPrefix(requestPath, "/api/") {
			apiHandler(ctx)
			return
		}

		assetPath := strings.TrimPrefix(path.Clean(requestPath), "/")
		if assetPath == "" || assetPath == "." {
			assetPath = "index.html"
		}
		body, err := assets.ReadFile("dist/" + assetPath)
		if err != nil {
			// Vue history-mode routes resolve to the SPA entrypoint.
			if !strings.Contains(path.Base(assetPath), ".") {
				body, err = assets.ReadFile("dist/index.html")
			}
		}
		if err != nil {
			ctx.Error(http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		contentType := mime.TypeByExtension(path.Ext(assetPath))
		if contentType == "" {
			contentType = "text/html; charset=utf-8"
		}
		ctx.Response.Header.SetContentType(contentType)
		ctx.SetStatusCode(http.StatusOK)
		ctx.SetBody(body)
	}
}
