package blarg

import (
    "net/http"
    "pat"
    "mustache"
    "io"
)

func HelloServer(w http.ResponseWriter, req *http.Request) {
    context := map[string]string { "c": req.URL.Query().Get(":name")}
    l := mustache.Render("hello {{c}}!\n", context)
    io.WriteString(w, l)
}

func init() {
    m := pat.New()
    m.Get("/", http.HandlerFunc(Index))
    m.Get("/list/:start", http.HandlerFunc(Index))
    // m.Get("/rss", http.HandlerFunc(Rss))
    // m.Get("/sitemap.xml", http.HandlerFunc(Sitemap))
    m.Get("/item/:item", http.HandlerFunc(Lookup))
    m.Get("/admin", http.HandlerFunc(Admin))

    // Register this pat with the default serve mux so that other packages
    // may also be exported. (i.e. /debug/pprof/*)
    http.Handle("/", m)
}
