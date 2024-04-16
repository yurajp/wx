package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/yurajp/wx/models"
)

func translator(w http.ResponseWriter, r *http.Request) {
	user := wait[host(r)]
	if pool.Contains(user) {
		log.Printf("User %s already in pool", string(user))
		return
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrader:", err)
		return
	}
	defer c.Close()

	pool.Register(c, user)
	delete(wait, host(r))
	log.Printf("New user: %s (%s) common %d", string(user), host(r), pool.Size())

	for {
		mtype, message, err := c.ReadMessage()
		if err != nil && err != io.EOF {
			log.Println(err)

			pool.Unregister(user)
			models.Publicate(pool)
			break
		}
		switch mtype {
		case 8:
			pool.Unregister(user)
			models.Publicate(pool)
			log.Println(message)
			return
		case websocket.TextMessage:
			if string(message) == "CLOSED" {
				pool.Unregister(user)
				return
			}
			var ms Message
			err = json.Unmarshal(message, &ms)
			if err != nil {
				fmt.Println(" no message detected")
				continue
			}
			dataCh <- ms
			log.Println("message from ", ms.From)
		case websocket.BinaryMessage:
			var ms Message
			err = json.Unmarshal(message, &ms)
			if err != nil {
				log.Printf("json: %s", err)
				continue
			}
			dataCh <- ms
		default:
		}
	}
}

func register(w http.ResponseWriter, r *http.Request) {
	remote := host(r)
	placeholder := "Your name"
	if r.Method == http.MethodGet {
		if u, ok := models.GetKnown(remote); ok && !wait.Already(remote) && !pool.Contains(u) {
			placeholder = string(u)
		}
		regTmpl.Execute(w, placeholder)
	}
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			log.Printf("parse form: %s", err)
		}
		name := r.FormValue("name")

		u, ok := models.GetKnown(remote)
		if ok && name == "" {
			wait[remote] = u
			fmt.Printf("~ User %s is waiting\n", string(u))
			chr := ChatRequest{string(u), addr}
			chatTmpl.Execute(w, chr)
		} else {
			user := User(name)
			if user.IsValid() {
				wait[remote] = user
				fmt.Printf("User %s is waiting\n", string(user))
				models.Reg.NewKnown(remote, user)

				chr := ChatRequest{string(user), addr}
				chatTmpl.Execute(w, chr)
			}
		}
	}
}

func showSaved(w http.ResponseWriter, r *http.Request) {
	u := r.URL.Query().Get("user")
	tmpl := buildSavedTmpl(u)

	tmpl.Execute(w, User(u))
}

func clearHandler(w http.ResponseWriter, r *http.Request) {
	u := r.URL.Query().Get("user")
	path := "data/" + u + ".txt"
	err := os.Truncate(path, 0)
	if err != nil {
		log.Printf("clear: %s", err)
	}
}

func filesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		return
	}
	r.ParseMultipartForm(1024 * 1024 * 8)
	file, header, err := r.FormFile("files")
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()
	filename := header.Filename
	size := header.Size
	from := r.FormValue("from")
	to := r.FormValue("to")

	var buf bytes.Buffer
	io.Copy(&buf, file)
	fpath := "server/files/" + filename
	err = os.WriteFile(fpath, buf.Bytes(), 0640)
	if err != nil {
		log.Printf("Error when write file: %s", err)
	}
	text := "FILE@" + filename
	ms := Message{From: User(from), To: User(to), Text: text}
	dataCh <- ms

	log.Printf("~ File %s from %s to %s (%vb)", filename, from, to, size)
}

func unreadHandler(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	models.ClearUnreaded(User(user))
}

func unsaveHandler(w http.ResponseWriter, r *http.Request) {
	mid := r.URL.Query().Get("mid")
	u := r.URL.Query().Get("user")
	path := "data/" + u + ".txt"
	bs, err := os.ReadFile(path)
	if err != nil {
		log.Print(err)
		return
	}
	del := "<!--"
	blx := strings.Split(string(bs), del)
	res := []string{}
	for _, bl := range blx {
		if !strings.HasPrefix(bl, mid) && len(bl) > 0 {
			res = append(res, del+bl)
		}
	}
	htm := strings.Join(res, "")
	npath := "data/" + u + ".txt"
	os.WriteFile(npath, []byte(htm), 0640)
}

func editAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	log.Print("~ avatar request")
	r.ParseMultipartForm(1024 * 1024 * 16)
	file, header, err := r.FormFile("newavatar")
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()
	filename := header.Filename
	ext := strings.Split(filename, ".")[1]
	if ext != "jpg" && ext != "png" && ext != "jpeg" {
		log.Printf(".%s is not supported image", ext)
		return
	}
	from := r.FormValue("from")
	newname := "server/files/avatars/" + from + "." + ext
	var buf bytes.Buffer
	io.Copy(&buf, file)
	err = os.WriteFile(newname, buf.Bytes(), 0640)
	if err != nil {
		log.Printf("Error when update avatar: %s", err)
		return
	}

	log.Printf("%s updates avatar", from)
}
