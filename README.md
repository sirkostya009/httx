# HTTp eXtended

Fork of [fasthttp/router](https://github.com/fasthttp/router/) adapted for `http.Handler`. 0 dependencies.

Main algorithm and ergonomics largely borrowed from `fasthttp/router` with a few additions and performance
optimizations.

All standard features present in `fasthttp/router` are present here as well.

Simple routes do 0 allocations. Redirects and case-insensitive path fixing redirects allocate a few bytes,
latter allocates only via a pool. Also, RedirectCaseInsensitivePath (RedirectFixedPath in `fasthttp/router`)
doesn't resolve incoming relative paths (`http.Server` already deals with that).

Regex is most expensive due to inefficiencies in stdlib's `regexp` implementation.

If you're not using `http.Server` and need path resolution, I can only suggest implementing a basic implemention
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
over ~260-route deeply-nested REST surface. See [bench_test.go](bench/bench_test.go).

```
Simple                                 GET /healthz
  httx          18 ns/op     0 B/op    0 allocs/op
  httprouter    20 ns/op     0 B/op    0 allocs/op
  chi          366 ns/op   368 B/op    2 allocs/op
  gin           39 ns/op     0 B/op    0 allocs/op
  net/http      68 ns/op     0 B/op    0 allocs/op

SingleParam                            GET /api/v2/sessions/{sessionId}
  httx          53 ns/op     0 B/op    0 allocs/op
  httprouter    80 ns/op    32 B/op    1 allocs/op
  chi          665 ns/op   704 B/op    4 allocs/op
  gin           62 ns/op     0 B/op    0 allocs/op
  net/http     221 ns/op    16 B/op    1 allocs/op

MultiParam                             GET .../repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff
  httx         188 ns/op     0 B/op    0 allocs/op
  httprouter   240 ns/op   192 B/op    1 allocs/op
  chi          989 ns/op   704 B/op    4 allocs/op
  gin          163 ns/op     0 B/op    0 allocs/op
  net/http     917 ns/op   240 B/op    4 allocs/op

RegexParam                             GET /api/v1/orders/{orderId:\d+}/lines/{lineNo:\d+}
  httx         440 ns/op    96 B/op    4 allocs/op
  chi          927 ns/op   704 B/op    4 allocs/op

Wildcard                               GET .../commits/{commitSha}/files/{filepath:*}
  httx         212 ns/op     0 B/op    0 allocs/op
  httprouter   223 ns/op   192 B/op    1 allocs/op
  chi         1043 ns/op   704 B/op    4 allocs/op
  gin          125 ns/op     0 B/op    0 allocs/op
  net/http    1520 ns/op   608 B/op    9 allocs/op

MethodMismatch                         OPTIONS .../payment_methods/{pmId}/transactions/{txnId}
  httx         461 ns/op    96 B/op    1 allocs/op
  httprouter  1197 ns/op   704 B/op    7 allocs/op
  chi          602 ns/op   368 B/op    2 allocs/op
  gin          581 ns/op   307 B/op    4 allocs/op
  net/http    6127 ns/op  2627 B/op   75 allocs/op

NotFound                               GET /api/v2/does/not/exist/at/all
  httx         204 ns/op     0 B/op    0 allocs/op
  httprouter   226 ns/op     0 B/op    0 allocs/op
  chi          374 ns/op   368 B/op    2 allocs/op
  gin          244 ns/op   131 B/op    1 allocs/op
  net/http     458 ns/op   128 B/op    7 allocs/op

TrailingSlash                          GET /inbox/
  httx          55 ns/op     0 B/op    0 allocs/op
  httprouter   321 ns/op   184 B/op    3 allocs/op
  gin          569 ns/op   280 B/op    8 allocs/op
  net/http     122 ns/op    16 B/op    1 allocs/op

CaseInsensitive                        GET /ARTICLES/Published/
  httx          95 ns/op     0 B/op    0 allocs/op
  httprouter   510 ns/op   216 B/op    4 allocs/op
  gin          760 ns/op   408 B/op    8 allocs/op
```

Due do compatibility with stdlib's `net/http.Request.SetPathValue` performance on param paths takes a hit,
especially the wildcard. Raw performance of `gin`, `httprouter` and `httx` is the same as the underlying algorithm
is the same :).

`gin`, `httprouter` and `net/http` don't support regex param validation. `chi` and `net/http` don't redirect on case-mismatched paths. `chi` doesn't redirect on trailing-slash mismatches either.

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
