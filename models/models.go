package models

import (
  "fmt"
  "log"
  "time"
  "sync"
  "strings"
  "net/http"
  "encoding/json"
  "os"
  
  "github.com/gorilla/websocket"
)

var (
  dataCh chan Message
  known = map[string]string{
    "127.0.0.1":"Local",
    "192.168.1.22":"Note",
    "192.168.1.21":"Note",
    "192.168.1.38":"Ubuntu",
    "192.168.1.19":"Tata",
  }
)

type User string
type Wscon struct {
  C *websocket.Conn
  sync.Mutex
}
type Pool struct {
  sync.Map
}
type Wait map[string]User

func (u User) IsValid() bool {
  if len([]rune(u)) < 4 {
    return false
  }
  return true
}

func (p *Pool) Size() int {
  size := 0
  p.Range(func(k, v any) bool {
    size++
    return true
  })
  return size
}

func (p *Pool) Known(r *http.Request) (User, bool) {
  radd := r.RemoteAddr
  rhost := strings.Split(radd, ":")[0]
  if u, ok := known[rhost]; ok && !p.Contains(string(u)) {
    return User(u), true
  }
  return User(""), false
}

func (p *Pool) Contains(u string) bool {
  _, ok := p.Load(u)
  return ok
}

func (p *Pool) Add(ws*websocket.Conn, user User) {
  p.Store(user, Wscon{C:ws})
  report := fmt.Sprintf(" 😊 %s joined %s", string(user), Stamp())
  p.Range(func(u, v any) bool {
    if u.(User) != user {
      wc := v.(Wscon)
      wc.Lock()
      wc.C.WriteMessage(1, []byte(report))
      wc.Unlock()
    }
    return true
  })
}

func (p *Pool) Unregister(u User) {
  p.Delete(u)
  log.Printf("User %s deleted", string(u))
  fmt.Printf(" %d users\n", p.Size())
  report := fmt.Sprintf(" ☹️ %s leave %s", string(u), Stamp())
  p.Range(func(k, v any) bool {
    wc := v.(Wscon)
    wc.Lock()
    wc.C.WriteMessage(1, []byte(report))
    wc.Unlock()
    return true
  })
}

func (p *Pool) Publicate() {
  list := []string{}
  p.Range(func(u, v interface{}) bool {
    list = append(list, string(u.(User)))
    return true
  })
  js, _ := json.Marshal(list)
  bmes := append([]byte("USERS"), js...)
  p.Range(func(k, v any) bool {
    wc := v.(Wscon)
    wc.Lock()
    wc.C.WriteMessage(1, bmes)
    wc.Unlock()
    return true
  })
}

func (p *Pool) BringOut(m Message) {
  var pub string
  if strings.HasPrefix(m.Text, "FILE@") {
    pub = m.Text
  } else {
    pub = fmt.Sprintf(" %s %s\n%s", string(m.From), Stamp(), m.Text)
  }
  p.Range(func(k, v interface{}) bool {
    wc := v.(Wscon)
    wc.Lock()
    wc.C.WriteMessage(websocket.TextMessage, []byte(pub))
    wc.Unlock()
    return true
  })
}

func (p *Pool) Resend(m Message) {
  to := m.To
  from := m.From
  ci, ok := p.Load(to)
  cfi, _ := p.Load(from)
  if !ok {
    mess := fmt.Sprintf("User %s not found", to)
    cfi.(Wscon).C.WriteMessage(websocket.TextMessage, []byte(mess))
    return 
  }
  var letter string
  if strings.HasPrefix(m.Text, "FILE@") {
    letter = m.Text
  } else {
    letter = fmt.Sprintf("%s PRIVATE from %s\n%s", Stamp(), string(from), m.Text)
  }
  cto := ci.(Wscon)
  cto.C.WriteMessage(websocket.TextMessage, []byte(letter))
  if to != from {
    cfi.(Wscon).C.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%s message to %s sended", Stamp(), string(to))))
  }
}

type Message struct {
  From User `json:"from"`
  To User `json:"to"`
  Text string `json:"text"`
}

func (ms Message) StoreData() {
  path := "data/" + string(ms.From) + ".txt"
  f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
  if err != nil {
    log.Printf("datafile: %w", err)
    return
  }
  defer f.Close()
  f.WriteString(ms.Text + "\n")
}

func (p *Pool) HandleData(dCh chan Message) {
  for {
   select {
    case ms := <-dCh:
      if (ms.To == "" || ms.To == "all") {
          p.BringOut(ms)
      } else if ms.To == "STORE" {
        log.Println("Message is saved")
        ms.StoreData()
      } else {
        p.Resend(ms)
      }
    default:
   }
  }
}

func Stamp() string {
  return time.Now().Format("15:04:05")
}
