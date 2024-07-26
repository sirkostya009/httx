package httx

import (
	"net/http"
	"slices"
)

type HandlerFunc func(*Context) error

func (h HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := Context{w, r, r.Context(), nil}
	_ = h(&ctx)
}

type ErrorHandler func(Context, error)

type ServeMux struct {
	*http.ServeMux
	ErrorHandler ErrorHandler
	middleware   []HandlerFunc
}

func DefaultErrorHandler(ctx Context, err error) {
	http.Error(ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
}

func NewServeMux(handler ...*http.ServeMux) *ServeMux {
	var httpMux *http.ServeMux
	if len(handler) > 0 {
		httpMux = handler[0]
	} else {
		httpMux = http.NewServeMux()
	}
	return &ServeMux{httpMux, DefaultErrorHandler, nil}
}

func (m *ServeMux) Pre(middleware ...HandlerFunc) {
	m.middleware = slices.Clip(append(m.middleware, middleware...))
}

func (m *ServeMux) HandleFunc(pattern string, handler HandlerFunc) {
	internalHandle(m.ServeMux, pattern, handler, m.middleware, m.ErrorHandler)
}

func internalHandle(mux *http.ServeMux, pattern string, handler HandlerFunc, middleware []HandlerFunc, errHandler ErrorHandler) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		ctx := Context{w, r, r.Context(), nil}
		//r = r.WithContext(&ctx)
		for _, mw := range middleware {
			if err := mw(&ctx); err != nil {
				errHandler(ctx, err)
				return
			}
		}
		err := handler(&ctx)
		if err != nil {
			errHandler(ctx, err)
		}
	})
}

func (m *ServeMux) Group(prefix string) Group {
	return Group{m, prefix, slices.Clone(m.middleware), m.ErrorHandler}
}
