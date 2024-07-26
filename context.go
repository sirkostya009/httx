package httx

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
	return xml.NewEncoder(ctx.ResponseWriter).Encode(a)
}

func (ctx *Context) WriteJSON(status int, a any) error {
	ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
	ctx.WriteHeader(status)
	return json.NewEncoder(ctx.ResponseWriter).Encode(a)
}

func (ctx *Context) WriteCSV(status int, CRLF bool, separator rune, data [][]string) error {
	ctx.ResponseWriter.Header().Set("Content-Type", "text/csv")
	ctx.WriteHeader(status)
	w := csv.NewWriter(ctx.ResponseWriter)
	w.Comma = separator
	w.UseCRLF = CRLF
	return w.WriteAll(data)
}

func (ctx *Context) ReadJSON(a any) error {
	return json.NewDecoder(ctx.Body).Decode(a)
}

func (ctx *Context) ReadXML(a any) error {
	return xml.NewDecoder(ctx.Body).Decode(a)
}

func (ctx *Context) ReadCSV(separator rune) *csv.Reader {
	r := csv.NewReader(ctx.Body)
	r.Comma = separator
	return r
}

func (ctx *Context) Set(key, value any) {
	// this assumes that users mostly don't use the Set method hence
	// the values map is not initialized until the first call to Set
	// to avoid the overhead of initializing the map per every request
	if ctx.values == nil {
		ctx.values = map[any]any{}
	}
	ctx.values[key] = value
}

func (ctx *Context) Value(key any) any {
	if v, ok := ctx.values[key]; ok {
		return v
	}
	return ctx.Context.Value(key)
}
