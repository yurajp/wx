package models

import (
  	"crypto/sha1"
  	"encoding/base64"
  	"encoding/json"
  	"fmt"
  	"log"
  	"os"
//	"strings"
  	"time"
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
		//  ls, _ :=  suspence[u]
		suspence[u] = append(suspence[u], hash)
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
	js, err := json.Marshal(suspence)
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
}

func (sus Suspence) Release(u User, ch chan Message) {
	if ls, ok := sus[u]; ok {
		for _, h := range ls {
			dm := shamess[h]
			txt := fmt.Sprintf(" %s\n%s", dm.Date, dm.Text)
			ms := Message{From: dm.From, To: u, Text: txt}
			ch <-ms
		}
		sus[u] = []string{}
		if len(ls) > 0 {
			go suspence.Update()
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
	n := 0
	for k, ls := range suspence {
		if k == u {
			continue
		}
		lnls := len(ls)
    for _, s := range shas {
		  for i := 0; i < lnls; i++ {
				if s == ls[i] {
				  lnls--
					ls[i] = ls[lnls]
					n++
				}
			}
			ls = ls[:lnls]
		}
		suspence[k] = ls
	}

	log.Printf("%d unreads of %v cleared", n, u)
	go shamess.Update()
	go suspence.Update()
}
