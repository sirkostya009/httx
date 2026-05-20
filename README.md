# HTTp eXtended

Fork of [fasthttp/router](https://github.com/fasthttp/router/) adapted for `http.Handler`. 0 dependencies.

Main algorithm and ergonomics largely borrowed from `fasthttp/router` with a few additions and performance
optimizations.

All standard features present in `fasthttp/router` are present here as well.

Simple routes do 0 allocations. Redirects and case-insensitive path fixing redirects allocate a few bytes,
latter allocates only via a pool. Also, RedirectCaseInsensitivePath (RedirectFixedPath in `fasthttp/router`)
doesn't resolve incoming relative paths (`http.Server` already deals with that).

Regex is most expensive due to inefficiencies in stdlib's `regexp` implementation.

If you're not using `http.Server` and need path resolution, I can only suggest copying a basic implemention
from [Appendix](#appendix).

## Usage

```go
mux := httx.NewMux()

mux.OnError = func(w http.ResponseWriter, r *http.Request, err error) {
	// handle err
}

// Middleware must be initialized before any route
mux.Use(func(next httx.HandlerFunc) httx.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) error {
		start := time.Now()
		defer func() { // must defer stuff running after because panics
			finish := time.Now()
			slog.Info("request", "method", r.Method, "uri", r.RequestURI, "time-ms", finish.Sub(start).Milliseconds())
		}()
		return next(w, r)
	}
})

mux.GET("/hello", func(w http.ResponseWriter, r *http.Request) error {
	_, err := w.Write([]byte("world!"))
	return err
})

mux.GET(`/{id:\d+}`, func(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id") // Go's 1.22 PathValue-compatible
	res, err := someDatabaseFunc(r.Context(), id)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(res)
})

_ = http.ListenAndServe(":8080", mux)
```

See [example_test.go](./example_test.go) for a more interesting example.

## Benchmarks

Comparison against `httprouter`, `chi`, `gin`, and `net/http` on AMD Ryzen AI Max+ 395. Run with `-benchtime=1000000x`
over a ~260-route deeply-nested REST surface. Each bench cycles through 64 distinct (method, path) hits generated
from a fixed seed — no single hot URL. See [bench_test.go](bench/bench_test.go).

```
Simple                                 {GET,POST,PUT,DELETE,PATCH} {/, /healthz, /livez, /readyz, /metrics}
  httx          16 ns/op     0 B/op    0 allocs/op
  httprouter    24 ns/op     0 B/op    0 allocs/op
  chi          400 ns/op   368 B/op    2 allocs/op
  gin           53 ns/op     0 B/op    0 allocs/op
  net/http      86 ns/op     0 B/op    0 allocs/op

SingleParam                            {*} /api/v{1..3}/sessions/{sessionId}
  httx          75 ns/op     0 B/op    0 allocs/op
  httprouter   104 ns/op    32 B/op    1 allocs/op
  chi          707 ns/op   704 B/op    4 allocs/op
  gin           78 ns/op     0 B/op    0 allocs/op
  net/http     252 ns/op    16 B/op    1 allocs/op

MultiParam                             {*} /api/v{1..3}/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
  httx         243 ns/op     0 B/op    0 allocs/op
  httprouter   272 ns/op   192 B/op    1 allocs/op
  chi         1009 ns/op   704 B/op    4 allocs/op
  gin          183 ns/op     0 B/op    0 allocs/op
  net/http     922 ns/op   240 B/op    4 allocs/op

RegexParam                             {*} /api/v{1..3}/orders/{orderId:\d+}/lines/{lineNo:\d+}
  httx         518 ns/op    96 B/op    4 allocs/op
  chi          976 ns/op   704 B/op    4 allocs/op

Wildcard                               {*} /api/v{1..3}/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath:*}
  httx         279 ns/op     0 B/op    0 allocs/op
  httprouter   289 ns/op   192 B/op    1 allocs/op
  chi         1078 ns/op   704 B/op    4 allocs/op
  gin          191 ns/op     0 B/op    0 allocs/op
  net/http    1868 ns/op   668 B/op    9 allocs/op

MethodMismatch                         {OPTIONS,TRACE} /api/v{1..3}/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}
  httx         564 ns/op    80 B/op    1 allocs/op
  httprouter  1273 ns/op   704 B/op    7 allocs/op
  chi          634 ns/op   368 B/op    2 allocs/op
  gin          588 ns/op   307 B/op    4 allocs/op
  net/http    6448 ns/op  2627 B/op   75 allocs/op

NotFound                               {*} {/api/v9,/admin,/totally,/does/not,/api/v2/missing}/{random}/{random}
  httx         156 ns/op     0 B/op    0 allocs/op
  httprouter   221 ns/op     0 B/op    0 allocs/op
  chi          380 ns/op   368 B/op    2 allocs/op
  gin          245 ns/op   131 B/op    1 allocs/op
  net/http     377 ns/op    82 B/op    4 allocs/op

TrailingSlash                          {*} {/inbox/, /articles/published/}
  httx         148 ns/op     0 B/op    0 allocs/op
  httprouter   329 ns/op    41 B/op    0 allocs/op
  gin          416 ns/op   248 B/op    3 allocs/op

CaseInsensitive                        {*} {/ARTICLES/Published/, /Articles/published/, /INBOX/, /Inbox/}
  httx         150 ns/op     0 B/op    0 allocs/op
  httprouter   345 ns/op    45 B/op    0 allocs/op
  gin          399 ns/op   272 B/op    3 allocs/op
```

`{*}` = all 5 methods registered (GET/POST/PUT/DELETE/PATCH) cycled. Where router has a single Allow value (e.g. MethodMismatch), only the method-mismatched verbs are tried.

Due to compatibility with stdlib's `net/http.Request.SetPathValue` performance on param paths takes a hit,
especially the wildcard. Raw performance of `gin`, `httprouter` and `httx` trees is the same as the underlying
algorithm is the same :).

`gin`, `httprouter` and `net/http` don't support regex param validation. `chi` and `net/http` don't redirect on case-mismatched paths. `chi` doesn't redirect on trailing-slash mismatches at all; `net/http` redirects `/foo` → `/foo/` only when `/foo/` is the registered route (one-direction).

All routers configured with status-only 404/405 handlers (no body) so miss-path numbers reflect dispatch cost, not default-body writing. `net/http` can't be customized this way — its 404 path always writes `"404 page not found\n"`.

## License

The original BSD 3-clause license from [fasthttp/router](https://github.com/fasthttp/router/blob/master/LICENSE). See [LICENSE](LICENSE).

## Appendix

### `ResolvePath` function:

```go
var base, _ = url.Parse("/")

func ResolvePath(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL = base.ResolveReference(r.URL)
		h.ServeHTTP(w, r)
	}
}
```
