package httx_test

import (
	"log"
	"net/http"

	"github.com/sirkostya009/httx"
)

type ExampleServer struct {
	/// some injected deps
	r *httx.Mux
}

func (s *ExampleServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// preflight code

	s.r.ServeHTTP(w, r)

	// postflight code
}

func Example() {
	exSrv := &ExampleServer{r: httx.NewMux()}

	httpSrv := &http.Server{
		Handler: exSrv,
		// some other optional config
	}

	err := httpSrv.ListenAndServe()
	if err != nil {
		log.Fatal("listen'n'serve:", err)
	}
}
