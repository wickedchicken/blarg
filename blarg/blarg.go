package blarg

import (
    "net/http"
    "pat"
    "mustache"
    "bytes"
)

// hello world, the web server
func HelloServer(w http.ResponseWriter, req *http.Request, tmpl *mustache.Template) {
    var buf bytes.Buffer;
    tmpl.Render (map[string]string { "c": req.URL.Query().Get(":name")}, &buf)
    buf.WriteTo(w)
}

func init() {
    tmpl,_ := mustache.ParseString("hello {{c}}!\n")
    tmplhdlr:= func(w http.ResponseWriter, req *http.Request) {
      HelloServer(w, req, tmpl)
    }

    m := pat.New()
    m.Get("/hello/:name", http.HandlerFunc(tmplhdlr))

    // Register this pat with the default serve mux so that other packages
    // may also be exported. (i.e. /debug/pprof/*)
    http.Handle("/", m)
}
