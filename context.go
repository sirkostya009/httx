package httx

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"strconv"
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
	ctx.ResponseWriter.Header()["Location"] = []string{url}
	ctx.WriteHeader(status)
	return nil
}

func (ctx *Context) WriteXML(status int, a any) error {
	ctx.ResponseWriter.Header()["Content-Type"] = []string{"application/xml"}
	ctx.WriteHeader(status)
	return xml.NewEncoder(ctx.ResponseWriter).Encode(a)
}

func (ctx *Context) WriteJSON(status int, a any) error {
	ctx.ResponseWriter.Header()["Content-Type"] = []string{"application/json"}
	ctx.WriteHeader(status)
	return json.NewEncoder(ctx.ResponseWriter).Encode(a)
}

func (ctx *Context) WriteCSV(status int, CRLF bool, separator rune, data [][]string) error {
	ctx.ResponseWriter.Header()["Content-Type"] = []string{"text/csv"}
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

func (ctx *Context) PathInt(name string, base, bitSize int) (int64, error) {
	return strconv.ParseInt(ctx.PathValue(name), base, bitSize)
}

func (ctx *Context) PathComplex(name string, bitSize int) (complex128, error) {
	return strconv.ParseComplex(ctx.PathValue(name), bitSize)
}

func (ctx *Context) PathBool(name string) (bool, error) {
	return strconv.ParseBool(ctx.PathValue(name))
}

func (ctx *Context) PathFloat(name string, bitSize int) (float64, error) {
	return strconv.ParseFloat(ctx.PathValue(name), bitSize)
}

func (ctx *Context) PathUint(name string, base, bitSize int) (uint64, error) {
	return strconv.ParseUint(ctx.PathValue(name), base, bitSize)
}

func (ctx *Context) FormInt(name string, base, bitSize int) (int64, error) {
	return strconv.ParseInt(ctx.FormValue(name), base, bitSize)
}

func (ctx *Context) FormComplex(name string, bitSize int) (complex128, error) {
	return strconv.ParseComplex(ctx.FormValue(name), bitSize)
}

func (ctx *Context) FormBool(name string) (bool, error) {
	return strconv.ParseBool(ctx.FormValue(name))
}

func (ctx *Context) FormFloat(name string, bitSize int) (float64, error) {
	return strconv.ParseFloat(ctx.FormValue(name), bitSize)
}

func (ctx *Context) FormUint(name string, base, bitSize int) (uint64, error) {
	return strconv.ParseUint(ctx.FormValue(name), base, bitSize)
}

func (ctx *Context) TrueClientIP() string {
	if tcip := ctx.Request.Header["True-Client-IP"]; len(tcip) > 0 {
		return tcip[0]
	}
	return ctx.RemoteAddr
}
