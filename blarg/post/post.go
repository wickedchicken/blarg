package post

import (
    "appengine"
    "appengine/datastore"
    "appengine/blobstore"
    "io/ioutil"
    "encoding/json"
    "time"
    "strings"
    "sort"
    "errors"
)

type Post struct {
  Title       string
  Content     appengine.BlobKey
  Postdate    time.Time
  StickyUrl   string
}

type Tag struct {
  Name        string
  Postdate    time.Time
  PostKey     *datastore.Key
}

type Tags []Tag

func (s Tags) Less(i, j int) bool { return s[i].Postdate.UnixNano() > s[j].Postdate.UnixNano() }
func (s Tags) Len() int { return len(s) }
func (s Tags) Swap(i, j int) { s[i], s[j] = s[j], s[i] }


func conv_title_to_url(title string) string{
  lc := strings.ToLower(title)

  strip := func(r rune) rune{
    switch {
      case 'a' <= r && r <= 'z':
        return r
      case '0' <= r && r <= '9':
        return r
      case r == ' ', r == '-', r == '_':
        return r
    }
    return ' '
  }

  return strings.Replace(strings.Map(strip, lc), " ", "-", -1)
}

func SavePost(context appengine.Context, title string, content appengine.BlobKey, tags []string, postdate time.Time) (*datastore.Key, error){
  p1 := Post{
      Title:    title,
      Content:  content,
      Postdate: postdate,
      StickyUrl: conv_title_to_url(title),
  }

  temppostkey := datastore.NewIncompleteKey(context, "post", nil)
  postkey,err := datastore.Put(context, temppostkey, &p1)
  if err != nil {
    return nil, err
  }

  t1 := Tag{
    Name:     "all posts",
    Postdate: postdate,
    PostKey:  postkey,
  }
  tagkey := datastore.NewIncompleteKey(context, "tag", nil)
  _,err = datastore.Put(context, tagkey, &t1)
  if err != nil {
    return nil, err
  }

  for _,t := range tags{
    t1 := Tag{
      Name:     t,
      Postdate: postdate,
      PostKey:  postkey,
    }
    tagkey := datastore.NewIncompleteKey(context, "tag", nil)
    _,err = datastore.Put(context, tagkey, &t1)
    if err != nil {
      return nil, err
    }
  }


  return postkey, nil
}

func GetTagSlice(context appengine.Context, key *datastore.Key) ([]string, error){
  var tagstructs []Tag
  _,err := (GetTagsAssociatedWithPost(key).GetAll(context, &tagstructs))
  if err != nil { return nil, err }

  tags := make([]string, len(tagstructs))
  for i,ts := range tagstructs{
    tags[i] = ts.Name
  }

  return tags, nil
}

func GetPostContent(context appengine.Context, p Post)(string, error){

  data, err := ioutil.ReadAll(blobstore.NewReader(context, p.Content))
  if err != nil {
    context.Errorf("ioutil.ReadAll: %v", err)
    return "", err
  }
  if len(data) <= 0{
    context.Errorf("len(data): %v", len(data))
    return "", errors.New("len(data) < 1")
  }

  var decoded interface{}
  err = json.Unmarshal(data, &decoded)
  if err != nil {
    context.Errorf("json.Unmarshal: %v", err)
    return "", err
  }

  q := decoded.(map[string]interface{})
  content,ok := q["data"].(string)
  if !ok{
    context.Errorf("post content has no 'data' field internally")
    return "", errors.New("post has no 'data' field")
  }

  return content, nil
}

func GetPost(context appengine.Context, key *datastore.Key) (Post, string, []string, error){
  var p2 Post
  err := datastore.Get(context, key, &p2)
  if err != nil { return p2, "", nil, err }
  tags, err := GetTagSlice(context, key)
  if err != nil { return p2, "", nil, err }
  content, err := GetPostContent(context, p2)
  if err != nil { return p2, "", nil, err }

  return p2, content, tags, err
}

