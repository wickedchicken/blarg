package blarg

import (
    "net/http"
    "pat"
    "mustache"
    "io"
)

// hello world, the web server
func HelloServer(w http.ResponseWriter, req *http.Request) {
    context := map[string]string { "c": req.URL.Query().Get(":name")}
    l := mustache.Render("hello {{c}}!\n", context)
    io.WriteString(w, l)
}

func init() {
    m := pat.New()
    m.Get("/hello/:name", http.HandlerFunc(HelloServer))

    // Register this pat with the default serve mux so that other packages
    // may also be exported. (i.e. /debug/pprof/*)
    http.Handle("/", m)
}
