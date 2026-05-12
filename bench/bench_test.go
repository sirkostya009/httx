// Package bench compares dispatch overhead of httx.Mux against julienschmidt/httprouter
// and the Go 1.22+ stdlib http.ServeMux on a few representative route shapes.
//
//	cd bench && go test -bench . -benchmem ./...
package bench

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/julienschmidt/httprouter"
	"github.com/sirkostya009/httx"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func newHTTX(f bool) *httx.Mux {
	m := httx.NewMux()
	m.RedirectTrailingSlash = f
	m.RedirectCaseInsensitivePath = f
	handler := func(w http.ResponseWriter, r *http.Request) error { return nil }
	m.GET("/", handler)
	m.GET("/users", handler)
	m.POST("/users", handler)
	m.GET("/users/{id}", handler)
	m.PUT("/users/{id}", handler)
	m.DELETE("/users/{id}", handler)
	m.GET("/users/{id}/posts", handler)
	m.GET("/users/{id}/posts/{pid}", handler)
	m.GET(`/orders/{id:\d+}`, handler)
	m.GET("/static/{filepath:*}", handler)
	if f {
		// routes registered in canonical form; benchmarks probe wrong-case /
		// trailing-slash variants of these so the case-insensitive + TSR
		// redirect machinery has to fire to produce a 301/308.
		m.GET("/articles/published", handler)
		m.GET("/inbox", handler)
	}
	return m
}

func newHTTPRouter(f bool) *httprouter.Router {
	r := httprouter.New()
	r.RedirectTrailingSlash = f
	r.RedirectFixedPath = f
	handler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {}
	r.GET("/", handler)
	r.GET("/users", handler)
	r.POST("/users", handler)
	r.GET("/users/:id", handler)
	r.PUT("/users/:id", handler)
	r.DELETE("/users/:id", handler)
	r.GET("/users/:id/posts", handler)
	r.GET("/users/:id/posts/:pid", handler)
	r.GET("/orders/:id", handler) // httprouter has no regex
	r.GET("/static/*filepath", handler)
	if f {
		r.GET("/articles/published", handler)
		r.GET("/inbox", handler)
	}
	return r
}

func newGin(f bool) *gin.Engine {
	// gin's default trailing-slash redirect is on by default; mirror the flag.
	r := gin.New()
	r.RedirectTrailingSlash = f
	r.RedirectFixedPath = false // gin has no case-insensitive fix
	handler := func(c *gin.Context) {}
	r.GET("/", handler)
	r.GET("/users", handler)
	r.POST("/users", handler)
	r.GET("/users/:id", handler)
	r.PUT("/users/:id", handler)
	r.DELETE("/users/:id", handler)
	r.GET("/users/:id/posts", handler)
	r.GET("/users/:id/posts/:pid", handler)
	r.GET("/orders/:id", handler) // gin has no regex
	r.GET("/static/*filepath", handler)
	if f {
		r.GET("/articles/published", handler)
		r.GET("/inbox", handler)
	}
	return r
}

func newChi(f bool) *chi.Mux {
	r := chi.NewRouter()
	handler := func(w http.ResponseWriter, r *http.Request) {}
	r.Get("/", handler)
	r.Get("/users", handler)
	r.Post("/users", handler)
	r.Get("/users/{id}", handler)
	r.Put("/users/{id}", handler)
	r.Delete("/users/{id}", handler)
	r.Get("/users/{id}/posts", handler)
	r.Get("/users/{id}/posts/{pid}", handler)
	r.Get(`/orders/{id:[0-9]+}`, handler) // chi supports regex
	r.Get("/static/*", handler)
	if f {
		// chi has no redirect-fix flags; it just won't match these miscased /
		// trailing-slash variants. Registered for parity with the other routers.
		r.Get("/articles/published", handler)
		r.Get("/inbox", handler)
	}
	return r
}

