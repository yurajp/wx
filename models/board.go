package models

import (
  "os"
  "log"
  "fmt"
  "encoding/json"
  "sync"
	"crypto/sha256"
	"encoding/base64"
	"html/template"
  
	"github.com/yurajp/wx/database"
	"github.com/gorilla/websocket"
)

const (
  NewUnewA = iota
  KnownUknownA
  KnownUnewA
  OtherUknownA
  InvalidInput
)

var CM ChatMap

type Auth struct {
  User User
  HPin string
}

type Registry map[string]Auth

type WC = *websocket.Conn
  
type Board struct {
  X sync.Mutex
  M map[User]WC
}

// NEW
type SidMap map[User][]string

type ChatMap map[User]SidMap


func (rg Registry) AddAuth(addr, name, pin string) {
   auth := Auth{User(name), HashPin(pin)}
   
   fmt.Printf("New auth: %+v\n", auth)
   
   rg[addr] = auth
   
   rg.Save()
}

func HashPin(pin string) string {
	hsh := sha256.Sum256([]byte(pin))
	return base64.URLEncoding.EncodeToString(hsh[:])
}

func (rg Registry) Save() {
  path := "data/registry"
  js, _ := json.Marshal(rg)
  os.WriteFile(path, js, 0640)
}

func LoadRegistry() Registry {
  path := "data/registry"
  if _, err := os.Stat(path); os.IsNotExist(err) {
    log.Print("Registry does not exist")
    return Registry{}
  }
  js, _ := os.ReadFile(path)
  var reg Registry
  json.Unmarshal(js, &reg)
  
  return reg
}

func (rg Registry) AllUsers() []User {
  users := []User{}
  for _, a := range rg {
    users = append(users, a.User)
  }
  return users
}

func (rg Registry) IsKnown(u User) bool {
  for _, auth := range rg {
    if auth.User == u {
      return true
    }
  }
  return false
}

func (rg Registry) OnRemote(remote string) (User, bool) {
  if auth, ok := rg[remote]; ok {
    return auth.User, true
  }
  return User(""), false
}

func (rg Registry) AuthorityCase(user User, remote string) int {
  isKnownU := rg.IsKnown(user)
  auth, isKnownA := rg[remote]
  if isKnownA {
    regU := auth.User
    if regU == user {
      return KnownUknownA
    }
    if isKnownU {
      return OtherUknownA
    }
  } else {
    if !isKnownU {
      return NewUnewA
    }
    if isKnownU {
      return KnownUnewA
    }
  }
  return InvalidInput
}

func (rg Registry) GetHPin(u User) string {
  for _, auth := range rg {
    if auth.User == u {
      return auth.HPin
    }
  }
  return ""
}

func (b *Board) Actives() int {
  i := 0
  for _, c := range b.M {
    if c != nil {
      i++
    }
  }
  return i
}

func (b *Board) AttendanceList() []string {
  res := []string{}
  for u, wc := range b.M {
    su := string(u)
    if wc != nil {
      su = "+" + su
    } else {
      su = "-" + su
    }
    res = append(res, su)
  }
  return res
}

func NewBoard(rg Registry) *Board {
  b := Board{X: sync.Mutex{}, M: make(map[User]WC)}
  
  for _, u := range rg.AllUsers() {
    b.M[u] = nil
  }
  if tm, err := template.ParseFiles("models/messages.tmpl"); err == nil {
    templ = tm
  } else {
    log.Printf("parsing messages.tmpl err: %v", err)
  }
  
  return &b
}

func NewChatMap(b *Board) *ChatMap {
  cm := make(ChatMap)
  for u := range b.M {
     sm := SidMap{}
     cm[u] = sm
  }
  return &cm
}


func ServerMessage(text string) *Message {
  m := NewMessage()
  m.Type = "server"
  m.Data = text
  return m
}

func (b *Board) Attach(u User, wc WC) {
  text := fmt.Sprintf("%s joined to chat", string(u))
  m := ServerMessage(text)
  b.BroadcastMessage(m)
  
  b.M[u] = wc
  b.X.Lock()
  b.X.Unlock()
  b.PublicateBoard()
  du := database.User(u)
  dms := database.S.LoadMessagesFor(du)
  ms := ConvertMessages(dms)
  
  b.SendMessagesTo(ms, u)
}

