# tinymux

Tinymux is an extremely thin and simple http multiplexer that is designed as a wrapper layer around standard library's
http `ServeMux` to provide a simple way for implementing middleware, error handling and response writing.

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
	mux.HandleHttp("GET /csv-hello", func(ctx *tinymux.Context) error {
		data := [][]string{
			{"foo", "bar"},
			{"hello", "world"},
		}
		return ctx.WriteCSV(200, false, ',', data)
	})

	mux.HandleHttp("GET /json-hello", func(ctx *tinymux.Context) error {
		someDatabaseFunc(ctx) // tinymux.Context is a wrapper around http.ResponseWriter and http.Request,
		                      // as well as http.Request's context.Context, meaning you can easily pass it around, as a
		                      // context object

		ctx.Value("session") // get the session value
		return ctx.WriteJSON(200, map[string]string{"hello": "world"})
	})

	// Pre middleware
	mux.Pre(func(ctx *tinymux.Context) error {
		var session string
		for _, cookie := range ctx.Cookies() {
			if cookie.Name == "session" {
				session = cookie.Value
				break
			}
		}
		// session validation, etc
		ctx.Set("session", session)
		return nil
	})

	_ = http.ListenAndServe(":8080", mux)
}
```

Since the mux wrapper requires anything that implements the `tinymux.StandardMultiplexer` interface, which happens to be
`http.ServeMux`, you can theoretically use any other mux that implements that interface.

## License

MIT. See [LICENSE](LICENSE).
