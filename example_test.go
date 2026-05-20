package httx_test

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/sirkostya009/httx"
)

type ExampleServer struct {
	/// some injected deps
	r *httx.Mux
}

type customWriter struct {
	http.ResponseWriter
	status int
}

func (w *customWriter) WriteHeader(statusCode int) {
	if w.status == 0 {
		w.status = statusCode
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

var cwPool = sync.Pool{New: func() any {
	return &customWriter{}
}}

func (s *ExampleServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cw := cwPool.Get().(*customWriter)
	defer cwPool.Put(cw)
	*cw = customWriter{w, 0}

	start := time.Now()
	s.r.ServeHTTP(cw, r)
	end := time.Now()

	if cw.status == 0 {
		cw.status = 200
	}

	logger := slog.InfoContext
	if cw.status >= 400 {
		logger = slog.ErrorContext
	}

	logger(r.Context(),
		"incoming",
		slog.Int("code", cw.status),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Duration("time", end.Sub(start)))
}

func Example() {
	mux := httx.NewMux()
	mux.OnError = func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(err.Error()))
	}

	requests := 0

	mux.Use(func(next httx.HandlerFunc) httx.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) error {
			requests++
			return next(w, r)
		}
	})

	mux.GET("/error", func(w http.ResponseWriter, r *http.Request) error {
		return errors.New("oops!")
	})

	mux.GET(`/{id:\d{1,23}}`, func(w http.ResponseWriter, r *http.Request) error {
		id, _ := strconv.Atoi(r.PathValue("id")) // Go's 1.22 PathValue-compatible, regex ensures id is a valid int
		_, err := someCtxFunc(r.Context(), id)
		if err != nil {
			return err
		}
		return json.NewEncoder(w).Encode(requests)
	})

	httpSrv := &http.Server{
		Addr:    ":2235",
		Handler: &ExampleServer{r: mux},
		// some other optional config
	}

	log.Fatal(httpSrv.ListenAndServe())
}

func someCtxFunc(context.Context, int) (any, error) {
	return nil, nil
}