func ConvertMessages(dms []*database.Message) []*Message {
  mss := []*Message{}
  for _, dm := range dms {
    j, _ := json.Marshal(dm)
    var m Message
    json.Unmarshal(j, &m)
    mss = append(mss, &m)
  }
  return mss
}

func (b *Board) Detach(u User) {
  b.X.Lock()
  b.M[u] = nil
  b.X.Unlock()
  b.PublicateBoard()
  text := fmt.Sprintf("%s leaved chat", string(u))
  m := ServerMessage(text)
  log.Printf("User %v leave", u)
  b.BroadcastMessage(m)
}

func (b *Board) PublicateBoard() {
  atLst := b.AttendanceList()
  jl, _ := json.Marshal(atLst)
  hm := HtmlMessage{"users", DateNow(), string(jl)}
  jm, _ := json.Marshal(hm)
  for _, wc := range b.M {
    if wc == nil {
      continue
    }
    b.X.Lock()
    wc.WriteMessage(1, jm)
    b.X.Unlock()
  }
}

func (b *Board) BroadcastMessage(m *Message) {
  hm, err := m.ToHtmlMessage()
  if err != nil {
    log.Printf("templ execute error: %v", err)
    return
  }

  for _, wc := range b.M {
    if wc == nil {
      continue
    }
    b.X.Lock()
    wc.WriteMessage(1, hm)
    b.X.Unlock()
  }
}

func (b *Board) ResendMessage(m *Message) {
  hm, err := m.ToHtmlMessage()
  if err != nil {
    log.Printf("templ execute error: %v", err)
    return
  }
  ufr := m.From
  wcfr := b.M[ufr]
  b.X.Lock()
  wcfr.WriteMessage(1, hm)
  b.X.Unlock()
  
  uto := m.To
  wcto := b.M[uto]
  if wcto == nil {
    return
  }
  b.X.Lock()
  wcto.WriteMessage(1, hm)
  b.X.Unlock()
}

func FromDb(dm *database.Message) *Message {
  j, _ := json.Marshal(*dm)
  var m Message
  json.Unmarshal(j, &m)
  return &m
}

func (cm *ChatMap) Renew(u User, m *Message) {
  sidmap := CM[u]
  var companion User
  if m.To == u {
    companion = m.From
  } else {
    companion = m.To
  }
  ls := sidmap[companion]
  ls = append(ls, m.Sid)
  sidmap[companion] = ls
  CM[u] = sidmap
}


func (b *Board) SendMessagesTo(mss []*Message, u User ) {
  wc := b.M[u]
  if wc == nil {
    return
  }
  for _, m := range mss {
    if m.IsPrivate() {
      CM.Renew(u, m)
    }
    
    hm, err := m.ToHtmlMessage()
    if err != nil {
      log.Printf("message error: %v", err)
    }
    b.X.Lock()
    wc.WriteMessage(1, hm)
    b.X.Unlock()
  }
}

func (m *Message) CrossDb() database.Message {
  j, _ := json.Marshal(*m)
  var dm database.Message
  json.Unmarshal(j, &dm)
  
  return dm
}

func (b *Board) ListenChat(dataCh chan *Message) {
  defer close(dataCh)
  
  for {
    ms := <-dataCh
    if ms.Type == "" {
      continue
    }
    if ms.From != User("server") {
      dms := ms.CrossDb()
      database.S.Store(dms)
    }
    
    if ms.IsPrivate() {
      CM.Renew(ms.To, ms)
      CM.Renew(ms.From, ms)
      b.ResendMessage(ms)
    } else {
      b.BroadcastMessage(ms)
    }
  }
}

func (u User) Avatar() string {
	user := string(u)
	path := "files/avatars/" + user
	res := "static/img/user.png"
	if _, err := os.Stat("server/" + path + ".jpg"); !os.IsNotExist(err) {
		res = path + ".jpg"
	} else if _, err := os.Stat("server/" + path + ".png"); !os.IsNotExist(err) {
		res = path + ".png"
	}
	return res
}
