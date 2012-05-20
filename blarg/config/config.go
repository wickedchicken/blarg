package config

import (
  "fmt"
  "encoding/json"
  "os"
)

func ReadJsonFile(filename string) (map[string]interface {}, error){
  f, err := os.Open(filename)
  if err != nil {
    return nil, err
  }
  defer f.Close()

  dec := json.NewDecoder(f)

  var config map[string]interface{}
  if err := dec.Decode(&config); err != nil {
    return nil, err
  }

  return config, nil
}

// anything in child overrides parent
func Merge(parent map[string]interface {}, child map[string]interface {}) map[string]interface {}{
  newmap := map[string]interface {} {}

  for k,v := range parent {
    newmap[k] = v
  }

  for k,v := range child {
    newmap[k] = v
  }

  return newmap
}

func Stringify(i interface{}) string{
  return fmt.Sprintf("%v", i)
}

func Stringify_map(m map[string]interface {}) map[string]string {
  newmap := map[string]string {}

  for k,v := range m{
    newmap[k] = Stringify(v)
  }

  return newmap
}
