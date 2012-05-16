package post

import (
    "appengine"
    "appengine/datastore"
    "time"
)

type Post struct {
    Title       string
    Content     string
    Tags        []string
    Postdate    time.Time
    StickyUrl   string
}

func SavePost(context appengine.Context, title string, content string, tags []string, postdate time.Time, stickyurl string) (*datastore.Key, error){

  p1 := Post{
      Title:    title,
      Content:  content,
      Tags:     tags,
      Postdate: postdate,
      StickyUrl: stickyurl,
  }

  return datastore.Put(context, datastore.NewIncompleteKey(context, "post", nil), &p1)
}

func GetPost(context appengine.Context, key *datastore.Key) (Post, error){
  var p2 Post
  err := datastore.Get(context, key, &p2)
  return p2, err
}

func ExecuteQuery(c appengine.Context, q *datastore.Query, start int, limit int, out chan<- Post, errout chan<- error){

  defer close(out)
  defer close(errout)

  for t,i := q.Run(c), 0; i < (start + limit) ; i++ {
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

    if i < start {
      continue
    }

    out <- x
  }
}

func GetCount(c appengine.Context, q *datastore.Query) (int, error){
  return q.Count(c)
}

func GetPostsSortedByDate() (*datastore.Query){
  return datastore.NewQuery("post").Order("-Postdate")
}

func GetPostsMatchingUrl(stickyurl string) (*datastore.Query){
  return datastore.NewQuery("post").Filter("StickyUrl =", stickyurl).Limit(1)
}

