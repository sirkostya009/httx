package tinymux

import "testing"

func TestMux(t *testing.T) {
	mux := NewMux(nil)

	mux.Pre(func(ctx *Context) error {
		ctx.Set("foo", "bar")
		return nil
	})

	mux.HandleHttp("/http", func(ctx *Context) error {
		_, err := ctx.WriteString("http")
		return err
	})

	mux.HandleHttp("/json", func(ctx *Context) error {
		return ctx.WriteJSON(200, []string{"json"})
	})
}
