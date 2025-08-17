package httx

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type routerGrouper interface {
	Group(string) *Group
	// ServeFiles(path string, rootPath string)
	// ServeFilesCustom(path string, fs *fasthttp.FS)
}

func makeRequest(m *Mux, method, path string) (rec *httptest.ResponseRecorder) {
	rec = httptest.NewRecorder()
	m.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return
}

func assertGroup(t *testing.T, gs ...routerGrouper) {
	for i, g := range gs {
		g2 := g.Group("/")

		v1 := reflect.ValueOf(g)
		v2 := reflect.ValueOf(g2)

		if v1.String() != v2.String() { // router -> group
			if v1.Pointer() == v2.Pointer() {
				t.Errorf("[%d] equal pointers: %p == %p", i, g, g2)
			}
		} else { // group -> subgroup
			// if v1.Pointer() != v2.Pointer() {
			// 	t.Errorf("[%d] mismatch pointers: %p != %p", i, g, g2)
			// }
		}

		if err := catchPanic(func() { g.Group("v999") }); err == nil {
			t.Error("an error was expected when a path does not begin with slash")
		}

		if err := catchPanic(func() { g.Group("/v999/") }); err == nil {
			t.Error("an error was expected when a path has a trailing slash")
		}

		if err := catchPanic(func() { g.Group("") }); err == nil {
			t.Error("an error was expected with an empty path")
		}

		// if err := catchPanic(func() { g.ServeFiles("static/{filepath:*}", "./") }); err == nil {
		// 	t.Error("an error was expected when a path does not begin with slash")
		// }

		// if err := catchPanic(func() {
		// 	g.ServeFilesCustom("", &fasthttp.FS{Root: "./"})
		// }); err == nil {
		// 	t.Error("an error was expected with an empty path")
		// }

	}
}

func TestGroup(t *testing.T) {
	r1 := NewMux()
	r2 := r1.Group("/boo")
	r3 := r1.Group("/goo")
	r4 := r1.Group("/moo")
	r5 := r4.Group("/foo")
	r6 := r5.Group("/foo")

	assertGroup(t, r1, r2, r3, r4, r5, r6)

	hit := false

	r1.POST("/foo", func(w http.ResponseWriter, r *http.Request) error {
		hit = true
		return nil
	})
	r2.POST("/bar", func(w http.ResponseWriter, r *http.Request) error {
		hit = true
		return nil
	})
	r3.POST("/bar", func(w http.ResponseWriter, r *http.Request) error {
		hit = true
		return nil
	})
	r4.POST("/bar", func(w http.ResponseWriter, r *http.Request) error {
		hit = true
		return nil
	})
	r5.POST("/bar", func(w http.ResponseWriter, r *http.Request) error {
		hit = true
		return nil
	})
	r6.POST("/bar", func(w http.ResponseWriter, r *http.Request) error {
		hit = true
		return nil
	})
	// r6.ServeFiles("/static/{filepath:*}", "./")
	// r6.ServeFS("/static/fs/{filepath:*}", fsTestFilesystem)
	// r6.ServeFilesCustom("/custom/static/{filepath:*}", &fasthttp.FS{Root: "./"})

	uris := []struct{ method, path string }{
		{"POST", "/foo"},
		// testing router group - r2 (grouped from r1)
		{"POST", "/boo/bar"},
		// testing multiple router group - r3 (grouped from r1)
		{"POST", "/goo/bar"},
		// testing multiple router group - r4 (grouped from r1)
		{"POST", "/moo/bar"},
		// testing sub-router group - r5 (grouped from r4)
		{"POST", "/moo/foo/bar"},
		// testing multiple sub-router group - r6 (grouped from r5)
		{"POST", "/moo/foo/foo/bar"},
		// testing multiple sub-router group - r6 (grouped from r5) to serve files
		// {"GET", "/moo/foo/foo/static/router.go"},
		// testing multiple sub-router group - r6 (grouped from r5) to serve fs
		// {"GET", "/moo/foo/foo/static/fs/LICENSE"},
		// testing multiple sub-router group - r6 (grouped from r5) to serve files with custom settings
		// {"GET", "/moo/foo/foo/custom/static/router.go"},
	}

	for _, uri := range uris {
		hit = false

		res := makeRequest(r1, uri.method, uri.path)
		// if err != nil {
		// 	t.Fatalf("Unexpected error when reading response: %s", err)
		// }
		if res.Code != http.StatusOK {
			t.Fatalf("Status code %d, want %d", res.Code, http.StatusOK)
		}
		if !strings.Contains(uri.path, "static") && !hit {
			t.Fatalf("Regular routing failed with router chaining. %s", uri)
		}
	}

	res := makeRequest(r1, "POST", "/qax")
	// if err != nil {
	// 	t.Fatalf("Unexpected error when reading response: %s", err)
	// }
	if res.Code != http.StatusNotFound {
		t.Errorf("NotFound behavior failed with router chaining.")
		t.FailNow()
	}
}

func TestGroup_shortcutsAndHandle(t *testing.T) {
	r := NewMux()
	g := r.Group("/v1")

	shortcuts := []func(path string, handler HandlerFunc){
		g.GET,
		g.HEAD,
		g.POST,
		g.PUT,
		g.PATCH,
		g.DELETE,
		g.CONNECT,
		g.OPTIONS,
		g.TRACE,
		g.ANY,
	}

	for _, fn := range shortcuts {
		fn("/bar", func(http.ResponseWriter, *http.Request) error { return nil })

		if err := catchPanic(func() { fn("buzz", func(http.ResponseWriter, *http.Request) error { return nil }) }); err == nil {
			t.Error("an error was expected when a path does not begin with slash")
		}

		if err := catchPanic(func() { fn("", func(http.ResponseWriter, *http.Request) error { return nil }) }); err == nil {
			t.Error("an error was expected with an empty path")
		}
	}

	methods := httpMethods[:len(httpMethods)-1] // Avoid customs methods
	for _, method := range methods {
		h, _ := r.trees[r.methodIndexOf(method)].Get("/v1/bar", nil)
		if h == nil {
			t.Errorf("Bad shorcurt")
		}
	}

	g2 := g.Group("/foo")

	for _, method := range httpMethods {
		g2.Handle(method, "/bar", func(http.ResponseWriter, *http.Request) error { return nil })

		if err := catchPanic(func() { g2.Handle(method, "buzz", func(http.ResponseWriter, *http.Request) error { return nil }) }); err == nil {
			t.Error("an error was expected when a path does not begin with slash")
		}

		if err := catchPanic(func() { g2.Handle(method, "", func(http.ResponseWriter, *http.Request) error { return nil }) }); err == nil {
			t.Error("an error was expected with an empty path")
		}

		h, _ := r.trees[r.methodIndexOf(method)].Get("/v1/foo/bar", nil)
		if h == nil {
			t.Errorf("Bad shorcurt")
		}
	}
}
