package httx_test

import (
	"errors"
	"github.com/sirkostya009/httx"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testPath(t *testing.T, url, expected string, code int) {
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
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
	mux := httx.NewServeMux()

	mux.Pre(func(ctx *httx.Context) error {
		ctx.Set("foo", "bar")
		return nil
	})

	mux.HandleFunc("/string", func(ctx *httx.Context) error {
		_, err := ctx.WriteString(ctx.Value("foo").(string))
		return err
	})

	mux.HandleFunc("/csv", func(ctx *httx.Context) error {
		return ctx.WriteCSV(200, false, ',', [][]string{
			{"foo"},
			{ctx.Value("foo").(string)},
		})
	})

	mux.HandleFunc("/json", func(ctx *httx.Context) error {
		return ctx.WriteJSON(200, map[string]string{"foo": ctx.Value("foo").(string)})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	testPath(t, server.URL+"/string", "bar", 200)

	testPath(t, server.URL+"/csv", "foo\nbar\n", 200)

	testPath(t, server.URL+"/json", `{"foo":"bar"}`+"\n", 200)
}

func TestMuxErrorHandler(t *testing.T) {
	mux := httx.NewServeMux()

	mux.HandleFunc("/error", func(ctx *httx.Context) error {
		return errors.New("error")
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	testPath(t, server.URL+"/error", "error\n", 500)
}
