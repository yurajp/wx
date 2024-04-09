package server

import (
  "net"
  "net/http"
  "crypto/tls"
  "html/template"
  "log"
  "strings"
  "fmt"
  "time"
  "os"
  "os/exec"
  
	"github.com/yurajp/wx/models"
	"github.com/gorilla/websocket"
)

type (
  Pool = models.Pool
  Registry = models.Registry
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
  pool *models.Pool
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

var preTmpl = `<!DOCTYPE html>
<html>
<head>
  <meta charset='UTF-8'>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" type="text/css" href="static/style.css">
</head>
<body>
  <div id="uname">{{ . }}</div>
 <div id="ctrl">
  <button id="clu">Clear unread</button>
  <button id="cls">Clear saved</button>
 </div>
 <div id='chat' style='display:"flex"'>
    <h2>Saved messages</h2>
    <div id='output'>
`

var postTmpl = `</div></div>  <script type="text/javascript" src="static/saved.js"></script></body></html>`

func buildSavedTmpl(u string) *template.Template {
  path := fmt.Sprintf("data/%s.txt", u)
  if _, err := os.Stat(path); os.IsNotExist(err) {
    os.Create(path)
  }
  data, _ := os.ReadFile(path)
  newTmpl := preTmpl + string(data) + postTmpl
  tmpl, err := template.New("").Parse(newTmpl)
  if err != nil {
      log.Println(err)
  }
  return tmpl
}

func Start() {
  addr = GetLocalIP() + port
  if !strings.HasPrefix(addr, "192.") {
    fmt.Println(" NO ROUTER! \n  Ctrl+C for quit!")
    return
  }
	serverCert, err := tls.LoadX509KeyPair(certPath, certKeyPath)
	if err != nil {
	  log.Println(err)
	  return
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
	}
  models.LoadSha()
  models.LoadSus()
  
	mux := http.NewServeMux()
	server := http.Server{
		Addr: addr,
		Handler: mux,
		TLSConfig: tlsConfig,
	}
  regTmpl, _ = template.ParseFiles("server/reg.html")
  chatTmpl, _ = template.ParseFiles("server/chat.html")
  fss := http.FileServer(http.Dir("server/static"))
  mux.Handle("/static/", http.StripPrefix("/static/", fss))
  fsf := http.FileServer(http.Dir("server/files"))
  mux.Handle("/files/", http.StripPrefix("/files/", fsf))
  mux.HandleFunc("/", register)
  mux.HandleFunc("/translator", translator)
  mux.HandleFunc("/saved", showSaved)
  mux.HandleFunc("/unread", unreadHandler)
  mux.HandleFunc("/clear", clearHandler)
  mux.HandleFunc("/files", filesHandler)
  mux.HandleFunc("/unsave", unsaveHandler)
  
  dataCh = make(chan Message, 4)
  defer close(dataCh)
  
  wait = Wait{}
  pool = NewPool()
  
  go models.HandleChat(dataCh, pool)
  
  go func() {
    time.Sleep(850 * time.Millisecond)
    exec.Command("xdg-open", "https://" + addr + "/").Run()
  }()
  
  fmt.Println(" WX server: ", addr)
  
  log.Fatal(server.ListenAndServeTLS("", ""))
}
