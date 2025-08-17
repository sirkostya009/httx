package httx

import (
	"net/http"
	"strings"
)

type Group struct {
	prefix string
	m      *Mux
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
	return &Group{g.prefix + prefix, g.m}
}

func (g *Group) Handle(method, path string, handler HandlerFunc) {
	if !strings.HasPrefix(path, "/") {
		panic(`group path must begin with "/"`)
	}
	if path == "" {
		panic("path must not be empty")
	}
	g.m.Handle(method, g.prefix+path, handler)
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
	g.m.Merge(g.prefix+path, handler)
}