func newStdlib(f bool) *http.ServeMux {
	m := http.NewServeMux()
	handler := func(w http.ResponseWriter, r *http.Request) {}
	m.HandleFunc("GET /", handler) // "/" in stdlib also matches everything; left as-is for parity
	m.HandleFunc("GET /users", handler)
	m.HandleFunc("POST /users", handler)
	m.HandleFunc("GET /users/{id}", handler)
	m.HandleFunc("PUT /users/{id}", handler)
	m.HandleFunc("DELETE /users/{id}", handler)
	m.HandleFunc("GET /users/{id}/posts", handler)
	m.HandleFunc("GET /users/{id}/posts/{pid}", handler)
	m.HandleFunc("GET /orders/{id}", handler) // stdlib has no regex
	m.HandleFunc("GET /static/{filepath...}", handler)
	if f {
		// stdlib has automatic TSR redirects but no case-insensitive fix.
		m.HandleFunc("GET /articles/published", handler)
		m.HandleFunc("GET /inbox", handler)
	}
	return m
}

type routerCase struct {
	name string
	h    http.Handler
}

func routers(f bool) []routerCase {
	return []routerCase{
		{"httx", newHTTX(f)},
		{"httprouter", newHTTPRouter(f)},
		{"chi", newChi(f)},
		{"gin", newGin(f)},
		{"stdlib", newStdlib(f)},
	}
}

var (
	plain = routers(false)
	fix   = routers(true)
	// stdlib / chi / gin have no case-insensitive redirect; left in
	// TrailingSlashFix bench since some of them do TSR.
	caseFix = fix[:3] // httx, httprouter, chi (chi 404s, but kept for parity)
	// httprouter, gin, stdlib have no regex param support; chi and httx do.
	regexOnly = []routerCase{plain[0], plain[2]}
)

func benchmarkPath(b *testing.B, rs []routerCase, method, path string) {
	b.Helper()
	for _, rc := range rs {
		b.Run(rc.name, func(b *testing.B) {
			req := httptest.NewRequest(method, path, nil)
			origPath := req.URL.Path
			w := httptest.NewRecorder()
			var start, end runtime.MemStats
			runtime.ReadMemStats(&start)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// httx and httprouter mutate path on redirects, reset
				req.URL.Path = origPath
				rc.h.ServeHTTP(w, req)
			}
			b.StopTimer()
			runtime.ReadMemStats(&end)
			totalBytes := end.TotalAlloc - start.TotalAlloc
			gcs := end.NumGC - start.NumGC
			b.ReportMetric(float64(end.HeapAlloc)/1024, "heap_KB")
			b.ReportMetric(float64(totalBytes)/1024, "total_KB")
			b.ReportMetric(float64(totalBytes)/float64(b.N), "totB/op")
			b.ReportMetric(float64(gcs)/float64(b.N)*1e6, "gc/Mop")
			b.ReportMetric(float64(gcs), "gc")
		})
	}
}

func BenchmarkSimple(b *testing.B)          { benchmarkPath(b, plain, "GET", "/users") }
func BenchmarkSingleParam(b *testing.B)     { benchmarkPath(b, plain, "GET", "/users/42") }
func BenchmarkMultiParam(b *testing.B)      { benchmarkPath(b, plain, "GET", "/users/42/posts/137") }
func BenchmarkRegexParam(b *testing.B)      { benchmarkPath(b, regexOnly, "GET", "/orders/12345") }
func BenchmarkWildcard(b *testing.B)        { benchmarkPath(b, plain, "GET", "/static/assets/css/main.css") }
func BenchmarkMethodMismatch(b *testing.B)  { benchmarkPath(b, plain, "PATCH", "/users/42") }
func BenchmarkNotFound(b *testing.B)        { benchmarkPath(b, plain, "GET", "/does/not/exist") }
func BenchmarkTrailingSlash(b *testing.B)   { benchmarkPath(b, fix, "GET", "/inbox/") }
func BenchmarkCaseInsensitive(b *testing.B) { benchmarkPath(b, caseFix, "GET", "/ARTICLES/Published/") }
