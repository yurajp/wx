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

func (ch ChatRequest) Avatar() string {
  user := ch.Name
  path := "files/avatars/" + user
  res := "files/avatars/user.png"
  if _, err := os.Stat("server/" + path + ".jpg"); !os.IsNotExist(err) {
    res = path + ".jpg"
  } else if _, err := os.Stat("server" + path + ".png"); !os.IsNotExist(err) {
    res = path + ".png"
  }
  return res
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

var postTmpl = `</div></div>  <script type="text/javascript" src="static/js/saved.js"></script></body></html>`

func buildSavedTmpl(u string) *template.Template {
  path := fmt.Sprintf("data/%s.txt", u)
  if _, err := os.Stat(path); os.IsNotExist(err) {
    os.Create(path)
  }
  data, err := os.ReadFile(path)
  if err != nil {
    log.Println(err)
    return &template.Template{}
  }
  preTmp, err := os.ReadFile("server/templates/preTmpl.htm")
  if err != nil {
    log.Println(err)
    return &template.Template{}
  }
  newTmpl := string(preTmp) + string(data) + postTmpl
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
  regTmpl, _ = template.ParseFiles("server/templates/reg.html")
  chatTmpl, _ = template.ParseFiles("server/templates/chat.html")
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
  
  fmt.Println("\n WXserver: ", addr)
  
  log.Fatal(server.ListenAndServeTLS("", ""))
}
