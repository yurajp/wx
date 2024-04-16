package models

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

var (
	Reg Registry
	B   = Board{}
	Ach = make(chan Attend)
)

type Registry map[string]User

type Board []Attend

type Attend struct {
	User
	On bool
}

func (atd Attend) String() string {
	pre := "-"
	if atd.On {
		pre = "+"
	}
	return pre + string(atd.User)
}

func Remove(u User) {
	for i, at := range B {
		if u == at.User {
			B[i].On = false
		}
	}
}

func Listen(p *Pool) {
	for {
		a := <-Ach
		u := a.User
		for i, atd := range B {
			if atd.User == u {
				B[i] = a
				break
			}
		}
		Publicate(p)
	}
}

func (rg Registry) WriteFile() error {
	path := "data/registry"
	js, err := json.Marshal(rg)
	if err != nil {
		return fmt.Errorf("registry marshal: %s", err)
	}
	err = os.WriteFile(path, js, 0640)
	if err != nil {
		return fmt.Errorf("write registry: %s", err)
	}
	return nil
}

func (rg Registry) NewKnown(addr string, u User) {
	if _, ok := rg[addr]; !ok {
		Reg[addr] = u
	}
	go Reg.WriteFile()

}

func LoadRegistry() {
	path := "data/registry"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, _ := os.Create(path)
		f.Close()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("read registry: %s", err)
	}
	if len(data) == 0 {
		data = []byte("{}")
	}
	err = json.Unmarshal(data, &Reg)
	if err != nil {
		log.Printf("unmarshal registry: %s", err)
	}
	for _, us := range RegUsers() {
		B = append(B, Attend{us, false})
	}
}

func GetKnown(rhost string) (User, bool) {
	if u, ok := Reg[rhost]; ok {
		return u, true
	}
	return User(""), false
}

func (b Board) Print() {
	for _, atd := range b {
		fmt.Println(atd.String())
	}
}

func RegUsers() []User {
	us := []User{}
	mp := make(map[User]bool)
	for _, u := range Reg {
		mp[u] = true
	}
	for u := range mp {
		us = append(us, u)
	}
	return us
}

func ListOff() []User {
	offus := []User{}
	for _, atd := range B {
		if !atd.On {
			offus = append(offus, atd.User)
		}
	}
	return offus
}

func Publicate(p *Pool) {
	list := []string{}
	for _, atd := range B {
		list = append(list, atd.String())
	}
	js, err := json.Marshal(list)
	if err != nil {
		log.Printf("publicate: json: %s", err)
		return
	}
	bmes := append([]byte("USERS@"), js...)
	p.Range(func(k, v any) bool {
		wc := Wscon{C: v.(Ws)}
		wc.Lock()
		wc.C.WriteMessage(1, bmes)
		wc.Unlock()
		return true
	})
}

func HandleChat(dCh chan Message, p *Pool) {

	LoadRegistry()
	go Listen(p)

	for {
		ms := <-dCh
		if ms.To == "all" || ms.To == "All" {
			Broadcast(ms, p)
		} else if ms.To == "STORE" {
			ms.StoreData()
		} else {
			Resend(ms, p)
		}
	}
}

func Broadcast(ms Message, p *Pool) {
	offs := ListOff()

	if len(offs) > 0 {
		ms.Suspend(offs)
	}
	var pub string
	if strings.HasPrefix(ms.Text, "FILE@") {
		pub = ms.Text
	} else {
		pub = fmt.Sprintf("%s %s\n%s", string(ms.From), Stamp(), ms.Text)
	}
	p.Range(func(k, v interface{}) bool {
		wc := Wscon{C: v.(Ws)}
		wc.Lock()
		wc.C.WriteMessage(websocket.TextMessage, []byte(pub))
		wc.Unlock()
		return true
	})
}

func Resend(m Message, p *Pool) {
	to := m.To
	from := m.From
	cfi, _ := p.Load(from)
	if !p.Contains(to) {
		m.Suspend([]User{to})
		cfi.(Ws).WriteMessage(1, []byte(fmt.Sprintf(" User %s offline now.", string(to))))
		return
	}
	ci, _ := p.Load(to)
	var letter string
	if strings.HasPrefix(m.Text, "FILE@") {
		letter = m.Text
	} else {
		letter = fmt.Sprintf("%s private from %s\n%s", Stamp(), string(from), m.Text)
	}
	cto := Wscon{C: ci.(Ws)}
	cto.Lock()
	cto.C.WriteMessage(websocket.TextMessage, []byte(letter))
	cto.Unlock()
	if to != from {
		cfr := Wscon{C: cfi.(Ws)}
		cfr.Lock()
		cfr.C.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%s message to %s delivered", Stamp(), string(to))))
		cfr.Unlock()
	}
}
