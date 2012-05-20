package layout

import (
    "net/http"
    "mustache"
    "blackfriday"
    "bytes"
    "io"
    "appengine"
    "blarg/post"
    "blarg/config"
    "fmt"
    "appengine/datastore"
    "time"
)

func getLimit(blog_config map[string]interface{}) (int, error){
  var limit int
  if limtext,ok := blog_config["post_limit"]; ok {
    _,err := fmt.Sscanf(config.Stringify(limtext), "%d", &limit)
    if err != nil{
      return 0, err
    }
  } else {
    limit = 20
  }

  return limit, nil
}

func getOffset(req *http.Request, limit int, pagename string) (int, error){
  var page int
  _,err := fmt.Sscanf(req.URL.Query().Get(pagename), "%d", &page)
  if err != nil{
    return 0, err
  }

  offset := int(page) * limit

  return offset, nil
}

func list(w http.ResponseWriter, req *http.Request, blog_config map[string]interface{}, offset int, limit int, query *datastore.Query) {

  template_dir := "templates/"

  appcontext := appengine.NewContext(req)

//  _, err := post.SavePost(appcontext, "awesome post great job",
//    "chickens\n======\n\nchickens are *boss*. look at this:\n\n* chickens are tasty\n* chickens are not green\n* you too can be a chicken with focused thought",
//    make([]string, 0),
//    time.Now(),
//    "")
//  if err != nil{
//      http.Error(w, err.Error(), http.StatusInternalServerError)
//      return
//  }

  postchan := make(chan post.Post, 16)
  errchan := make(chan error)

  idx, err := post.GetCount(appcontext, query)
  if err != nil{
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
  }

  io.WriteString(w, fmt.Sprintf("<div class=\"entry\"><p>total dudes = %v</p></div>", idx))

  if idx < 1 {
    io.WriteString(w, fmt.Sprintf("<div class=\"entry\"><p>no posts found :(</p></div>"))
  } else {
    fmt.Printf("noooo %d %d\n", offset, limit)
    go post.ExecuteQuery(appcontext, query, offset, limit, postchan, errchan)

    for p := range postchan{
      fmt.Printf("yayyyyyy!\n")
      con := bytes.NewBuffer(blackfriday.MarkdownCommon(bytes.NewBufferString(p.Content).Bytes())).String()
      context := map[string]interface{} { "c": con }
      total_con := config.Stringify_map(config.Merge(blog_config, context))
      c := mustache.RenderFile(template_dir + "list_entry.html.mustache", total_con)
      io.WriteString(w, c)
    }

    err, ok := <-errchan
    if ok {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
  }
}

func sidebar(w http.ResponseWriter, req *http.Request, blog_config map[string]interface{}){
  template_dir := "templates/"

  sidebar_links := bytes.NewBufferString("")

  context := map[string]interface{} { "label_link": "http://qrunk.com",
                                      "label_title": "qrunk.com"}
  total_con := config.Stringify_map(config.Merge(blog_config, context))
  c := mustache.RenderFile(template_dir + "sidebar_entry.html.mustache", total_con)
  io.WriteString(sidebar_links, c)

  sidebar_topics := bytes.NewBufferString("")
  context = map[string]interface{} { "label_link": "/label/cool",
                                     "label_title": "cool"}
  total_con = config.Stringify_map(config.Merge(blog_config, context))
  c = mustache.RenderFile(template_dir + "sidebar_entry.html.mustache", total_con)
  io.WriteString(sidebar_topics, c)

  context = map[string]interface{} { "sidebar_links": string(sidebar_links.Bytes()),
                                     "sidebar_topics": string(sidebar_topics.Bytes())}
  total_con = config.Stringify_map(config.Merge(blog_config, context))
  c = mustache.RenderFile(template_dir + "sidebar.html.mustache", total_con)
  io.WriteString(w, c)
}

func std_layout(blog_config map[string]interface{}, f func(w http.ResponseWriter, req *http.Request))func(w http.ResponseWriter, req *http.Request){
  bloginfo := config.Stringify_map(blog_config)

  fmt.Printf("yess %s\n", bloginfo["blog_config"])

  p := func(w http.ResponseWriter, req *http.Request){
    start := time.Now()
    template_dir := "templates/"

    h := mustache.RenderFile(template_dir + "header.html.mustache", bloginfo)
    io.WriteString(w, h)

    f(w,req)

    sidebar(w, req, blog_config)

    delta := time.Since(start).Seconds()

    timing := map[string]string { "timing": fmt.Sprintf("%0.2fs", delta) }
    if delta > 0.100 {
      timing["slow_code"] = "true"
    }
    f := mustache.RenderFile(template_dir + "footer.html.mustache", timing)
    io.WriteString(w, f)
  }

  return p
}

func IndexListHandler(blog_config map[string]interface{})func(w http.ResponseWriter, req *http.Request){
  limit,err := getLimit(blog_config)
  if err != nil{
    panic(err)
  }

  l := func(w http.ResponseWriter, req *http.Request){

    list(w, req, blog_config, 0, limit, post.GetPostsSortedByDate())
  }
  return std_layout(blog_config, l)
}

func IndexPageHandler(blog_config map[string]interface{})func(w http.ResponseWriter, req *http.Request){
  l := func(w http.ResponseWriter, req *http.Request){
    limit,err := getLimit(blog_config)
    if err != nil{
      panic(err)
    }
    offset, err := getOffset(req, limit, ":page")
    if err != nil{
        http.Error(w, "bad request: " + err.Error(), http.StatusBadRequest)
    } else {
      list(w, req, blog_config, offset, limit, post.GetPostsSortedByDate())
    }
  }
  return std_layout(blog_config, l)
}


