package blarg

import (
    "net/http"
    "pat"
    "blarg/config"
    "blarg/layout"
)


func init() {
  blog_config,err := config.ReadJsonFile("blarg_config.json")
  if err != nil {
    panic(err)
  }

  root := config.Stringify(blog_config["blog_root"]) // test for default

  m := pat.New()
  m.Get(root + "list/:page/:invalid/", http.HandlerFunc(http.NotFound))
  m.Get(root + "list/:page", http.HandlerFunc(layout.IndexPageHandler(blog_config)))
  m.Get(root + "list/", http.HandlerFunc(layout.IndexListHandler(blog_config)))
  m.Get(root + "index/", http.HandlerFunc(layout.IndexListHandler(blog_config)))

  // matching on / will match all URLs
  // so you have to catch invalid top-level URLs first

  m.Get(root + ":invalid/", http.HandlerFunc(http.NotFound))
  m.Get(root , http.HandlerFunc(layout.IndexListHandler(blog_config)))

  // m.Get("/rss", http.HandlerFunc(Rss))
  // m.Get("/sitemap.xml", http.HandlerFunc(Sitemap))
  //m.Get("/item/:item", http.HandlerFunc(Lookup))
  //m.Get("/admin", http.HandlerFunc(Admin))

  http.Handle(root, m)
}
