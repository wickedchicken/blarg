package blarg

import (
    "io"
    "net/http"
    "pat"
)

// hello world, the web server
func HelloServer(w http.ResponseWriter, req *http.Request) {
    io.WriteString(w, "hello, "+req.URL.Query().Get(":name")+"!\n")
}

func init() {
    m := pat.New()
    m.Get("/hello/:name", http.HandlerFunc(HelloServer))

    // Register this pat with the default serve mux so that other packages
    // may also be exported. (i.e. /debug/pprof/*)
    http.Handle("/", m)
}
