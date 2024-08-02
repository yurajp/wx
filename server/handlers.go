package server

import (
  "log"
  "fmt"
  "net/http"
  "encoding/json"
  "io"
  "io/ioutil"
  "time"
  "bytes"
  "os"
  "strings"
  
	"github.com/yurajp/wx/models"
	"github.com/yurajp/wx/database"
)

type AuthCase struct {
  User User
  Case int
}


func translator(w http.ResponseWriter, r *http.Request) {
  user := wait[host(r)]
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrader:", err)
	return
  }
  c.SetReadDeadline(time.Now().Add(500 * time.Minute))
  defer c.Close()
	
  B.Attach(user, c)
  delete(wait, host(r))
  log.Printf("%s (%s), common %d", string(user), host(r), B.Actives())
  for {
	  mtype, data, err := c.ReadMessage()
	  if err != nil && err != io.EOF {
		  log.Println(err)
		  B.Detach(user)
		  c = nil
      return
    }
    if mtype == 8 {
	    B.Detach(user)
	    c = nil
	    return
	  }
	  m := models.NewMessage()
	  err = json.Unmarshal(data, &m)
	  if err != nil {
	    log.Printf("cannot get message: %v", err)
		  continue
	  }
	  if m.Type == "" {
	    continue
	  }
		dataCh <-m
	}
}

func welcome(w http.ResponseWriter, r *http.Request) {
	remote := host(r)
  reg := models.LoadRegistry()
  
	if r.Method == http.MethodGet {
	  placeholder := "Your name"
		if us, ok := reg.OnRemote(remote); ok {
			placeholder = string(us)
		}
		regTmpl.Execute(w, placeholder)
	}
	
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			log.Printf("parse form: %s", err)
			return
		}
		name := r.FormValue("name")
		user := User(name)
		if name == "" {
		  if u, ok := reg.OnRemote(remote); ok {
		    user = u
		  }
		}
		
    authCase := reg.AuthorityCase(user, remote)
    
    switch (authCase) {
    case models.InvalidInput:
      http.Error(w, "not valid name", http.StatusBadRequest)
      
      fmt.Println(" Invalid auth case")
      
    case models.KnownUknownA:
      wait[remote] = user
      log.Printf("user %v online", user)
      err = chatTmpl.Execute(w, user)
      if err != nil {
        log.Printf("chat Tmpl err: %v", err)
            Quit <-struct{}{}
      }
    default:
      ac := AuthCase{user, authCase}
      
      fmt.Printf("%+v\n", ac)
      
      err := authTmpl.Execute(w, ac)
      if err != nil {
        fmt.Println(err)
      }
    }
	}
}

