package post

import (
    "encoding/json"
    "errors"
    "io/ioutil"
    "strings"
    "time"

    "appengine"
    "appengine/blobstore"
    "appengine/datastore"
)

type Post struct {
  Title       string
  Content     appengine.BlobKey
  Postdate    time.Time
  StickyUrl   string
  Tags        []string
}

type TagIndex struct {
  Tags        []string
  Postdate    time.Time
}

type FullPost struct {
  Post        Post
  Content     string
}

type TagCounts struct{
  Name        []string
  Count       []int
}

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

func NullFilter(Post) bool { return true }

func CalculateTagCounts(context appengine.Context) (map[string]int, error){
  postchan := make(chan Post, 16)
  errchan := make(chan error)

  go ExecuteQuery(context, GetAllPosts(), -1, -1, NullFilter, postchan, errchan)

  tags := map[string]int {}
  for p := range postchan{
    for t := range p.Tags{
      c,ok := tags[p.Tags[t]]
      if ok{
        tags[p.Tags[t]] = c + 1
      } else {
        tags[p.Tags[t]] = 1
      }
    }
  }

  err, ok := <-errchan
  if ok {
    return nil, err
  }

  return tags, nil
}

func SavePost(context appengine.Context, title string, content appengine.BlobKey, tags []string, postdate time.Time) (*datastore.Key, error){
  // transaction

  temppostkey := datastore.NewIncompleteKey(context, "post", nil)

  p1 := Post{
      Title:    title,
      Content:  content,
      Postdate: postdate,
      StickyUrl: conv_title_to_url(title),
      Tags:     tags,
  }

  postkey,err := datastore.Put(context, temppostkey, &p1)
  if err != nil {
    return nil, err
  }

  tagkey := datastore.NewIncompleteKey(context, "tagindices", postkey)
  t1 := TagIndex{
    Tags: tags,
    Postdate:  postdate,
  }
  _,err = datastore.Put(context, tagkey, &t1)
  if err != nil {
    return nil, err
  }

  tagcounts, err := CalculateTagCounts(context)
  if err != nil {
    return nil, err
  }

  var name []string
  var count []int
  for k,v := range tagcounts{
    name = append(name, k)
    count = append(count, v)
  }

  taggggkey := datastore.NewKey(context, "tagcounts", "", 1, nil)
  t2 := TagCounts{
    Name: name,
    Count: count,
  }
  _,err = datastore.Put(context, taggggkey, &t2)
  if err != nil {
    return nil, err
  }

  // end transaction

  return postkey, nil
}

func GetTagSlice(context appengine.Context, key *datastore.Key) ([]string, error){
  var p2 Post
  err := datastore.Get(context, key, &p2)
  if err != nil { return nil, err }

  return p2.Tags, nil
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
  // double Get of post, fix this
  tags,err := GetTagSlice(context, key)
  if err != nil { return p2, "", nil, err }
  content, err := GetPostContent(context, p2)
  if err != nil { return p2, "", nil, err }

  return p2, content, tags, err
}

func GetPosts(c appengine.Context, keys []*datastore.Key, start int, limit int, out chan<- FullPost, errout chan<- error){

  defer close(out)
  defer close(errout)

  if start > len(keys){ start = len(keys) }
  if start+limit > len(keys){ limit = len(keys) - start }

  for _,t := range keys[start:start+limit]{
    post, content, _, err := GetPost(c, t)

    if err != nil{
      errout <- err
      return
    }

    p1 := FullPost{
      Post: post,
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

func GetPostsMatchingTagCurried(tag string) (func (appengine.Context) ([]*datastore.Key, error)){
  return func(c appengine.Context) ([]*datastore.Key, error) {
    return GetPostsMatchingTag(c, tag)
  }
}

func GetPostsMatchingTag(c appengine.Context, tag string) ([]*datastore.Key, error){
  query := datastore.NewQuery("tagindices").Filter("Tags =", tag).Order("-Postdate").KeysOnly()

  tagindices, err := query.GetAll(c, nil)
  if err != nil {
    return nil, err
  }

  keys := make([]*datastore.Key, len(tagindices))
  for i,k := range tagindices{
    keys[i] = k.Parent()
  }

  return keys, nil
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
  query := datastore.NewQuery("tagcounts")
  var rawtags []TagCounts
  _,err := query.GetAll(context, &rawtags)
  if err != nil { return nil, nil, err }
  if len(rawtags) > 1 { return nil, nil, errors.New("not only one tagcounts struct") }
  if len(rawtags) < 1 { return make([]string, 0), make([]int, 0), nil }
  if len(rawtags[0].Name) != len(rawtags[0].Count){ return nil, nil, errors.New("tagcounts Name length doesn't match Counts length") }

  tags := rawtags[0].Name
  counts := rawtags[0].Count
  return tags, counts, nil
}
