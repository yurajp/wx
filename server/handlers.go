package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yurajp/wx/models"
)

type AuthCase struct {
  Name User
  Case string
}

func translator(w http.ResponseWriter, r *http.Request) {
	user := wait[host(r)]
	if pool.Contains(user) {
		log.Printf("User %s already in chat", string(user))
		return
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrader:", err)
		return
	}
	c.SetReadDeadline(time.Now().Add(500 * time.Minute))
	defer c.Close()

	pool.Register(c, user)
	delete(wait, host(r))
	log.Printf("New member: %s (%s), common %d", string(user), host(r), pool.Size())

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

func isKnownAndFree(remote string) (User, bool) {
  u, ok := models.GetKnown(remote)
  if !ok {
    return User(""), false
  }
  if wait.Already(remote) {
    return u, false
  }
  if pool.Contains(u) {
    return u, false
  }
  return u, true
}

func isKnownNewOrOther(remote, name string) (User, int) {
  u, ad := models.GetKnown(remote)
  reg := name != "" && models.IsRegistered(User(name))
  other := (u != User(name))
  switch {
		case ad && (name == "" || !other):
      return u, 0
		case ad && reg && other:
		  return User(name), 1
		case !ad && reg:
		  return User(name), 2
		case !ad && !reg:
		  return User(name), 3
		}
		return User(""), -1
}


func register(w http.ResponseWriter, r *http.Request) {
	remote := host(r)

	if r.Method == http.MethodGet {
	  placeholder := "Your name"
		if us, ok := isKnownAndFree(remote); ok {
			placeholder = string(us)
		}
		regTmpl.Execute(w, placeholder)
	}
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			log.Printf("parse form: %s", err)
		}
		name := r.FormValue("name")
    user, nCase := isKnownNewOrOther(remote, name)
    
    switch (nCase) {
    case -1:
      http.Error(w, "not valid name", http.StatusBadRequest)
    case 0:
      if models.GetHPin(user) == "" {
        ac := AuthCase{user, "0"}
        authTmpl.Execute(w, ac)
      } else {
			  wait[remote] = user
			  chr := ChatRequest{string(user), addr}
			  chatTmpl.Execute(w, chr)
      }
		case 1:
		  ac := AuthCase{User(name), "1"}
		  authTmpl.Execute(w, ac)
		case 2:
		  ac := AuthCase{User(name), "2"}
		  authTmpl.Execute(w, ac)
		case 3:
		  if User(name).IsValid() {
		    ac := AuthCase{User(name), "3"}
		    authTmpl.Execute(w, ac)
		  } else {
		    http.Error(w, "not valid name", http.StatusBadRequest)
		  }
    }
	}
}

func authHandler(w http.ResponseWriter, r *http.Request) {
  if r.Method != http.MethodPost {
    w.WriteHeader(405)
    http.Error(w, "NotAllowed", 405)
    return
  }
  data, err := ioutil.ReadAll(r.Body)
  if err != nil {
    log.Printf("read post: %v", err)
  }
  defer r.Body.Close()
  
  type Post struct {
    Pin string `json:"pin"`
    Name string `json:"name"`
    Case string `json:"cas"`
  }
  
  var post Post
  err = json.Unmarshal(data, &post)
  if err != nil {
    log.Printf("unmarshal data: %v", err)
  }
  
  hpin := models.HashPin(post.Pin)
  name := post.Name
  acase := post.Case
  
  success := []byte(`{"message":"success"}`)
  wrongpin := []byte(`{"message":"wrong pin"}`)
  
  switch (acase) {
  case "0":
    go models.UpdateAuth(name, hpin)
  case "1":
    if hpin != models.GetHPin(User(name)) {
		  w.WriteHeader(http.StatusUnauthorized)
		  w.Write(wrongpin)
      return
    }
 
  case "2":
    spin := models.GetHPin(User(name))
    if spin != hpin {
		  w.WriteHeader(http.StatusUnauthorized)
		  w.Write(wrongpin)
      return
    }
    go models.Reg.NewKnown(host(r), User(name))
  case "3":
    if !User(name).IsValid() {
		  w.WriteHeader(http.StatusUnauthorized)
		  w.Write([]byte(`{"message":"not valid name"}`))
		  return
    } 
    go models.UpdateAuth(name, hpin)
    go models.Reg.NewKnown(host(r), User(name))
  }
  
  log.Printf("auth case %s handled", acase)
  
  w.Header().Set("Content-Type", "application/json")
  w.Write(success)
  
//   chr := ChatRequest{name, addr}
//   time.Sleep(2 * time.Second)
// 	chatTmpl.Execute(w, chr)

}

func cont(w http.ResponseWriter, r *http.Request) {
  us := r.URL.Query().Get("user")
  wait[host(r)] = User(us)
  chr := ChatRequest{us, addr}
  
  chatTmpl.Execute(w, chr)
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
