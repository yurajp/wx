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
  "os/signal"
  "syscall"
  "context"
  "os/exec"
  "errors"
  
	"github.com/yurajp/wx/config"
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
  port string
  addr string
  regTmpl *template.Template
  chatTmpl *template.Template
  authTmpl *template.Template
  dataCh chan Message
  pool *models.Pool
  wait Wait
  Quit = make(chan struct{}, 1)
)  

var  upgrader = websocket.Upgrader{
    ReadBufferSize: 1024,
    WriteBufferSize: 1024,
  }


func NewPool() *Pool {
  return &Pool{}
}

func (ch ChatRequest) Avatar() string {
  user := User(ch.Name)
  return user.Avatar()
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
  port = config.Conf.Port
  addr = GetLocalIP() + port
  if !strings.HasPrefix(addr, "192.") {
    fmt.Println(" NO ROUTER! \n  Ctrl+C for quit!")
    return
  }
  certKeyPath := config.Conf.CertKeyPath
  certPath := config.Conf.CertPath
  
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
  authTmpl, err = template.ParseFiles("server/templates/auth.html")
  if err != nil {
    log.Printf("parse template: %v", err)
  }
  chatTmpl, _ = template.ParseFiles("server/templates/chat.html")
  fss := http.FileServer(http.Dir("server/static"))
  mux.Handle("/static/", http.StripPrefix("/static/", fss))
  fsf := http.FileServer(http.Dir("server/files"))
  mux.Handle("/files/", http.StripPrefix("/files/", fsf))
  mux.HandleFunc("/", register)
  mux.HandleFunc("/auth", authHandler)
  mux.HandleFunc("/cont", cont)
  mux.HandleFunc("/translator", translator)
  mux.HandleFunc("/saved", showSaved)
  mux.HandleFunc("/unread", unreadHandler)
  mux.HandleFunc("/clear", clearHandler)
  mux.HandleFunc("/files", filesHandler)
  mux.HandleFunc("/unsave", unsaveHandler)
  mux.HandleFunc("/newavatar", editAvatar)
  
  dataCh = make(chan Message, 1)
  defer close(dataCh)
  
  wait = Wait{}
  pool = NewPool()
  
  go models.HandleChat(dataCh, pool)
  
  go func() {
    time.Sleep(850 * time.Millisecond)
    exec.Command("xdg-open", "https://" + addr + "/").Run()
  }()
  
  fmt.Println("\n WXserver: \n", addr)
  
  go func() {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
    
    qCtx, exit := context.WithTimeout(context.Background(), 2 * time.Second)
    defer exit()
    if err := server.Shutdown(qCtx); err != nil {
      log.Printf("Server shutdown error: %v", err)
      server.Close()
    }
    fmt.Println("\n Graceful shutdown by interrupt\n")
    
    Quit <-struct{}{}
  }()
  
  err = server.ListenAndServeTLS("", "")
  if !errors.Is(err, http.ErrServerClosed) {
    log.Printf("ListenAndServe: %v", err)
    Quit <-struct{}{}
  }
}
