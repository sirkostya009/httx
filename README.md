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

Comparison against `httprouter`, `chi`, `gin`, and stdlib `net/http`. Run with `-benchtime=10s`, no CPU limit, on AMD Ryzen AI Max+ 395 (32 logical cores). Source: [bench/](bench/).

```
bench            httx    httprouter  chi    gin    stdlib
Simple           17ns    22ns        221ns  38ns   68ns
SingleParam      41ns    63ns        394ns  47ns   112ns
MultiParam       68ns    73ns        434ns  59ns   197ns
RegexParam       199ns   -           480ns  -      -
Wildcard         57ns    54ns        371ns  47ns   392ns
MethodMismatch   249ns   614ns       543ns  53ns   1684ns
NotFound         179ns   371ns       383ns  65ns   200ns
TrailingSlash    54ns    269ns       386ns  514ns  110ns
CaseInsensitive  94ns    458ns       386ns  -      -
```

`-` means the router does not support that feature. `gin` and `stdlib` don't have regex param validation. `gin` and `stdlib` don't redirect on case-mismatched paths.

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
