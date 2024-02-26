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

	mux.HandleHttp("GET /csv-hello", func(ctx *tinymux.Context) error {
		data := [][]string{
			{"foo", "bar"},
			{"hello", "world"},
		}
		return ctx.WriteCSV(200, false, ',', data)
	})

	_ = http.ListenAndServe(":8080", mux)
}
```

## License

MIT. See [LICENSE](LICENSE).
