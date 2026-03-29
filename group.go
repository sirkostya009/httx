package httx

import (
	"io/fs"
	"net/http"
	"slices"
	"strings"
)

type Group struct {
	prefix string
	m      *Mux
	mw     []func(HandlerFunc) HandlerFunc
}

func (g *Group) Group(prefix string) *Group {
	if prefix == "" {
		panic("group prefix must not be empty")
	}
	if len(prefix) > 1 && strings.HasSuffix(prefix, "/") {
		panic("group prefix must not be empty")
	}
	if !strings.HasPrefix(prefix, "/") {
		panic(`group prefix must begin with "/"`)
	}
	return &Group{g.prefix + prefix, g.m, slices.Clip(g.mw)}
}

func (g *Group) Use(mw ...func(HandlerFunc) HandlerFunc) {
	// clipping ensures we don't modify the original mw array in Merge
	g.mw = slices.Clip(append(g.mw, mw...))
}

func (g *Group) Handle(method, path string, handler HandlerFunc) {
	if !strings.HasPrefix(path, "/") {
		panic(`group path must begin with "/"`)
	}
	if path == "" {
		panic("path must not be empty")
	}
	temp := g.m.mw
	g.m.mw = g.mw
	g.m.Handle(method, g.prefix+path, handler)
	g.m.mw = temp
}

func (g *Group) GET(path string, handler HandlerFunc) {
	g.Handle(http.MethodGet, path, handler)
}

func (g *Group) POST(path string, handler HandlerFunc) {
	g.Handle(http.MethodPost, path, handler)
}

func (g *Group) PUT(path string, handler HandlerFunc) {
	g.Handle(http.MethodPut, path, handler)
}

func (g *Group) PATCH(path string, handler HandlerFunc) {
	g.Handle(http.MethodPatch, path, handler)
}

func (g *Group) DELETE(path string, handler HandlerFunc) {
	g.Handle(http.MethodDelete, path, handler)
}

func (g *Group) HEAD(path string, handler HandlerFunc) {
	g.Handle(http.MethodHead, path, handler)
}

func (g *Group) CONNECT(path string, handler HandlerFunc) {
	g.Handle(http.MethodConnect, path, handler)
}

func (g *Group) OPTIONS(path string, handler HandlerFunc) {
	g.Handle(http.MethodOptions, path, handler)
}

func (g *Group) TRACE(path string, handler HandlerFunc) {
	g.Handle(http.MethodTrace, path, handler)
}

func (g *Group) ANY(path string, handler HandlerFunc) {
	g.Handle(MethodWild, path, handler)
}

func (g *Group) Merge(path string, handler http.Handler) {
	temp := g.m.mw
	g.m.mw = g.mw
	g.m.Merge(g.prefix+path, handler)
	g.m.mw = temp
}

func (g *Group) FS(path string, f fs.FS) {
	g.FileSystem(path, http.FS(f))
}

func (g *Group) FileSystem(path string, f http.FileSystem) {
	g.Merge(path, fswrap(http.FileServer(f)))
}