func authHandler(w http.ResponseWriter, r *http.Request) {
  data, err := ioutil.ReadAll(r.Body)
  if err != nil {
    log.Printf("read post: %v", err)
    return
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
  sucmap := map[string]string{"message": "success"}
  success, err := json.Marshal(sucmap)
  if err != nil {
    log.Printf("success marshal: %v", err)
  }
  wrongmap := map[string]string{"message": "wrong pin"}
  wrongpin, _ := json.Marshal(wrongmap)
  user := User(name)
  remote := host(r)
  reg := models.LoadRegistry()
  
	w.Header().Set("Content-Type", "application/json")
  
  switch (acase) {
  case "0":
    if user.IsValid() {
      reg.AddAuth(remote, name, hpin)
    } else {
		  w.Write([]byte(`{"message":"not valid name"}`))
		  return
    }
  case "2":
    if hpin != reg.GetHPin(user) {
		  w.Write(wrongpin)
      return
    }
 
  case "3":
    if hpin != reg.GetHPin(user) {
		  w.Write(wrongpin)
      return
    }
  }
  w.Write(success)
}

func proceed(w http.ResponseWriter, r *http.Request) {
  user := r.URL.Query().Get("user")
  wait[host(r)] = User(user)
  
  chatTmpl.Execute(w, User(user))
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
  quote := r.FormValue("quote")

	var buf bytes.Buffer
	io.Copy(&buf, file)
	fpath := "server/files/" + filename
	err = os.WriteFile(fpath, buf.Bytes(), 0640)
	if err != nil {
		log.Printf("Error when write file: %s", err)
	}
	ms := models.NewMessage()
	ms.Type = "file"
	ms.From = User(from) 
	ms.To = User(to) 
	ms.Data = filename
	ms.Quote = quote
	dataCh <-ms

	log.Printf("file %s from %s to %s (%vb)", filename, from, to, size)
}

func editAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	log.Print("avatar request")
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

func deleteHandler(w http.ResponseWriter, r *http.Request) {
  sid := r.URL.Query().Get("sid")
  us := r.URL.Query().Get("user")
  isFrom, isFile, isVoice, fname := database.S.IsFromOrToAndFile(sid, us) 
  if isFrom {
    err := database.S.Delete(sid)
    if err != nil {
      log.Printf("db message delete error: %v", err)
    }
    if isFile {
      err := DeleteFile(fname)
      if err != nil {
        log.Print("file delete error: %v", err)
      }
    }
    if isVoice {
      err := DeleteVoice(fname)
      if err != nil {
        log.Print("record delete error: %v", err)
      }
    }
  }
}

func DeleteFile(f string) error {
  path := "server/files/" + f
  if _, err := os.Stat(path); err != nil {
    if os.IsNotExist(err) {
      return fmt.Errorf("may be already deleted: %s", err)
    } else {
      return err
    }
  }
  return os.Remove(path)
}

func DeleteVoice(f string) error {
  path := "server/files/records/" + f
  if _, err := os.Stat(path); err != nil {
    if os.IsNotExist(err) {
      return fmt.Errorf("may be already deleted: %s", err)
    } else {
      return err
    }
  }
  return os.Remove(path)
}


func clear(w http.ResponseWriter, r *http.Request) {
  user := r.URL.Query().Get("user")
  i, j, v, err := database.S.ClearChat(user)
  if err != nil {
    log.Printf("clear db for %s error: %v", user, err)
  }
  log.Printf("%v cleared %d messages, %d files, %d records", user, i, j, v)
}

func record(w http.ResponseWriter, r *http.Request) {
  r.ParseMultipartForm(6 * 1024 * 1024)
	file, _, err := r.FormFile("voice")
	if err != nil {
		log.Printf("cannot get record from form: %s", err)
		return
	}
	defer file.Close()
	
	var buf bytes.Buffer
	io.Copy(&buf, file)
	if buf.Len() == 0 {
	  log.Printf("empty file")
	  return
	}
	vms := models.NewMessage()
	vms.Type = "voice"
	vms.From = User(r.FormValue("from"))
	vms.To = User(r.FormValue("to"))
	vms.Quote = r.FormValue("quote")
	filename := vms.Sid + ".wav"
	vms.Data = filename
	fpath := "server/files/records/" + filename
	err = os.WriteFile(fpath, buf.Bytes(), 0640)
	if err != nil {
		log.Printf("Error when write record: %s", err)
	}
	log.Print("voice message saved")
	
  dataCh <-vms
}

func filter(w http.ResponseWriter, r *http.Request) {
  user := r.URL.Query().Get("user")
  other := r.URL.Query().Get("other")
  showMap, ok := models.CM[User(user)]
  if !ok {
    showMap = models.SidMap{}
  }
  showList := showMap[User(other)]

  hl := struct {
    List []string `json:"list"`
  }{showList}
  list, err := json.Marshal(hl)
  if err != nil {
    log.Printf("handle filter  error: %v", err)
    return
  }
  w.Write(list)
}

func avatar(w http.ResponseWriter, r *http.Request) {
  name := r.URL.Query().Get("comp")
  av := User(name).Avatar()
  
  w.Write([]byte(av))
}
