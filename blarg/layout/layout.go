package layout

import (
    "net/http"
    "github.com/hoisie/mustache"
    "github.com/russross/blackfriday"
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

func labels(tags []string, blog_config map[string]interface{}) string{
  root := config.Stringify(blog_config["blog_root"])
  labels := bytes.NewBufferString("")

  for _,l := range tags{
    if l != "all posts" && l != "static" {
      fmt.Fprintf(labels, "<a href=\"%slabel/%s\">%s</a>", root, l, l)
    }
  }

  return string(labels.Bytes())
}

func list(w http.ResponseWriter, req *http.Request, blog_config map[string]interface{}, url_stem string, offset int, limit int, queries []*datastore.Query) {

  appcontext := appengine.NewContext(req)
  template_dir := "templates/"
  postchan := make(chan post.FullPost, 16)
  errchan := make(chan error)

  keys, err := post.UniquePosts(appcontext, queries)
  if err != nil{
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
  }

  idx := len(keys)

  pages := idx / limit
  if (idx % limit) > 0{
    pages += 1
  }

  curpage := offset / limit

  if idx < 1 {
    io.WriteString(w, fmt.Sprintf("<div class=\"entry\"><p>no posts found :(</p></div>"))
  } else {
    //go post.ExecuteQuery(appcontext, query, offset, limit, postchan, errchan)
    go post.GetPosts(appcontext, keys, offset, limit, postchan, errchan)

    myurl := appengine.DefaultVersionHostname(appcontext)
    scheme := "http://"
    if req.TLS != nil {
      scheme = "https://"
    }

    urlprefix := scheme + myurl + config.Stringify(blog_config["blog_root"])

    for p := range postchan{
      con := bytes.NewBuffer(blackfriday.MarkdownCommon(bytes.NewBufferString(p.PostStruct.Content).Bytes())).String()
      context := map[string]interface{} { "c": con, "labels": labels(p.Tags, blog_config),
                                          "link_to_entry": urlprefix + "article/" + p.PostStruct.StickyUrl,
                                          "post_date":  p.PostStruct.Postdate.Format("Jan 02 2006"),
                                          "post_title": p.PostStruct.Title}
      total_con := config.Stringify_map(config.Merge(blog_config, context))
      c := mustache.RenderFile(template_dir + "list_entry.html.mustache", total_con)
      io.WriteString(w, c)
    }

    err, ok := <-errchan
    if ok {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }

    context := map[string]interface{} {}

    root := config.Stringify(blog_config["blog_root"])

    context["prev_page"] = "&lt;&lt; prev"
    context["next_page"] = "next &gt;&gt;"

    if pages > 1 { context["pb"] = "true" }
    if curpage > 0 { context["prev_page"] = fmt.Sprintf("<a href=\"%v%v/%v\">&lt;&lt; prev</a>", root, url_stem, curpage - 1) }
    if curpage < (pages - 1) { context["next_page"] = fmt.Sprintf("<a href=\"%v%v/%v\">next &gt;&gt;</a>", root, url_stem, curpage + 1) }

    total_con := config.Stringify_map(config.Merge(blog_config, context))
    c := mustache.RenderFile(template_dir + "list.html.mustache", total_con)
    io.WriteString(w, c)
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

  appcontext := appengine.NewContext(req)
  tags, counts, err := post.GetAllTags(appcontext)
  if err != nil{
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
  }

  for i := range tags{
    n,count := tags[i], counts[i]
    context = map[string]interface{} { "label_link": "/label/" +n,
                                       "label_title": n,
                                       "label_count": fmt.Sprintf("%v",count)}
    total_con = config.Stringify_map(config.Merge(blog_config, context))
    c = mustache.RenderFile(template_dir + "sidebar_entry.html.mustache", total_con)
    io.WriteString(sidebar_topics, c)
  }

  context = map[string]interface{} { "sidebar_links": string(sidebar_links.Bytes()),
                                     "sidebar_topics": string(sidebar_topics.Bytes())}
  total_con = config.Stringify_map(config.Merge(blog_config, context))
  c = mustache.RenderFile(template_dir + "sidebar.html.mustache", total_con)
  io.WriteString(w, c)
}

func std_layout(blog_config map[string]interface{}, f func(w http.ResponseWriter, req *http.Request))func(w http.ResponseWriter, req *http.Request){

  p := func(w http.ResponseWriter, req *http.Request){
    appcontext := appengine.NewContext(req)
    myurl := appengine.DefaultVersionHostname(appcontext)
    scheme := "http://"
    if req.TLS != nil {
      scheme = "https://"
    }
    context := map[string]interface{} { "app_url": scheme + myurl + config.Stringify(blog_config["blog_root"])}


    bloginfo := config.Stringify_map(config.Merge(blog_config,context))

    start := time.Now()
    template_dir := "templates/"

    h := mustache.RenderFile(template_dir + "header.html.mustache", bloginfo)
    io.WriteString(w, h)

    f(w,req)

    sidebar(w, req, blog_config)

    delta := time.Since(start).Seconds()

    timing := map[string]interface{} { "timing": fmt.Sprintf("%0.2fs", delta) }
    if delta > 0.100 {
      timing["slow_code"] = "true"
    }
    bloginfo = config.Stringify_map(config.Merge(blog_config,timing))
    f := mustache.RenderFile(template_dir + "footer.html.mustache", bloginfo)
    io.WriteString(w, f)
  }

  return p
}

func IndexListHandler(blog_config map[string]interface{}, url_stem string)func(w http.ResponseWriter, req *http.Request){
  limit,err := getLimit(blog_config)
  if err != nil{
    panic(err)
  }

  l := func(w http.ResponseWriter, req *http.Request){
    list(w, req, blog_config, url_stem, 0, limit, post.GetPostsNotMatchingTag("static"))
  }
  return std_layout(blog_config, l)
}

func IndexPageHandler(blog_config map[string]interface{}, url_stem string)func(w http.ResponseWriter, req *http.Request){
  l := func(w http.ResponseWriter, req *http.Request){
    limit,err := getLimit(blog_config)
    if err != nil{
      panic(err)
    }
    offset, err := getOffset(req, limit, ":page")
    if err != nil{
        http.Error(w, "bad request: " + err.Error(), http.StatusBadRequest)
    } else {
      list(w, req, blog_config, url_stem, offset, limit, post.GetPostsNotMatchingTag("static"))
    }
  }
  return std_layout(blog_config, l)
}

func LabelPage(blog_config map[string]interface{}, url_stem string)func(w http.ResponseWriter, req *http.Request){
  l := func(w http.ResponseWriter, req *http.Request){
    label := req.URL.Query().Get(":label")
    context := map[string]interface{} { "search_label": label}
    total_con := config.Merge(blog_config, context)
    q := func(w http.ResponseWriter, req *http.Request){
      limit,err := getLimit(blog_config)
      if err != nil{
        panic(err)
      }
      offset, err := getOffset(req, limit, ":page")
      if err != nil{
          http.Error(w, "bad request: " + err.Error(), http.StatusBadRequest)
      } else {
        list(w, req, total_con, url_stem, offset, limit, post.GetPostsMatchingTag(label))
      }
    }
    std_layout(total_con, q)(w, req)
  }

  return l
}

func LabelList(blog_config map[string]interface{}, url_stem string)func(w http.ResponseWriter, req *http.Request){
  limit,err := getLimit(blog_config)
  if err != nil{
    panic(err)
  }

  l := func(w http.ResponseWriter, req *http.Request){
    label := req.URL.Query().Get(":label")
    context := map[string]interface{} { "search_label": label}
    total_con := config.Merge(blog_config, context)
    q := func(w http.ResponseWriter, req *http.Request){
      list(w, req, total_con, url_stem, 0, limit, post.GetPostsMatchingTag(label))
    }
    std_layout(total_con, q)(w, req)
  }

  return l
}

func GetArticle(blog_config map[string]interface{}, url_stem string)func(w http.ResponseWriter, req *http.Request){
  template_dir := "templates/"
  l := func(w http.ResponseWriter, req *http.Request){
    appcontext := appengine.NewContext(req)
    query := post.GetPostsMatchingUrl(req.URL.Query().Get(":name"))
    idx,err := post.GetCount(appcontext, query)

    if err != nil{
      http.Error(w, err.Error(), http.StatusInternalServerError)
    }

    if idx < 1 {
      http.Error(w, err.Error(), http.StatusNotFound)
      io.WriteString(w, fmt.Sprintf("<div class=\"entry\"><p>no posts found :(</p></div>"))
    } else {

      myurl := appengine.DefaultVersionHostname(appcontext)
      scheme := "http://"
      if req.TLS != nil {
        scheme = "https://"
      }

      urlprefix := scheme + myurl + config.Stringify(blog_config["blog_root"])

      var ps []post.Post
      keys, err := query.GetAll(appcontext, &ps)
      if err != nil{
        http.Error(w, err.Error(), http.StatusInternalServerError)
      }

      p := ps[0]
      tags, err := post.GetTagSlice(appcontext, keys[0])
      if err != nil{
        http.Error(w, err.Error(), http.StatusInternalServerError)
      }

      con := bytes.NewBuffer(blackfriday.MarkdownCommon(bytes.NewBufferString(p.Content).Bytes())).String()
      context := map[string]interface{} { "c": con, "labels": labels(tags, blog_config),
                                          "link_to_entry": urlprefix + "article/" + p.StickyUrl,
                                          "comment_display": "none",
                                          "post_date":  p.Postdate.Format("Oct 02 2006"),
                                          "post_title": p.Title}
      total_con := config.Stringify_map(config.Merge(blog_config, context))
      c := mustache.RenderFile(template_dir + "entry.html.mustache", total_con)
      io.WriteString(w, c)
    }
  }
  return std_layout(blog_config, l)
}


func GetSitemap(blog_config map[string]interface{})func(w http.ResponseWriter, req *http.Request){
  template_dir := "templates/"
  l := func(w http.ResponseWriter, req *http.Request){
    appcontext := appengine.NewContext(req)

    myurl := appengine.DefaultVersionHostname(appcontext)
    scheme := "http://"
    if req.TLS != nil {
      scheme = "https://"
    }

    urlprefix := scheme + myurl + config.Stringify(blog_config["blog_root"])

    postchan := make(chan post.Post, 16)
    errchan := make(chan error)
    go post.ExecuteQuery(appcontext, post.GetAllPosts(), -1, -1, func(post.Post)bool{ return true },postchan, errchan)

    entries := bytes.NewBufferString("")

    for p := range postchan{
      context := map[string]interface{} { "url": urlprefix + "article/" + p.StickyUrl,
                                          "lastmod_date":  p.Postdate.Format("2006-01-02")}
      total_con := config.Stringify_map(config.Merge(blog_config, context))
      c := mustache.RenderFile(template_dir + "sitemap_entry.mustache", total_con)
      io.WriteString(entries, c)
    }

    err, ok := <-errchan
    if ok {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
    context := map[string]interface{} {"content": entries}
    total_con := config.Stringify_map(config.Merge(blog_config, context))
    c := mustache.RenderFile(template_dir + "sitemap.mustache", total_con)
    io.WriteString(w, c)
  }
  return l
}
