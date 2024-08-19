package httx

import (
	"net/http"
)

var DefaultServeMux = NewServeMux(http.DefaultServeMux)

func Pre(middleware ...HandlerFunc) {
	DefaultServeMux.Pre(middleware...)
}

func HandleFunc(pattern string, handler HandlerFunc) {
	internalHandle(http.DefaultServeMux, pattern, handler, DefaultServeMux.middleware, DefaultServeMux.ErrorHandler)
}

func Group(prefix string) MuxGroup {
	return MuxGroup{DefaultServeMux, prefix, DefaultServeMux.middleware, DefaultServeMux.ErrorHandler}
}
