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

Comparison against `httprouter`, `chi`, `gin`, and `net/http` on AMD Ryzen AI Max+ 395. Run with `-benchtime=10000x` with a warm up run. See [bench_test.go](bench/bench_test.go).

`gin` and `net/http` don't support regex param validation. `chi` and `net/http` don't redirect on case-mismatched paths. `chi` doesn't redirect on trailing-slash mismatches either.

```
Simple
  httx          19 ns/op     0 B/op    0 allocs/op
  httprouter    21 ns/op     0 B/op    0 allocs/op
  chi          291 ns/op   368 B/op    2 allocs/op
  gin           38 ns/op     0 B/op    0 allocs/op
  net/http      64 ns/op     0 B/op    0 allocs/op

SingleParam
  httx          41 ns/op     0 B/op    0 allocs/op
  httprouter    65 ns/op    64 B/op    1 allocs/op
  chi          463 ns/op   704 B/op    4 allocs/op
  gin           47 ns/op     0 B/op    0 allocs/op
  net/http     108 ns/op    16 B/op    1 allocs/op

MultiParam
  httx          70 ns/op     0 B/op    0 allocs/op
  httprouter    63 ns/op    64 B/op    1 allocs/op
  chi          519 ns/op   704 B/op    4 allocs/op
  gin           59 ns/op     0 B/op    0 allocs/op
  net/http     183 ns/op    48 B/op    2 allocs/op

RegexParam
  httx         180 ns/op    48 B/op    2 allocs/op
  chi          554 ns/op   704 B/op    4 allocs/op

Wildcard
  httx          38 ns/op     0 B/op    0 allocs/op
  httprouter    46 ns/op    32 B/op    1 allocs/op
  chi          440 ns/op   704 B/op    4 allocs/op
  gin           46 ns/op     0 B/op    0 allocs/op
  net/http     358 ns/op    96 B/op    5 allocs/op

MethodMismatch
  httx         115 ns/op    64 B/op    1 allocs/op
  httprouter   703 ns/op   276 B/op    8 allocs/op
  chi          657 ns/op   631 B/op    2 allocs/op
  gin          201 ns/op   131 B/op    3 allocs/op
  net/http    1940 ns/op   588 B/op   27 allocs/op

NotFound
  httx          56 ns/op     0 B/op    0 allocs/op
  httprouter   359 ns/op   100 B/op    3 allocs/op
  chi          512 ns/op   468 B/op    5 allocs/op
  gin          113 ns/op   100 B/op    1 allocs/op
  net/http     186 ns/op    48 B/op    3 allocs/op

TrailingSlash
  httx          54 ns/op     0 B/op    0 allocs/op
  httprouter   258 ns/op   184 B/op    3 allocs/op
  gin          417 ns/op   280 B/op    8 allocs/op
  net/http     103 ns/op    16 B/op    1 allocs/op

CaseInsensitive
  httx          94 ns/op     0 B/op    0 allocs/op
  httprouter   400 ns/op   216 B/op    4 allocs/op
  gin          596 ns/op   408 B/op    8 allocs/op
```

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
