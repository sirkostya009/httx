package tinymux

import "net/http"

type Handler func(*Context) error

type StandardMultiplexer interface {
	HandleFunc(string, func(http.ResponseWriter, *http.Request))
	Handle(pattern string, handler http.Handler)
	Handler(r *http.Request) (http.Handler, string)
	http.Handler
}

type Mux struct {
	StandardMultiplexer
	ErrorHandler func(*Context, error)
	middleware   []Handler
}

func DefaultErrorHandler(ctx *Context, err error) {
	http.Error(ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
}

func NewMux(mux StandardMultiplexer) *Mux {
	if mux == nil {
		mux = http.NewServeMux()
	}
	return &Mux{mux, DefaultErrorHandler, nil}
}

func (m *Mux) HandleHttp(pattern string, handler Handler) {
	m.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		ctx := &Context{w, r, r.Context(), map[any]any{}}
		for _, mw := range m.middleware {
			err := mw(ctx)
			if err != nil {
				m.ErrorHandler(ctx, err)
				return
			}
		}
		err := handler(ctx)
		if err != nil {
			m.ErrorHandler(ctx, err)
		}
	})
}

func (m *Mux) Pre(ph ...Handler) {
	m.middleware = append(m.middleware, ph...)
}
