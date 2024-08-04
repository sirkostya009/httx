package httx

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"strconv"
)

type Context struct {
	http.ResponseWriter
	*http.Request
	context.Context
	values map[any]any
}

func (ctx *Context) NoContent(status int) error {
	ctx.WriteHeader(status)
	return nil
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
	if err := xml.NewEncoder(ctx.ResponseWriter).Encode(a); err != nil {
		return err
	}
	ctx.ResponseWriter.Header()["Content-Type"] = []string{"application/xml"}
	ctx.WriteHeader(status)
	return nil
}

func (ctx *Context) WriteJSON(status int, a any) error {
	if err := json.NewEncoder(ctx.ResponseWriter).Encode(a); err != nil {
		return err
	}
	ctx.ResponseWriter.Header()["Content-Type"] = []string{"application/json"}
	ctx.WriteHeader(status)
	return nil
}

func (ctx *Context) WriteCSV(status int, CRLF bool, separator rune, data [][]string) error {
	w := csv.NewWriter(ctx.ResponseWriter)
	w.Comma = separator
	w.UseCRLF = CRLF
	if err := w.WriteAll(data); err != nil {
		return err
	}
	ctx.ResponseWriter.Header()["Content-Type"] = []string{"text/csv"}
	ctx.WriteHeader(status)
	return nil
}

func (ctx *Context) WriteHTML(status int, html string) error {
	if _, err := ctx.WriteString(html); err != nil {
		return err
	}
	ctx.ResponseWriter.Header()["Content-Type"] = []string{"application/html"}
	ctx.WriteHeader(status)
	return nil
}

func (ctx *Context) WriteText(status int, s string) error {
	if _, err := ctx.WriteString(s); err != nil {
		return err
	}
	ctx.ResponseWriter.Header()["Content-Type"] = []string{"text/plain"}
	ctx.WriteHeader(status)
	return nil
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

func (ctx *Context) ReadText() (string, error) {
	b, err := io.ReadAll(ctx.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
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

// WithContext overrides http.Request's WithContext method.
//
// Return can be safely ignored.
func (ctx *Context) WithContext(c context.Context) *Context {
	ctx.Request = ctx.Request.WithContext(c)
	ctx.Context = ctx.Request.Context()
	return ctx
}

// WithValue A handy shorthand for the context.WithValue method.
//
// Meant to solve the same problem as Set, but with more standard
// and widespread context mechanics.
//
// If you have a lot of fields you wish to set this way,
// consider using Set instead.
func (ctx *Context) WithValue(key, val any) *Context {
	return ctx.WithContext(context.WithValue(ctx.Context, key, val))
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
