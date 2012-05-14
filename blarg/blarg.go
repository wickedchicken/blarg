package blarg

import (
    "net/http"
    "pat"
    "mustache"
    "io"
)

func Index(w http.ResponseWriter, req *http.Request) {

  bloginfo := map[string]string { "blog_name": "my blarg!" }
  static_private := "static_private/"

  h := mustache.RenderFile(static_private + "header.html.mustache", bloginfo)
  io.WriteString(w, h)

  //context := map[string]string { "c": req.URL.Query().Get(":name") }
  context := map[string]string { "c": "yolo!" }
  c := mustache.RenderFile(static_private + "splash.html.mustache", context)
  io.WriteString(w, c)

  timing := map[string]string { "render": "0.01s" }
  f := mustache.RenderFile(static_private + "footer.html.mustache", timing)
  io.WriteString(w, f)
}

func init() {
  m := pat.New()
  m.Get("/hello/:name", http.HandlerFunc(Index))
  m.Get("/", http.HandlerFunc(Index))
  //m.Get("/list/:start", http.HandlerFunc(Index))
  // m.Get("/rss", http.HandlerFunc(Rss))
  // m.Get("/sitemap.xml", http.HandlerFunc(Sitemap))
  //m.Get("/item/:item", http.HandlerFunc(Lookup))
  //m.Get("/admin", http.HandlerFunc(Admin))

  // Register this pat with the default serve mux so that other packages
  // may also be exported. (i.e. /debug/pprof/*)
  http.Handle("/", m)
}
