package httx

import (
	"slices"
	"strings"
)

type MuxGroup struct {
	mux          *ServeMux
	prefix       string
	middleware   []HandlerFunc
	ErrorHandler ErrorHandler
}

func (g *MuxGroup) Pre(middleware ...HandlerFunc) {
	g.middleware = slices.Clip(append(g.middleware, middleware...))
}

func (g *MuxGroup) HandleFunc(pattern string, handler HandlerFunc) {
	// todo: reimplement for go 1.23
	if i := strings.IndexRune(pattern, ' '); i == -1 {
		pattern = g.prefix + pattern
	} else {
		pattern = pattern[:i+1] + g.prefix + pattern[i+1:]
	}
	internalHandle(g.mux.ServeMux, pattern, handler, g.middleware, g.ErrorHandler)
}

func (g *MuxGroup) Group(pattern string) MuxGroup {
	return MuxGroup{g.mux, g.prefix + pattern, g.middleware, g.ErrorHandler}
}
