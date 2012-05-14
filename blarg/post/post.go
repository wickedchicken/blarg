package post

import (
    "appengine"
    "appengine/datastore"
    "time"
    // "encoding/json"
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
