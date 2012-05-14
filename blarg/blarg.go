package blarg

import (
    "net/http"
    "pat"
    "mustache"
    "blackfriday"
    "bytes"
    "io"
    "time"
    "appengine"
    "appengine/datastore"
)

type Post struct {
    Title       string
    Content     string
    Postdate    time.Time
    StickyUrl   string
    Sidebar     bool
}

func Index(w http.ResponseWriter, req *http.Request) {

  cd := appengine.NewContext(req)

  p1 := Post{
      Title:    "awesome post great job",
      Content:  "chickens\n======\n\nchickens are *boss*. look at this:\n\n* chickens are tasty\n* chickens are not green\n* you too can be a chicken with focused thought",
      Postdate: time.Now(),
      StickyUrl: "",
      Sidebar: false,
  }

  key, err := datastore.Put(cd, datastore.NewIncompleteKey(cd, "post", nil), &p1)
  if err != nil {
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
  }

  var p2 Post
  if err = datastore.Get(cd, key, &p2); err != nil {
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
  }

  bloginfo := map[string]string { "blog_title": "my blarg!" }
  static_private := "static_private/"

  h := mustache.RenderFile(static_private + "header.html.mustache", bloginfo)
  io.WriteString(w, h)

  //context := map[string]string { "c": req.URL.Query().Get(":name") }
  //context := map[string]string { "c": "yolo!" }
  //c := mustache.RenderFile(static_private + "splash.html.mustache", context)
  //io.WriteString(w, c)

  //what
  con := bytes.NewBuffer(blackfriday.MarkdownCommon(bytes.NewBufferString(p2.Content).Bytes())).String()
  context := map[string]string { "c": con }
  c := mustache.RenderFile(static_private + "splash.html.mustache", context)
  io.WriteString(w, c)

  timing := map[string]string { "render": "0.01s" }
  f := mustache.RenderFile(static_private + "footer.html.mustache", timing)
  io.WriteString(w, f)
}

func init() {
  m := pat.New()
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
