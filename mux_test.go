package tinymux

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testPath(t *testing.T, url, expected string, code int) {
	res, err := http.Get(url)
	if err != nil {
		t.Error(err)
	}
	if res.StatusCode != code {
		t.Errorf("%s: expected 200, got %d", url, res.StatusCode)
	}
	buf := make([]byte, res.ContentLength)
	n, _ := res.Body.Read(buf)
	if s := string(buf[:n]); s != expected {
		t.Errorf("%s: expected '%s', got '%s'", url, expected, s)
	}
}

func TestMux(t *testing.T) {
	mux := NewMux(nil)

	mux.Pre(func(ctx *Context) error {
		ctx.Set("foo", "bar")
		return nil
	})

	mux.HandleHttp("/string", func(ctx *Context) error {
		_, err := ctx.WriteString(ctx.Value("foo").(string))
		return err
	})

	mux.HandleHttp("/csv", func(ctx *Context) error {
		return ctx.WriteCSV(200, false, ',', [][]string{
			{"foo"},
			{ctx.Value("foo").(string)},
		})
	})

	mux.HandleHttp("/json", func(ctx *Context) error {
		return ctx.WriteJSON(200, map[string]string{"foo": ctx.Value("foo").(string)})
	})

	server := httptest.NewServer(mux)

	testPath(t, server.URL+"/string", "bar", 200)

	testPath(t, server.URL+"/csv", "foo\nbar\n", 200)

	testPath(t, server.URL+"/json", `{"foo":"bar"}`, 200)

	server.Close()
}

func TestMuxErrorHandler(t *testing.T) {
	mux := NewMux(nil)

	mux.HandleHttp("/error", func(ctx *Context) error {
		return errors.New("error")
	})

	server := httptest.NewServer(mux)

	testPath(t, server.URL+"/error", "error\n", 500)

	server.Close()
}
