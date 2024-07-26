package httx

import (
	"slices"
	"strings"
)

type Group struct {
	mux          *ServeMux
	prefix       string
	middleware   []HandlerFunc
	ErrorHandler ErrorHandler
}

func (g *Group) Pre(middleware ...HandlerFunc) {
	g.middleware = slices.Clip(append(g.middleware, middleware...))
}

func (g *Group) HandleFunc(pattern string, handler HandlerFunc) {
	// todo: reimplement for go 1.23
	if i := strings.IndexRune(pattern, ' '); i == -1 {
		pattern = g.prefix + pattern
	} else {
		pattern = pattern[:i+1] + g.prefix + pattern[i+1:]
	}
	internalHandle(g.mux.ServeMux, pattern, handler, g.middleware, g.ErrorHandler)
}

func (g *Group) Group(pattern string) Group {
	return Group{g.mux, g.prefix + pattern, slices.Clone(g.middleware), g.ErrorHandler}
}
