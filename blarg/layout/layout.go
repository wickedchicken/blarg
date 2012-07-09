package layout

import (
    "net/http"
    "net/url"
    "bytes"
    "io"
    "io/ioutil"
    "encoding/json"
    "appengine"
    "blarg/post"
    "blarg/config"
    "fmt"
    "strings"
    "appengine/datastore"
    "appengine/user"
    "time"
    "appengine/channel"
    "appengine/blobstore"
    "math/rand"
    "github.com/russross/blackfriday"
    "github.com/hoisie/mustache"
)

func realhostname(req *http.Request, c appengine.Context)(string, error){
  if req.RequestURI == ""{
    return appengine.DefaultVersionHostname(c), nil
  }
  myurl, err := url.Parse(req.RequestURI)
  if err != nil{ return "", err }
  if !myurl.IsAbs() {
    return appengine.DefaultVersionHostname(c), nil
  }
  return myurl.Host, nil
}

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
      fmt.Fprintf(labels, "<a href=\"%slabel/%s\">%s</a> ", root, l, l)
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

    myurl, err := realhostname(req, appcontext)
    if err != nil{
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    scheme := "http://"
    if req.TLS != nil {
      scheme = "https://"
    }

    urlprefix := scheme + myurl + config.Stringify(blog_config["blog_root"])

    for p := range postchan{
      con := bytes.NewBuffer(blackfriday.MarkdownCommon(bytes.NewBufferString(p.Content).Bytes())).String()
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
        return
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
    myurl, err := realhostname(req, appcontext)
    if err != nil{
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
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

    u := user.Current(appcontext)
    if u != nil{
      logout,err := user.LogoutURL(appcontext, "/")
      if err != nil {
          http.Error(w, "error generating logout URL!", http.StatusInternalServerError)
          appcontext.Errorf("user.LogoutURL: %v", err)
          return
      }
      timing["me"] = fmt.Sprintf("%s", u)
      timing["logout"] = logout
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
        return
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
          return
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
      return
    }

    if idx < 1 {
      http.Error(w, err.Error(), http.StatusNotFound)
      io.WriteString(w, fmt.Sprintf("<div class=\"entry\"><p>no posts found :(</p></div>"))
      return
    } else {

      myurl, err := realhostname(req, appcontext)
      if err != nil{
          http.Error(w, err.Error(), http.StatusInternalServerError)
          return
      }
      scheme := "http://"
      if req.TLS != nil {
        scheme = "https://"
      }

      urlprefix := scheme + myurl + config.Stringify(blog_config["blog_root"])

      var ps []post.Post
      keys, err := query.GetAll(appcontext, &ps)
      if err != nil{
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
      }

      p := ps[0]
      tags, err := post.GetTagSlice(appcontext, keys[0])
      if err != nil{
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
      }

      content, err := post.GetPostContent(appcontext, p)
      if err != nil{
        http.Error(w, err.Error(), http.StatusInternalServerError)
        appcontext.Errorf("post.GetPostContent: %v", err)
        return
      }
      con := bytes.NewBuffer(blackfriday.MarkdownCommon(bytes.NewBufferString(content).Bytes())).String()
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

    myurl, err := realhostname(req, appcontext)
    if err != nil{
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    scheme := "http://"
    if req.TLS != nil {
      scheme = "https://"
    }

    urlprefix := scheme + myurl + config.Stringify(blog_config["blog_root"])

    postchan := make(chan post.Post, 16)
    errchan := make(chan error)
    go post.ExecuteQuery(appcontext, post.GetAllPosts(), -1, -1, func(post.Post)bool{ return true },postchan, errchan)

    entries := bytes.NewBufferString("")

    me_context := map[string]interface{} { "url": urlprefix,
                                        "lastmod_date":  post.GetLatestDate(appcontext).Format("2006-01-02")}
    me_total_con := config.Stringify_map(config.Merge(blog_config, me_context))
    me_c := mustache.RenderFile(template_dir + "sitemap_entry.mustache", me_total_con)
    io.WriteString(entries, me_c)
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
        return
    }
    context := map[string]interface{} {"content": entries}
    total_con := config.Stringify_map(config.Merge(blog_config, context))
    c := mustache.RenderFile(template_dir + "sitemap.mustache", total_con)
    io.WriteString(w, c)
  }
  return l
}

func entrybar(blog_config map[string]interface{}, w http.ResponseWriter, req *http.Request){
  template_dir := "templates/"
  appcontext := appengine.NewContext(req)
  myurl, err := realhostname(req, appcontext)
  if err != nil{
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
  }
  scheme := "http://"
  if req.TLS != nil {
    scheme = "https://"
  }

  urlprefix := scheme + myurl + config.Stringify(blog_config["blog_root"])

  postchan := make(chan post.Post, 16)
  errchan := make(chan error)
  go post.ExecuteQuery(appcontext, post.GetAllPosts(), -1, -1, func(post.Post)bool{ return true },postchan, errchan)

  for p := range postchan{
    context := map[string]interface{} { "url": urlprefix + "article/" + p.StickyUrl,
                                        "lastmod_date":  p.Postdate.Format("2006-01-02")}
    total_con := config.Stringify_map(config.Merge(blog_config, context))
    c := mustache.RenderFile(template_dir + "edit_entrybar.html.mustache", total_con)
    io.WriteString(w, c)
  }

  err, ok := <-errchan
  if ok {
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
  }
}

func admin_layout(blog_config map[string]interface{}, f func(w http.ResponseWriter, req *http.Request))func(w http.ResponseWriter, req *http.Request){

  p := func(w http.ResponseWriter, req *http.Request){
    appcontext := appengine.NewContext(req)
    myurl, err := realhostname(req, appcontext)
    if err != nil{
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    scheme := "http://"
    if req.TLS != nil {
      scheme = "https://"
    }
    context := map[string]interface{} { "app_url": scheme + myurl + config.Stringify(blog_config["blog_root"])}


    logout,err := user.LogoutURL(appcontext, "/")
    if err != nil {
        http.Error(w, "error generating logout URL!", http.StatusInternalServerError)
        appcontext.Errorf("user.LogoutURL: %v", err)
        return
    }
    u := user.Current(appcontext)
    context["me"] = fmt.Sprintf("%s", u)
    context["logout"] = logout

    bloginfo := config.Stringify_map(config.Merge(blog_config,context))

    start := time.Now()
    template_dir := "templates/"

    h := mustache.RenderFile(template_dir + "header.admin.html.mustache", bloginfo)
    io.WriteString(w, h)

    f(w,req)

    delta := time.Since(start).Seconds()

    timing := map[string]interface{} { "timing": fmt.Sprintf("%0.2fs", delta) }
    if delta > 0.100 {
      timing["slow_code"] = "true"
    }

    bloginfo = config.Stringify_map(config.Merge(blog_config,timing))
    f := mustache.RenderFile(template_dir + "footer.admin.html.mustache", bloginfo)
    io.WriteString(w, f)
  }
  return p
}

func EditPost(blog_config map[string]interface{})func(w http.ResponseWriter, req *http.Request){
  template_dir := "templates/"
  l := func(w http.ResponseWriter, req *http.Request){
    appcontext := appengine.NewContext(req)

    context := map[string]interface{} {}

    name := req.URL.Query().Get(":name")
    if name != ""{
      query := post.GetPostsMatchingUrl(req.URL.Query().Get(":name"))
      idx,err := post.GetCount(appcontext, query)

      if err != nil{
        http.Error(w, err.Error(), http.StatusInternalServerError)
      }

      if idx < 1 {
        context["articlesource"] = "Enter your post here!"
        context["labels"] = ""
        context["title"] = "No post with that URL found, making new post!"
      } else {
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

        context["articlesource"] = p.Content
        context["labels"] = strings.Join(tags, ", ")
        context["title"] = p.Title
      }
    }  else {
      context["articlesource"] = "Enter your post here!"
      context["labels"] = "enter, labels, here"
      context["title"] = "enter title here"
    }

    key := fmt.Sprintf("%v", rand.Int())
    tok, err := channel.Create(appcontext, key)
    if err != nil {
        http.Error(w, "Couldn't create Channel", http.StatusInternalServerError)
        appcontext.Errorf("channel.Create: %v", err)
        return
    }

    posturl := "/admin/post?g=" + key
    uploadurl, err := blobstore.UploadURL(appcontext, posturl, nil)
    if err != nil {
        http.Error(w, "Couldn't create blob URL", http.StatusInternalServerError)
        appcontext.Errorf("blobstore.UploadURL: %v", err)
        return
    }

    context["token"] = tok
    context["session"] = key
    context["uploadurl"] = uploadurl

    total_con := config.Stringify_map(config.Merge(blog_config, context))
    c := mustache.RenderFile(template_dir + "edit.html.mustache", total_con)
    io.WriteString(w, c)

    if err != nil {
        appcontext.Errorf("mainTemplate: %v", err)
    }
  }
  return admin_layout(blog_config, l)
}

func GetPostText(blog_config map[string]interface{}) func(w http.ResponseWriter, req *http.Request){
  l := func(w http.ResponseWriter, req *http.Request){
    appcontext := appengine.NewContext(req)
    query := post.GetPostsMatchingUrl(req.URL.Query().Get(":name"))
    idx,err := post.GetCount(appcontext, query)

    if err != nil{
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
    }

    if idx < 1 {
      http.Error(w, err.Error(), http.StatusNotFound)
      io.WriteString(w, fmt.Sprintf("<div class=\"entry\"><p>no posts found :(</p></div>"))
    } else {
      var ps []post.Post
      _, err := query.GetAll(appcontext, &ps)
      if err != nil{
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
      }

      p := ps[0]

      content, err := post.GetPostContent(appcontext, p)
      if err != nil{
        http.Error(w, err.Error(), http.StatusInternalServerError)
        appcontext.Errorf("post.GetPostContent: %v", err)
        return
      }
      con := bytes.NewBuffer(blackfriday.MarkdownCommon(bytes.NewBufferString(content).Bytes())).String()
      io.WriteString(w, con)
    }
  }
  return l
}

func RenderPost(blog_config map[string]interface{}) func(w http.ResponseWriter, req *http.Request){
  l := func(w http.ResponseWriter, req *http.Request){
    appcontext := appengine.NewContext(req)
    err := req.ParseMultipartForm(1024 * 1024)
    if err != nil {
      appcontext.Errorf("sending markdown payload: %v", err)
      http.Error(w, err.Error(), http.StatusBadRequest)
      return
    }
    dataFile, _, err := req.FormFile("data")
    if err != nil{
      http.Error(w, "did not specify data as a param", http.StatusBadRequest)
      return
    }

    data, err := ioutil.ReadAll(dataFile)
    if err != nil{
      appcontext.Errorf("ioutil.ReadAll(): %v", err)
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
    }

    if len(data) < 1{
      appcontext.Errorf("len(data): %v", len(data))
      http.Error(w, "did not specify data as a param", http.StatusBadRequest)
      return
    }

    key := req.FormValue("g")
    if key == ""{
      http.Error(w, "did not specify a key!", http.StatusBadRequest)
      return
    }

    var decoded interface{}
    err = json.Unmarshal(data, &decoded)
    if err != nil {
      http.Error(w, err.Error(), http.StatusBadRequest)
      return
    }

    q := decoded.(map[string]interface{})
    str,ok := q["data"]
    if !ok{
      http.Error(w, "error: must supply JSON with 'data' specified!", http.StatusBadRequest)
      return
    }

    con := map[string]interface{} {"markdown": bytes.NewBuffer(blackfriday.MarkdownCommon(bytes.NewBufferString(fmt.Sprintf("%v", str)).Bytes())).String()}
    err = channel.SendJSON(appcontext, key, con)
    if err != nil {
        appcontext.Errorf("sending markdown payload: %v", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
  }
  return l
}

func Save(blog_config map[string]interface{}) func(w http.ResponseWriter, req *http.Request){
  l := func(w http.ResponseWriter, req *http.Request){
    appcontext := appengine.NewContext(req)
    blobs, _, err := blobstore.ParseUpload(req)
    if err != nil {
      appcontext.Errorf("error parsing blobstore! %v", err)
      http.Error(w, "error parsing blobstore!", http.StatusBadRequest)
      return
    }

    //key := "coolguys"
    key := req.FormValue("g")
    if key == ""{
      http.Error(w, "did not specify a key!", http.StatusBadRequest)
      return
    }


    send_message := func(stat string, color string){
      con := map[string]interface{} {"status": stat, "color": color}
      err := channel.SendJSON(appcontext, key, con)
      if err != nil {
          appcontext.Errorf("sending update message: %v", err)
          http.Error(w, err.Error(), http.StatusInternalServerError)
      }
    }

    bdata, ok := blobs["data"]
    if !ok{
      http.Error(w, "did not specify data as a param", http.StatusBadRequest)
      send_message("internal error while saving!", "#AA0000")
      return
    }
    if len(bdata) != 1 {
      appcontext.Errorf("error parsing blobstore!", err)
      http.Error(w, "error parsing blobstore!", http.StatusBadRequest)
      return
    }
    jsonkey := bdata[0].BlobKey
    data, err := ioutil.ReadAll(blobstore.NewReader(appcontext, jsonkey))
    if err != nil {
      appcontext.Errorf("error parsing blobstore!", err)
      http.Error(w, "error parsing blobstore!", http.StatusBadRequest)
      return
    }
    if len(data) <= 0{
      http.Error(w, "did not specify data as a param", http.StatusBadRequest)
      send_message("internal error while saving!", "#AA0000")
      return
    }

    var decoded interface{}
    err = json.Unmarshal(data, &decoded)
    if err != nil {
      http.Error(w, err.Error(), http.StatusBadRequest)
      send_message("internal error while saving!", "#AA0000")
      return
    }

    q := decoded.(map[string]interface{})
    _,ok = q["data"].(string)
    if !ok{
      http.Error(w, "error: must supply JSON with 'data' specified!", http.StatusBadRequest)
      send_message("internal error while saving!", "#AA0000")
      return
    }
    title,ok := q["title"].(string)
    if !ok{
      http.Error(w, "error: must supply JSON with 'title' specified!", http.StatusBadRequest)
      send_message("internal error while saving!", "#AA0000")
      return
    }
    labels,ok := q["labels"].(string)
    if !ok{
      http.Error(w, "error: must supply JSON with 'labels' specified!", http.StatusBadRequest)
      send_message("internal error while saving!", "#AA0000")
      return
    }
    individual_labels := strings.Split(labels, ",")
    real_labels := make([]string, len(individual_labels))
    for i := range individual_labels{
      real_labels[i] = strings.ToLower(strings.Trim(individual_labels[i], " \t"))
    }

    _, err = post.SavePost(appcontext, title, jsonkey, real_labels, time.Now());
    if err != nil{
      appcontext.Errorf("saving a post: %v", err)
      http.Error(w, err.Error(), http.StatusInternalServerError)
      send_message("internal error while saving!", "#AA0000")
      return
    }

    send_message("saved", "#00AA00")
  }
  return l
}
