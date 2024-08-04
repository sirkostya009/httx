/*
Package httx a thin wrapper over standard net/http package.

This package allows for easier management of request contexts,
middleware, centralized error handling for a price of a small startup overhead.

# Usage

	mux := httx.NewServeMux(http.DefaultServeMux) // I mean, why the hell not use the default serve mux?

	// Middleware must be initialized before any route
	mux.Pre(func(ctx *httx.Context) error {
	    for _, cookie := range ctx.Cookies() {
	        if cookie.Name == "session" {
	            // session validation, etc
	            ctx.Set("session", cookie.Value)
	            break
	        }
	    }
	    return nil
	})

	// Method prefix is available since go ver 1.22
	mux.HandleFunc("GET /csv", func(ctx *httx.Context) error {
	    data := [][]string{
	        {"foo", "bar"},
	        {"hello", "world"},
	    }
	    return ctx.WriteCSV(200, false, ',', data)
	})

	mux.HandleFunc("GET /json", func(ctx *httx.Context) error {
	    someDatabaseFunc(ctx) // httx.Context is a wrapper around http.ResponseWriter and http.Request,
	                          // as well as http.Request's context.Context, allowing for usage of it
	                          // as an actual context instance

	    return ctx.WriteJSON(200, map[string]any{
	        "session": ctx.Value("session"), // get the session value
	    })
	})

	_ = http.ListenAndServe(":8080", mux) // you can easily go with nil here due to httx modifying http.DefaultServeMux
*/
package httx
