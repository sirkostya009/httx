# tinymux

Tinymux is an extremely thin and simple http multiplexer that is designed as a wrapper layer for `http.ServeMux` to
simplify usage of middleware, error handling and response writing.

## Usage

```go
package main

import (
	"github.com/sirkostya009/tinymux"
	"net/http"
)

func main() {
	mux := tinymux.NewMux(http.DefaultServeMux) // nil to internally create a new ServeMux

	// Method prefix is available since go ver 1.22
	mux.HandleHttp("GET /csv", func(ctx *tinymux.Context) error {
		data := [][]string{
			{"foo", "bar"},
			{"hello", "world"},
		}
		return ctx.WriteCSV(200, false, ',', data)
	})

	mux.HandleHttp("GET /json", func(ctx *tinymux.Context) error {
		someDatabaseFunc(ctx) // tinymux.Context is a wrapper around http.ResponseWriter and http.Request,
		                      // as well as http.Request's context.Context, meaning you can easily pass it around, as a
		                      // context object

		return ctx.WriteJSON(200, map[string]any{
			"session": ctx.Value("session"), // get the session value
		})
	})

	// Pre middleware
	mux.Pre(func(ctx *tinymux.Context) error {
		for _, cookie := range ctx.Cookies() {
			if cookie.Name == "session" {
				// session validation, etc
				ctx.Set("session", cookie.Value)
				break
			}
		}
		return nil
	})

	_ = http.ListenAndServe(":8080", mux) // you can easily substitute with nil here due to tinymux modifying the http.DefaultServeMux
}
```

Since the mux wrapper requires anything that implements the `tinymux.StandardMultiplexer` interface, which happens to be
`http.ServeMux`, you can theoretically substitute it with any other implementation of that interface.

## License

MIT. See [LICENSE](LICENSE).
