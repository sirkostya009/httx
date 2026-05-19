# HTTp eXtended

Fork of [fasthttp/router](https://github.com/fasthttp/router/) adapted for `http.Handler`.

Engineered as a simple improvement upon standard `net/http` implementation of ServeMux, with main algorithm and ergonomics largely borrowed from `fasthttp/router`.

Thus, this multiplexer has optional and regex path params unlike the standard one.

Inherits 0 allocation routing, except for redirects. This is a deliberate choice attempting to strip away any external deps from codebase.

Additionally, RedirectCaseInsensitivePath (RedirectFixedPath in `fasthttp/router`) works differently by only matching case insensitive paths, with path resolution done by `http.Server`.

If you're not using `http.Server` and need path resolution, I suggest you utilize `ResolvePath` mux wrapper function from [appendix section](#appendix).

You _may_ want to disable redirects if you run into GC issues (but this router would probably be the least of your allocation problems anyway).

# Usage

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

## Benchmarks

Comparison against `httprouter`, `chi`, `gin`, and `net/http` on AMD Ryzen AI Max+ 395. Run with `-benchtime=10000x` with a warm up run. Source: [bench/](bench/).

`gin` and `net/http` don't support regex param validation. `gin` and `net/http` don't redirect on case-mismatched paths.

```
Simple
  httx          17 ns/op     0 B/op    0 allocs/op
  httprouter    23 ns/op     0 B/op    0 allocs/op
  chi          264 ns/op   368 B/op    2 allocs/op
  gin           38 ns/op     0 B/op    0 allocs/op
  net/http      67 ns/op     0 B/op    0 allocs/op

SingleParam
  httx          41 ns/op     0 B/op    0 allocs/op
  httprouter    58 ns/op    64 B/op    1 allocs/op
  chi          467 ns/op   704 B/op    4 allocs/op
  gin           48 ns/op     0 B/op    0 allocs/op
  net/http     108 ns/op    16 B/op    1 allocs/op

MultiParam
  httx          71 ns/op     0 B/op    0 allocs/op
  httprouter    65 ns/op    64 B/op    1 allocs/op
  chi          511 ns/op   704 B/op    4 allocs/op
  gin           60 ns/op     0 B/op    0 allocs/op
  net/http     188 ns/op    48 B/op    2 allocs/op

RegexParam
  httx         190 ns/op    48 B/op    2 allocs/op
  chi          547 ns/op   704 B/op    4 allocs/op

Wildcard
  httx          38 ns/op     0 B/op    0 allocs/op
  httprouter    48 ns/op    32 B/op    1 allocs/op
  chi          435 ns/op   704 B/op    4 allocs/op
  gin           46 ns/op     0 B/op    0 allocs/op
  net/http     363 ns/op    96 B/op    5 allocs/op

MethodMismatch
  httx         121 ns/op    64 B/op    1 allocs/op
  httprouter   715 ns/op   276 B/op    8 allocs/op
  chi          676 ns/op   631 B/op    2 allocs/op
  gin           48 ns/op    52 B/op    0 allocs/op
  net/http    1907 ns/op   588 B/op   27 allocs/op

NotFound
  httx          55 ns/op     0 B/op    0 allocs/op
  httprouter   360 ns/op   100 B/op    3 allocs/op
  chi          512 ns/op   468 B/op    5 allocs/op
  gin           64 ns/op    52 B/op    0 allocs/op
  net/http     186 ns/op    48 B/op    3 allocs/op

TrailingSlash
  httx          54 ns/op     0 B/op    0 allocs/op
  httprouter   234 ns/op   184 B/op    3 allocs/op
  chi          512 ns/op   468 B/op    5 allocs/op
  gin          516 ns/op   280 B/op    8 allocs/op
  net/http     108 ns/op    16 B/op    1 allocs/op

CaseInsensitive
  httx          95 ns/op     0 B/op    0 allocs/op
  httprouter   409 ns/op   216 B/op    4 allocs/op
  chi          527 ns/op   468 B/op    5 allocs/op
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
