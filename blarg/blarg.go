package blarg

import (
    "net/http"
    "pat"
    "mustache"
    "blackfriday"
    "bytes"
    "io"
    "appengine"
    "blarg/post"
    "blarg/config"
    "fmt"
)


//  key, err := post.SavePost(appcontext, "awesome post great job",
//    "chickens\n======\n\nchickens are *boss*. look at this:\n\n* chickens are tasty\n* chickens are not green\n* you too can be a chicken with focused thought",
//    make([]string, 0),
//    time.Now(),
//    "")

func List(w http.ResponseWriter, req *http.Request, blog_config map[string]interface{}) {
  appcontext := appengine.NewContext(req)

  bloginfo := config.Stringify(blog_config)

  template_dir := "templates/"

  h := mustache.RenderFile(template_dir + "header.html.mustache", bloginfo)
  io.WriteString(w, h)

  //context := map[string]string { "c": req.URL.Query().Get(":name") }
  //context := map[string]string { "c": "yolo!" }
  //c := mustache.RenderFile(template_dir + "splash.html.mustache", context)
  //io.WriteString(w, c)

  postchan := make(chan post.Post, 16)
  errchan := make(chan error)

  query := post.GetPostsSortedByDate()

  idx, err := post.GetCount(appcontext, query)
  if err != nil{
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
  }

  io.WriteString(w, fmt.Sprintf("total dudes = %v", idx))

  go post.ExecuteQuery(appcontext, query, 0, 20, postchan, errchan)


  for p := range postchan{
    con := bytes.NewBuffer(blackfriday.MarkdownCommon(bytes.NewBufferString(p.Content).Bytes())).String()
    context := map[string]string { "c": con }
    c := mustache.RenderFile(template_dir + "splash.html.mustache", context)
    io.WriteString(w, c)
  }

  err, ok := <-errchan
  if ok {
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
  }

  timing := map[string]string { "render": "0.01s" }
  f := mustache.RenderFile(template_dir + "footer.html.mustache", timing)
  io.WriteString(w, f)
}

func add_config(f func(http.ResponseWriter, *http.Request, map[string]interface{}), config map[string]interface{}) func(http.ResponseWriter, *http.Request){
  l := func(w http.ResponseWriter, req *http.Request){
    f(w, req, config)
  }

  return l
}

func add_seek(f func(http.ResponseWriter, *http.Request, int, int), start int, limit int) func(http.ResponseWriter, *http.Request){
  l := func(w http.ResponseWriter, req *http.Request){
    f(w, req, start, limit)
  }

  return l
}

func init() {
  blog_config,err := config.ReadJsonFile("blarg_config.json")
  if err != nil {
    panic(err)
  }

  //limit := blog_config["limit"]

  m := pat.New()
  m.Get("/", http.HandlerFunc(add_config(List, blog_config)))
  //m.Get("/list/:start", http.HandlerFunc(Index))
  // m.Get("/rss", http.HandlerFunc(Rss))
  // m.Get("/sitemap.xml", http.HandlerFunc(Sitemap))
  //m.Get("/item/:item", http.HandlerFunc(Lookup))
  //m.Get("/admin", http.HandlerFunc(Admin))

  // Register this pat with the default serve mux so that other packages
  // may also be exported. (i.e. /debug/pprof/*)
  http.Handle("/", m)
}
