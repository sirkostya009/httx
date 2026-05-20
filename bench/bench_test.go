// Package bench compares dispatch overhead of httx.Mux against julienschmidt/httprouter,
// go-chi/chi, gin-gonic/gin and the Go 1.22+ stdlib http.ServeMux on a realistic
// deeply-nested REST surface.
//
//	cd bench && go test -bench . -benchmem ./...
package bench

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/julienschmidt/httprouter"
	"github.com/sirkostya009/httx"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

// Realistic deeply-nested API surface modeled after GitHub/GitLab/AWS-style services.
// Each template is registered with multiple HTTP methods.
var deepTemplates = []string{
	"/api/v{ver}/organizations/{orgId}/projects/{projectId}",
	"/api/v{ver}/organizations/{orgId}/projects/{projectId}/repositories/{repoId}",
	"/api/v{ver}/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}",
	"/api/v{ver}/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff",
	"/api/v{ver}/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath:*}",
	"/api/v{ver}/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}",
	"/api/v{ver}/organizations/{orgId}/teams/{teamSlug}/members/{userId}",
	"/api/v{ver}/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}",
	"/api/v{ver}/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}",
	"/api/v{ver}/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}",
	"/api/v{ver}/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}",
	"/api/v{ver}/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}",
	"/api/v{ver}/datasets/{datasetId}/tables/{tableId}/columns/{columnId}",
	"/api/v{ver}/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}",
	"/api/v{ver}/webhooks/{whId}/deliveries/{deliveryId}",
	"/api/v{ver}/integrations/{provSlug}/connections/{connId}/syncs/{syncId}",
	`/api/v{ver}/orders/{orderId:\d+}/lines/{lineNo:\d+}`,
	"/api/v{ver}/sessions/{sessionId}",
}

var topLevel = []string{"/", "/healthz", "/livez", "/readyz", "/metrics"}

