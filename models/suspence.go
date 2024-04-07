package models

import (
  "log"
  "os"
  "encoding/json"
  "fmt"
  "encoding/base64"
  "crypto/sha1"
  "time"
  "strings"
)

type Suspence map[User][]string

type DateMess struct {
  Message
  Date string
}

type ShaMess map[string]DateMess


var shamess ShaMess
var suspence Suspence


func LoadSha() {
  path := "data/shamess"
  if _, err := os.Stat(path); os.IsNotExist(err) {
    os.Create(path)
    shamess = ShaMess{}
    return
  }
  js, err := os.ReadFile(path)
  if err != nil {
    log.Printf("load shamess: %s", err)
    shamess = ShaMess{}
    return
  }
  if len(js) == 0 {
    shamess = ShaMess{}
    return
  }
  err = json.Unmarshal(js, &shamess)
  if err != nil {
    log.Println(err)
    shamess = ShaMess{}
  }
}

func LoadSus() {
  path := "data/suspence"
  if _, err := os.Stat(path); os.IsNotExist(err) {
    os.Create(path)
    suspence = Suspence{}
    return
  }
  js, err := os.ReadFile(path)
  if err != nil {
    log.Printf("load suspence: %s", err)
    suspence = Suspence{}
    return
  }
  if len(js) == 0 {
    suspence = Suspence{}
    return
  }
  err = json.Unmarshal(js, &suspence)
  if err != nil {
    log.Println(err)
    suspence = Suspence{}
  }
}

func (ms Message) Suspend(offs []User) {
  fmt.Println("~ suspend: ", offs)
  hash := shamess.Add(ms)
  for _, u := range offs {
    ls, _ :=  suspence[u]
    suspence[u] = append(ls, hash)
  }
  go suspence.Update()
  
  go shamess.Update()
}

func genHash() string {
  nt := fmt.Sprintf("%d", time.Now().UnixNano())
  hash := sha1.New()
  hash.Write([]byte(nt))
  sha := base64.URLEncoding.EncodeToString(hash.Sum(nil))
  return string(sha[:8])
}

func (shm ShaMess) Add(ms Message) string {
  hsh := genHash()
  dm := DateMess{ms, Stamp()}
  shm[hsh] = dm
  return hsh
}

func (sus Suspence) Update() {
  js, err := json.Marshal(sus)
  if err != nil {
    log.Println(err)
    return
  }
  path := "data/suspence"
  err = os.WriteFile(path, js, 0640)
  if err != nil {
    log.Println(err)
    return
  }
//  log.Println("suspence updated")
}

func (shm ShaMess) Update() {
  js, err := json.Marshal(shm)
  if err != nil {
    log.Println(err)
    return
  }
  path := "data/shamess"
  err = os.WriteFile(path, js, 0640)
  if err != nil {
    log.Println(err)
    return
  }
//  log.Println("hashes updated")
}

func (sus Suspence) Release(wsc Wscon, u User) {
  if ls, ok := sus[u]; ok {
    i := 0
    for _, h := range ls {
      dm := shamess[h]
      var mes string
      if strings.HasPrefix(dm.Message.Text, "FILE@") {
        mes = dm.Message.Text
      } else {
        mes = fmt.Sprintf(" %s %s\n%s", string(dm.Message.From), dm.Date, dm.Message.Text)
      }
      wsc.Lock()
      wsc.C.WriteMessage(1, []byte(mes))
      wsc.Unlock()
      i++
    }
    sus[u] = []string{}
    if i > 0 {
      go sus.Update()
    }
  }
}

func (sh ShaMess) Check(sus Suspence) {
  ch := false
  for k := range sh {
    n := 0
    for _, ls := range sus {
      for _, h := range ls {
        if k == h {
          n++
          continue
        }
      }
    }
    if n == 0 {
      delete(sh, k)
      ch = true
    }
  }
  if ch {
    go sh.Update()
  }
}

func ClearUnreaded(u User) {
   shas := []string{}
   for sh, dm := range shamess {
    if dm.Message.From == u {
      shas = append(shas, sh)
      delete(shamess, sh)
    }
  }
  for k, ls := range suspence {
    if k == u {
      continue
    }
    for i := 0; i < len(ls) - 1; i++ {
      for _, s := range shas {
        if s == ls[i] {
          ls[i] = ls[len(ls)-1]
          ls = ls[:len(ls)-1]
        }
      }
    }
    suspence[k] = ls
  }
  go shamess.Update()
  go suspence.Update()
}
