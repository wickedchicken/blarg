package blarg

import (
	"blarg/config"
	"blarg/layout"
	"github.com/bmizerany/pat"
	"net/http"
)

func init() {
	blog_config, err := config.ReadJsonFile("blarg_config.json")
	if err != nil {
		panic(err)
	}

	root := config.Stringify(blog_config["blog_root"]) // test for default

	m := pat.New()

	handle_method := func(method string, urlpattern string, handler http.HandlerFunc) {
		m.Add(method, root+urlpattern, handler)
	}

	handle := func(urlpattern string, handler http.HandlerFunc) {
		m.Get(root+urlpattern, handler)
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

	handle("admin/dump/all.json/:invalid", http.NotFound)
	handle("admin/dump/all.json", layout.JSONAllEntries(blog_config))
	handle("admin/dump/:invalid", http.NotFound)

	handle("admin/gettext/:name/:invalid", http.NotFound)
	handle("admin/gettext/:name", layout.GetPostText(blog_config))
	handle_method("POST", "admin/render/:invalid", http.NotFound)
	handle_method("POST", "admin/render/", layout.RenderPost(blog_config))

	// pat seems to interfere with the blobstore's MIME parsing
	http.HandleFunc(root+"admin/post", layout.Save(blog_config))

	handle("sitemap.xml", layout.GetSitemap(blog_config))

	m.Get("/index.rss", http.HandlerFunc(layout.GetRSS(blog_config)))

	// matching on / will match all URLs
	// so you have to catch invalid top-level URLs first

	handle(":invalid/", http.NotFound)
	handle("", layout.IndexListHandler(blog_config, "list"))

	http.Handle(root, m)
}