func UniquePosts(context appengine.Context, queries []*datastore.Query) ([]*datastore.Key, error){
  seen := map[string]*datastore.Key {}

  alltags := make(Tags, 0)
  for _,query := range queries{
    var newtags Tags
    _,err := query.GetAll(context, &newtags)
    if err != nil { return nil, err }

    alltags = append(alltags, newtags...)
  }

  sort.Sort(alltags)

  keys := make([]*datastore.Key, 0)

  for _,t := range alltags{
    if _, ok := seen[t.PostKey.String()]; !ok{
      seen[t.PostKey.String()] = t.PostKey
      keys = append(keys, t.PostKey)
    }
  }

  return keys, nil
}

type FullPost struct {
  PostStruct Post
  Tags []string
  Content string
}

func GetPosts(c appengine.Context, keys []*datastore.Key, start int, limit int, out chan<- FullPost, errout chan<- error){

  defer close(out)
  defer close(errout)

  if start > len(keys){ start = len(keys) }
  if start+limit > len(keys){ limit = len(keys) - start }

  for _,t := range keys[start:start+limit]{
    post, content, tags, err := GetPost(c, t)

    if err != nil{
      errout <- err
      return
    }

    p1 := FullPost{
      PostStruct: post,
      Tags: tags,
      Content: content,
    }

    out <- p1
  }
}

func ExecuteQuery(c appengine.Context, q *datastore.Query, start int, limit int, filter func(Post) bool, out chan<- Post, errout chan<- error){

  defer close(out)
  defer close(errout)

  for t,i := q.Run(c), 0;  ; i++ {
    if ((start < 0) || (limit < 0)) || (i < (start + limit)){
      var x Post
      // key, err := t.Next(&x)
      _, err := t.Next(&x)

      if err == datastore.Done {
        return
      }
      if err != nil {
        errout <- err
        return
      }

      if !filter(x) {
        i--
        continue
      }

      if i < start {
        continue
      }

      out <- x
    } else {
      break
    }
  }
}

func GetCount(c appengine.Context, q *datastore.Query) (int, error){
  return q.Count(c)
}

func GetPostsNotMatchingTag(tag string) ([]*datastore.Query){
  queries := []*datastore.Query { datastore.NewQuery("tag").Filter("Name <", tag),
                                  datastore.NewQuery("tag").Filter("Name >", tag),
                                }
  return queries
}

func GetPostsMatchingTag(tag string) ([]*datastore.Query){
  return []*datastore.Query {datastore.NewQuery("tag").Filter("Name =", tag)}
}

func GetPostsSortedByDate() ([]*datastore.Query){
  return []*datastore.Query {datastore.NewQuery("post").Order("-Postdate")}
}

func GetTagsAssociatedWithPost(postkey *datastore.Key) (*datastore.Query){
  return datastore.NewQuery("tag").Filter("PostKey =", postkey).Order("Name")
}

func GetPostsMatchingUrl(stickyurl string) (*datastore.Query){
  return datastore.NewQuery("post").Filter("StickyUrl =", stickyurl).Limit(1)
}

func GetAllPosts() (*datastore.Query){
  return datastore.NewQuery("post")
}

func GetLatestDate(context appengine.Context)(time.Time){
  q := datastore.NewQuery("post").Order("-Postdate")
  var mePosts []Post
  _,err := q.Limit(1).GetAll(context, &mePosts)
  if err != nil { return time.Now() }
  if len(mePosts) < 1{
    return time.Now()
  }
  return mePosts[0].Postdate
}

func GetAllTags(context appengine.Context) ([]string,[]int, error){
  seen := map[string]int {}

  query := datastore.NewQuery("tag")
  var rawtags Tags
  _,err := query.GetAll(context, &rawtags)
  if err != nil { return nil, nil, err }

  for _,t := range rawtags{
    if _,ok := seen[t.Name]; !ok {
      seen[t.Name] = 1
    } else {
      seen[t.Name] += 1
    }
  }

  delete(seen, "all posts")
  delete(seen, "static")

  tags := make([]string, len(seen))
  counts := make([]int, len(seen))

  i := 0
  for k := range seen {
    tags[i] = k
    i += 1
  }

  sort.Strings(tags)

  for i,v := range tags {
    counts[i] = seen[v]
  }

  return tags, counts, nil
}
