package tinymux

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"net/http"
)

type Context struct {
	http.ResponseWriter
	*http.Request
	context.Context
	values map[any]any
}

func (ctx *Context) WriteString(s string) (int, error) {
	return ctx.ResponseWriter.Write([]byte(s))
}

func (ctx *Context) Redirect(status int, url string) error {
	ctx.ResponseWriter.Header().Set("Location", url)
	ctx.WriteHeader(status)
	return nil
}

func (ctx *Context) WriteXML(status int, a any) error {
	ctx.ResponseWriter.Header().Set("Content-Type", "application/xml")
	ctx.WriteHeader(status)
	b, err := xml.Marshal(a)
	if err != nil {
		return err
	}
	_, err = ctx.ResponseWriter.Write(b)
	return err
}

func (ctx *Context) WriteJSON(status int, a any) error {
	ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
	ctx.WriteHeader(status)
	b, err := json.Marshal(a)
	if err != nil {
		return err
	}
	_, err = ctx.ResponseWriter.Write(b)
	return err
}

func (ctx *Context) WriteCSV(status int, CRLF bool, separator rune, data [][]string) error {
	ctx.ResponseWriter.Header().Set("Content-Type", "text/csv")
	ctx.WriteHeader(status)
	w := csv.NewWriter(ctx.ResponseWriter)
	w.Comma = separator
	w.UseCRLF = CRLF
	return w.WriteAll(data)
}

func (ctx *Context) Set(key, value any) {
	ctx.values[key] = value
}

func (ctx *Context) Value(key any) any {
	if v, ok := ctx.values[key]; ok {
		return v
	}
	return ctx.Context.Value(key)
}
