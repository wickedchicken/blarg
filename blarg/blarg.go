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

  handle_method := func(method string, urlpattern string, handler func(http.ResponseWriter, *http.Request)){
    m.Add(method, root + urlpattern, http.HandlerFunc(handler))
  }

  handle := func(urlpattern string, handler func(http.ResponseWriter, *http.Request)){
    m.Get(root + urlpattern, http.HandlerFunc(handler))
  }

  handle("list/:page/:invalid", http.NotFound)
  handle("list/:page", layout.IndexPageHandler(blog_config, "list"))
  handle("list/", layout.IndexListHandler(blog_config, "list"))
  handle("index/", layout.IndexListHandler(blog_config, "list"))

  handle("article/:name/:invalid", http.NotFound)
  handle("article/:name", layout.GetArticle(blog_config, "article"))
  handle("article/", http.NotFound)


  handle("label/:label/:page/:invalid", http.NotFound)
  handle("label/:label/:page", layout.LabelPage(blog_config, "label"))
  handle("label/:label", layout.LabelList(blog_config, "label"))
  handle("label/", http.NotFound)

  handle("admin/edit/:name/:invalid", http.NotFound)
  handle("admin/edit/:name", layout.EditPost(blog_config))
  handle("admin/edit/", layout.EditPost(blog_config))

  handle("admin/gettext/:name/:invalid", http.NotFound)
  handle("admin/gettext/:name", layout.GetPostText(blog_config))
  handle_method("POST", "admin/render/:invalid", http.NotFound)
  handle_method("POST", "admin/render/", layout.RenderPost(blog_config))
  handle_method("POST", "admin/post/:invalid", http.NotFound)
  handle_method("POST", "admin/post/", layout.Save(blog_config))

  handle("sitemap.xml", layout.GetSitemap(blog_config))

  // matching on / will match all URLs
  // so you have to catch invalid top-level URLs first

  handle(":invalid/", http.NotFound)
  handle("", layout.IndexListHandler(blog_config, "list"))

  // m.Get("/rss", http.HandlerFunc(Rss))
  //m.Get("/admin", http.HandlerFunc(Admin))

  http.Handle(root, m)
}
