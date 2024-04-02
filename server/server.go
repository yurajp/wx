package server

import (
  "net"
  "net/http"
  "crypto/tls"
  "html/template"
  "log"
  "bytes"
  "io"
  "strings"
  "encoding/json"
  "fmt"
  "time"
  "os"
  "os/exec"
  
	"github.com/yurajp/wx/models"
	"github.com/gorilla/websocket"
)

type (
  Pool = models.Pool
  Message = models.Message
  User = models.User
  Wait map[string]User
)

type ChatRequest struct {
  Name string
  Addr string
}

var (
  port = ":6555"
  addr string
  certKeyPath = "/data/data/com.termux/files/home/mkcerts/localhost+2-key.pem"
  certPath = "/data/data/com.termux/files/home/mkcerts/localhost+2.pem"
  regTmpl *template.Template
  chatTmpl *template.Template
  dataCh chan Message
  pool = NewPool()
  wait Wait
)  

var  upgrader = websocket.Upgrader{
    ReadBufferSize: 1024,
    WriteBufferSize: 1024,
  }


func NewPool() *Pool {
  return &Pool{}
}

func (w Wait) Already(r string) bool {
  _, exists := w[r]
  return exists
}

func host(r *http.Request) string {
  return strings.Split(r.RemoteAddr, ":")[0]
}

func GetLocalIP() string {
    conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        fmt.Println(err)
        return ""
    }
    defer conn.Close()
    localAddr := conn.LocalAddr().(*net.UDPAddr)
    return localAddr.IP.String()
}

func translator(w http.ResponseWriter, r *http.Request) {
  user := wait[host(r)]
  if pool.Contains(string(user)) {
    log.Printf("User %s already in pool", string(user))
    return
  }
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrader:", err)
		return
	}
	defer c.Close()
	
	pool.Add(c, user)
	delete(wait, host(r))
  log.Printf("New user: %s (%s) common %d", string(user), host(r), pool.Size())
  pool.Publicate()
  for {
	mtype, message, err := c.ReadMessage()
	if err != nil && err != io.EOF {
	    log.Println(err)
	    pool.Delete(user)
	    break
	}
	switch (mtype) {
	  case 8:
	    pool.Unregister(user)
	    pool.Publicate()
	    log.Println(message)
	    return
	  case websocket.TextMessage:
	    if string(message) =="CLOSED" {
	      pool.Unregister(user)
	      return
	    }
	    var ms Message
	    err = json.Unmarshal(message, &ms)
	    if err != nil {
	      fmt.Println(" no message detected")
	      continue
	    }
	    dataCh <-ms
	    log.Println("message from ", ms.From)
	  case websocket.BinaryMessage:
	    var ms Message
	    err = json.Unmarshal(message, &ms)
	    if err != nil {
	      log.Printf("json: %w", err)
	      continue
	    }
	    dataCh <-ms
	  default:
	  } 
	}
}

func register(w http.ResponseWriter, r *http.Request) {
  remote := host(r)
  placeholder := "Your name"
  if r.Method == http.MethodGet {
    if u, ok := pool.Known(r); ok && !wait.Already(remote) && !pool.Contains(string(u)) {
      placeholder = string(u)
    }
    regTmpl.Execute(w, placeholder)
  }
  if r.Method == http.MethodPost {
    err := r.ParseForm()
    if err != nil {
      log.Printf("parse form: %w", err)
    }
    name := r.FormValue("name")
    if u, ok := pool.Known(r); ok && name == "" {
      wait[remote] = u
      fmt.Printf("~ User %s is waiting\n", string(u))
      chr := ChatRequest{string(u), addr}
        chatTmpl.Execute(w, chr)
    } else {
      user := User(name)
      if user.IsValid() {
        wait[remote] = user
      fmt.Printf("User %s is waiting\n", string(user))
      chr := ChatRequest{string(user), addr}
        chatTmpl.Execute(w, chr)
      }
    }
  }
}

var preTmpl = `<!DOCTYPE html>
<html>
<head>
  <meta charset='UTF-8'>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" type="text/css" href="static/style.css">
  <script type="text/javascript" src="static/saved.js"></script>
</head>
<body>
 <div id="ctrl">
  <button id="username">{{ .Name }}</button>
  <button id="open">Clear</button>
 </div>
 <div id='addr'>{{ .Addr }}</div>
  <div id='chat' style='display:"flex"'>
    <div id='output'>
`

var postTmpl = "</div></div></body></html>"

func buildSavedTmpl(u string) (*template.Template, error) {
  path := fmt.Sprintf("data/%s.txt", u)
  if _, err := os.Stat(path); err != nil {
    return &(template.Template{}), err
  } else {
    data, err := os.ReadFile(path)
    if err != nil {
      return &(template.Template{}), err
    }
    tmpl, err := template.New("").Parse(preTmpl + string(data) + postTmpl)
    if err != nil {
      log.Println(err)
    }
    return tmpl, nil
  }
}

func showSaved(w http.ResponseWriter, r *http.Request) {
  u := r.URL.Query().Get("user")
  log.Printf("user %s queries saved messages", u)
  tmpl, err := buildSavedTmpl(u)
  if err != nil {
    log.Printf("Saved of %s: %w", u, err)
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }
  chr := ChatRequest{u, addr}
  tmpl.Execute(w, chr)
}

func clearHandler(w http.ResponseWriter, r *http.Request) {
  u := r.URL.Query().Get("user")
  path := "data/" + u + ".txt"
  err := os.Truncate(path, 0)
  if err != nil {
    log.Printf("clear: %w", err)
  }
}

func filesHandler(w http.ResponseWriter, r *http.Request) {
  if r.Method == http.MethodGet {
    return
  }
  r.ParseMultipartForm(1024 * 1024 * 6)
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
  ms := Message{User(from), User(to), text}
  dataCh <-ms
  
  log.Printf("~ File %s from %s to %s (%vb)", filename, from, to, size) 
}

func Start() {
  addr = GetLocalIP() + port
  if !strings.HasPrefix(addr, "192.") {
    fmt.Println("  NO ROUTER! \n  Ctrl+C for quit!")
    return
  }
	serverCert, err := tls.LoadX509KeyPair(certPath, certKeyPath)
	if err != nil {
		log.Fatalf(" Error loading cert and key: %w", err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
	}
	mux := http.NewServeMux()
	server := http.Server{
		Addr: addr,
		Handler: mux,
		TLSConfig: tlsConfig,
	}
  
  dataCh = make(chan Message, 4)
  defer close(dataCh)
  wait = Wait{}
  regTmpl, _ = template.ParseFiles("server/reg.html")
  chatTmpl, _ = template.ParseFiles("server/chat.html")
  fss := http.FileServer(http.Dir("server/static"))
  mux.Handle("/static/", http.StripPrefix("/static/", fss))
  fsf := http.FileServer(http.Dir("server/files"))
  mux.Handle("/files/", http.StripPrefix("/files/", fsf))
  mux.HandleFunc("/", register)
  mux.HandleFunc("/translator", translator)
  mux.HandleFunc("/saved", showSaved)
  mux.HandleFunc("/clear", clearHandler)
  mux.HandleFunc("/files", filesHandler)
  go pool.HandleData(dataCh)
  
  fmt.Println(" WSSX server: ", addr)

  go func() {
    time.Sleep(750 * time.Millisecond)
    exec.Command("xdg-open", "https://" + addr + "/").Run()
  }()
  
  log.Fatal(server.ListenAndServeTLS("", ""))
}
