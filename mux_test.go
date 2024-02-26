package tinymux

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func readAsString(b io.ReadCloser, l int64) string {
	buf := make([]byte, l)
	n, _ := b.Read(buf)
	return string(buf[:n])
}

func TestMux(t *testing.T) {
	mux := NewMux(nil)

	mux.Pre(func(ctx *Context) error {
		ctx.Set("foo", "bar")
		return nil
	})

	mux.HandleHttp("GET /string", func(ctx *Context) error {
		_, err := ctx.WriteString(ctx.Value("foo").(string))
		return err
	})

	mux.HandleHttp("GET /csv", func(ctx *Context) error {
		return ctx.WriteCSV(200, false, ',', [][]string{
			{"foo"},
			{ctx.Value("foo").(string)},
		})
	})

	mux.HandleHttp("GET /json", func(ctx *Context) error {
		return ctx.WriteJSON(200, map[string]string{"foo": ctx.Value("foo").(string)})
	})

	server := httptest.NewServer(mux)

	res, err := http.Get(server.URL + "/string")
	if err != nil {
		t.Error(err)
	}
	if res.StatusCode != 200 {
		t.Errorf("/string: expected 200, got %d", res.StatusCode)
	}
	if s := readAsString(res.Body, res.ContentLength); s != "bar" {
		t.Errorf("expected 'bar', got '%s'", s)
	}

	res, err = http.Get(server.URL + "/csv")
	if err != nil {
		t.Error(err)
	}
	if res.StatusCode != 200 {
		t.Errorf("/csv: expected 200, got %d", res.StatusCode)
	}
	if s := readAsString(res.Body, res.ContentLength); s != "foo\nbar\n" {
		t.Errorf("expected 'foo\nbar\n', got '%s'", s)
	}

	res, err = http.Get(server.URL + "/json")
	if err != nil {
		t.Error(err)
	}
	if res.StatusCode != 200 {
		t.Errorf("/json: expected 200, got %d", res.StatusCode)
	}
	if s := readAsString(res.Body, res.ContentLength); s != `{"foo":"bar"}` {
		t.Errorf("expected '{\"foo\":\"bar\"}', got '%s'", s)
	}

	server.Close()
}

func TestMuxErrorHandler(t *testing.T) {
	mux := NewMux(nil)

	mux.HandleHttp("GET /error", func(ctx *Context) error {
		return errors.New("error")
	})

	server := httptest.NewServer(mux)

	res, err := http.Get(server.URL + "/error")
	if err != nil {
		t.Error(err)
	}
	if res.StatusCode != 500 {
		t.Errorf("/error: expected 500, got %d", res.StatusCode)
	}
	if s := readAsString(res.Body, res.ContentLength); s != "error\n" {
		t.Errorf("expected 'error', got '%s'", s)
	}

	server.Close()
}