// braceToColon converts httx/chi/stdlib brace syntax to gin/httprouter colon syntax.
func braceToColon(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '{' {
			i++
			isWild := false
			start := i
			for i < len(s) && s[i] != '}' && s[i] != ':' {
				i++
			}
			name := s[start:i]
			if i < len(s) && s[i] == ':' {
				i++
				if i < len(s) && s[i] == '*' {
					isWild = true
					i++
				}
				for i < len(s) && s[i] != '}' {
					i++
				}
			}
			i++
			if isWild {
				b.WriteByte('*')
			} else {
				b.WriteByte(':')
			}
			b.WriteString(name)
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// braceToStdlib converts brace-with-regex syntax to net/http syntax
// (drops regex, converts :* catchall to {name...}).
func braceToStdlib(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '{' {
			b.WriteByte('{')
			i++
			start := i
			for i < len(s) && s[i] != '}' && s[i] != ':' {
				i++
			}
			b.WriteString(s[start:i])
			if i < len(s) && s[i] == ':' {
				i++
				if i < len(s) && s[i] == '*' {
					b.WriteString("...")
					i++
				}
				for i < len(s) && s[i] != '}' {
					i++
				}
			}
			if i < len(s) && s[i] == '}' {
				b.WriteByte('}')
				i++
			}
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// braceToChi strips the wildcard ":*" suffix chi expects bare "*".
func braceToChi(s string) string {
	if before, _, ok := strings.Cut(s, ":*}"); ok {
		// "/static/{filepath:*}" -> "/static/*"
		j := strings.LastIndexByte(before, '{')
		if j >= 0 {
			return s[:j] + "*"
		}
	}
	return s
}

type tmpl struct{ brace, colon, stdlib, chi string }

func buildTemplates() []tmpl {
	all := make([]tmpl, 0, 256)
	for _, p := range topLevel {
		all = append(all, tmpl{p, p, p, p})
	}
	for ver := 1; ver <= 3; ver++ {
		for _, t := range deepTemplates {
			brace := strings.ReplaceAll(t, "{ver}", strconv.Itoa(ver))
			all = append(all, tmpl{brace, braceToColon(brace), braceToStdlib(brace), braceToChi(brace)})
		}
	}
	return all
}

// methods is the set of methods we register each template with — exercises 405
// detection and creates realistic per-resource handler counts.
var methods = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

// statusOnly returns an http.HandlerFunc that writes the given status code and
// no body. We use this to equalize miss-path callbacks across routers so the
// bench measures dispatch overhead and not the cost of each router's default
// "404 page not found\n"-style body writing.
func statusOnly(code int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
	}
}

func newHTTX(f bool) *httx.Mux {
	m := httx.NewMux()
	m.RedirectTrailingSlash = f
	m.RedirectCaseInsensitivePath = f
	h := func(w http.ResponseWriter, r *http.Request) error { return nil }
	for _, t := range buildTemplates() {
		for _, meth := range methods {
			m.Handle(meth, t.brace, h)
		}
	}
	if f {
		m.GET("/articles/published", h)
		m.GET("/inbox", h)
	}
	return m
}

func newHTTPRouter(f bool) *httprouter.Router {
	r := httprouter.New()
	r.RedirectTrailingSlash = f
	r.RedirectFixedPath = f
	r.NotFound = statusOnly(404)
	r.MethodNotAllowed = statusOnly(405)
	h := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {}
	for _, t := range buildTemplates() {
		path := t.colon
		// httprouter doesn't support regex, strip it
		path = stripRegex(path)
		for _, meth := range methods {
			r.Handle(meth, path, h)
		}
	}
	if f {
		r.GET("/articles/published", h)
		r.GET("/inbox", h)
	}
	return r
}

func newGin(f bool) *gin.Engine {
	r := gin.New()
	r.RedirectTrailingSlash = f
	r.RedirectFixedPath = f
	r.HandleMethodNotAllowed = true
	r.NoRoute(func(c *gin.Context) { c.Writer.WriteHeader(404) })
	r.NoMethod(func(c *gin.Context) { c.Writer.WriteHeader(405) })
	h := func(c *gin.Context) {}
	for _, t := range buildTemplates() {
		path := stripRegex(t.colon)
		for _, meth := range methods {
			r.Handle(meth, path, h)
		}
	}
	if f {
		r.GET("/articles/published", h)
		r.GET("/inbox", h)
	}
	return r
}

func newChi(f bool) *chi.Mux {
	r := chi.NewRouter()
	r.NotFound(statusOnly(404))
	r.MethodNotAllowed(statusOnly(405))
	h := func(w http.ResponseWriter, r *http.Request) {}
	for _, t := range buildTemplates() {
		for _, meth := range methods {
			r.Method(meth, t.chi, http.HandlerFunc(h))
		}
	}
	if f {
		r.Get("/articles/published", h)
		r.Get("/inbox", h)
	}
	return r
}

func newStdlib(f bool) *http.ServeMux {
	m := http.NewServeMux()
	h := func(w http.ResponseWriter, r *http.Request) {}
	for _, t := range buildTemplates() {
		path := stripRegex(t.stdlib)
		for _, meth := range methods {
			m.HandleFunc(meth+" "+path, h)
		}
	}
	if f {
		m.HandleFunc("GET /articles/published", h)
		m.HandleFunc("GET /inbox", h)
	}
	return m
}

// stripRegex removes ":<pattern>" from inside braces, leaving just {name}.
// For routers that don't support regex (httprouter, gin, stdlib).
func stripRegex(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '{' {
			j := strings.IndexByte(s[i:], '}')
			if j < 0 {
				b.WriteString(s[i:])
				break
			}
			seg := s[i : i+j+1]
			if before, _, ok := strings.Cut(seg, ":"); ok {
				b.WriteString(before)
				b.WriteByte('}')
			} else {
				b.WriteString(seg)
			}
			i += j + 1
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
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
		{"net/http", newStdlib(f)},
	}
}

var (
	plain = routers(false)
	fix   = routers(true)
	// chi does no redirects (verified: 404s on both case and slash mismatches).
	tsrOnly = []routerCase{fix[0], fix[1], fix[3], fix[4]} // httx, httprouter, gin, net/http
	// stdlib + chi have no case-insensitive redirect.
	caseFix = []routerCase{fix[0], fix[1], fix[3]} // httx, httprouter, gin
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

			// warm caches + lazy init (regex machine pools, etc.)
			for range 100 {
				req.URL.Path = origPath
				rc.h.ServeHTTP(w, req)
			}

			var start, end runtime.MemStats
			runtime.ReadMemStats(&start)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
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

// Realistic hit paths exercising varied depth and param counts.

func BenchmarkSimple(b *testing.B) {
	// hits /healthz — shallow static, no params
	benchmarkPath(b, plain, "GET", "/healthz")
}

func BenchmarkSingleParam(b *testing.B) {
	// hits /api/v{ver}/sessions/{sessionId} — 1 param
	benchmarkPath(b, plain, "GET", "/api/v2/sessions/sess-7f3e2d1c8b9a")
}

func BenchmarkMultiParam(b *testing.B) {
	// hits /api/v{ver}/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff — 5 params, deep
	benchmarkPath(b, plain, "GET", "/api/v2/organizations/acme-corp/projects/widget-svc/repositories/main-monorepo/branches/release-2026.04/commits/3a5f9d2b8e1c4f6a9d8b7c2e5f1a3b9c8d7e6f5a/diff")
}

func BenchmarkRegexParam(b *testing.B) {
	// hits /api/v{ver}/orders/{orderId:\d+}/lines/{lineNo:\d+} — 2 regex-validated params
	benchmarkPath(b, regexOnly, "GET", "/api/v1/orders/12345/lines/678")
}

func BenchmarkWildcard(b *testing.B) {
	// hits .../commits/{commitSha}/files/{filepath:*} — catchall at depth, 5 params + tail
	benchmarkPath(b, plain, "GET", "/api/v3/organizations/acme/projects/x/repositories/r/branches/main/commits/abc/files/src/internal/auth/middleware/oidc.go")
}

func BenchmarkMethodMismatch(b *testing.B) {
	// /api/v{ver}/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId} is registered for
	// GET/POST/PUT/DELETE/PATCH but not OPTIONS — exercises 405 + Allow-header build
	benchmarkPath(b, plain, "OPTIONS", "/api/v2/billing/accounts/acct-7f3e2d1c/payment_methods/pm-9a8b/transactions/txn-456")
}

func BenchmarkNotFound(b *testing.B) {
	// /api/v2/does/not/exist/at/all is not registered under any method — full miss-path walk
	benchmarkPath(b, plain, "GET", "/api/v2/does/not/exist/at/all")
}

func BenchmarkTrailingSlash(b *testing.B) {
	// /inbox is registered, request has trailing slash — 301/308 redirect to /inbox
	benchmarkPath(b, tsrOnly, "GET", "/inbox/")
}

func BenchmarkCaseInsensitive(b *testing.B) {
	// /articles/published is registered, request is wrong-cased + trailing slash — redirect
	benchmarkPath(b, caseFix, "GET", "/ARTICLES/Published/")
}
