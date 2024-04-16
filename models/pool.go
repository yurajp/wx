package models

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type User string
type Wscon struct {
	C *websocket.Conn
	sync.Mutex
}

type Ws = *websocket.Conn

func (u User) Avatar() string {
	user := string(u)
	path := "files/avatars/" + user
	res := "static/img/user.png"
	if _, err := os.Stat("server/" + path + ".jpg"); !os.IsNotExist(err) {
		res = path + ".jpg"
	} else if _, err := os.Stat("server" + path + ".png"); !os.IsNotExist(err) {
		res = path + ".png"
	}
	return res
}

type Pool struct {
	sync.Map
}

type Wait map[string]User

func (u User) IsValid() bool {
	return len([]rune(u)) >= 4
}

func (p *Pool) Size() int {
	size := 0
	p.Range(func(k, v any) bool {
		size++
		return true
	})
	return size
}

func (p *Pool) Contains(u User) bool {
	_, ok := p.Load(u)
	return ok
}

func (p *Pool) Register(ws Ws, user User) {
	p.Store(user, ws)
	Ach <- Attend{user, true}
	report := fmt.Sprintf(" 😊 %s joined %s", string(user), Stamp())
	p.Range(func(u, v any) bool {
		if u.(User) != user {
			wc := Wscon{C: ws}
			wc.Lock()
			wc.C.WriteMessage(1, []byte(report))
			wc.Unlock()
		}
		return true

	})

	go func() {
		suspence.Release(ws, user)
		shamess.Check(suspence)
	}()
}

func (p *Pool) Unregister(u User) {
	p.Delete(u)
	Ach <- Attend{u, false}
	log.Printf("User %s deleted", string(u))
	fmt.Printf(" %d users\n", p.Size())
	report := fmt.Sprintf(" ☹️ %s leave %s", string(u), Stamp())
	p.Range(func(k, v any) bool {
		wc := Wscon{C: v.(Ws)}
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
		wc := Wscon{C: v.(Ws)}
		wc.Lock()
		wc.C.WriteMessage(1, bmes)
		wc.Unlock()
		return true
	})
}

type Message struct {
	From User   `json:"from"`
	To   User   `json:"to"`
	Text string `json:"text"`
}

func (ms Message) StoreData() {
	path := "data/" + string(ms.From) + ".txt"
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		log.Printf("datafile: %s", err)
		return
	}
	defer f.Close()

	f.WriteString(idSaved(ms.Text))
}

func Stamp() string {
	return time.Now().Format("02 Jan 15:04")
}

func idSaved(htm string) string {
	htm = strings.TrimPrefix(htm, "<div")
	h := genHash()
	pre := fmt.Sprintf("<!--%s--><div id=%s", h, h)
	htm = pre + htm + "\n"
	return strings.Replace(htm, "✫", "✘", -1)

}
