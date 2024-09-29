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
	"github.com/yurajp/wx/database"
	"github.com/gorilla/websocket"
)

type (
  Registry = models.Registry
  Message = models.Message
  User = models.User
  Wait map[string]User
)


var (
  port string
  addr string
  B *models.Board
  regTmpl *template.Template
  chatTmpl *template.Template
  authTmpl *template.Template
  dataCh chan *Message
  wait Wait
  Quit = make(chan struct{}, 1)
)  

var  upgrader = websocket.Upgrader{
    ReadBufferSize: 1024,
    WriteBufferSize: 1024,
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

func Start() {
  port = config.Conf.Port
  addr = GetLocalIP() + port
  if !strings.HasPrefix(addr, "192.") {
    fmt.Println(" NO ROUTER CONNECTED!\n  Ctrl+C for quit!")
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
  
	mux := http.NewServeMux()
	server := http.Server{
		Addr: addr,
		Handler: mux,
		TLSConfig: tlsConfig,
	}
  regTmpl, err = template.ParseFiles("server/templates/welcome.html")
  if err != nil {
    log.Printf("parse welcomeTmpl: %v", err)
  }
  authTmpl, err = template.ParseFiles("server/templates/auth.html")
  if err != nil {
    log.Printf("parse authtmpl: %v", err)
  }
  chatTmpl, err = template.ParseFiles("server/templates/chat.html")
  if err != nil {
    log.Printf("parse chatTmpl: %v", err)
  }
  fss := http.FileServer(http.Dir("server/static"))
  mux.Handle("/static/", http.StripPrefix("/static/", fss))
  fsf := http.FileServer(http.Dir("server/files"))
  mux.Handle("/files/", http.StripPrefix("/files/", fsf))
  mux.HandleFunc("/", welcome)
  mux.HandleFunc("/auth", authHandler)
  mux.HandleFunc("/cont", proceed)
  mux.HandleFunc("/translator", translator)
  mux.HandleFunc("/delete", deleteHandler)
  mux.HandleFunc("/files", filesHandler)
  mux.HandleFunc("/newavatar", editAvatar)
  mux.HandleFunc("/clear", clear)
  mux.HandleFunc("/record", record)
  mux.HandleFunc("/filter", filter)
  mux.HandleFunc("/avatar", avatar)
  
  dataCh = make(chan *Message, 5)
  
  wait = Wait{}
  rg := models.LoadRegistry()
  B = models.NewBoard(rg)
  models.CM = *models.NewChatMap(B)
  models.Addr = addr
  
  go B.ListenChat(dataCh)
 
  go func() {
    time.Sleep(850 * time.Millisecond)
    exec.Command("xdg-open", "https://" + addr + "/").Run()
  }()
  
  go func() {
    err := database.SetStorage()
    if err != nil {
      log.Printf("Database error: %v", err)
      Quit <-struct{}{}
    }
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
    database.S.Db.Close()
    fmt.Println("\n Shutdown by interrupt\n")
    
    Quit <-struct{}{}
  }()
  
  err = server.ListenAndServeTLS("", "")
  if !errors.Is(err, http.ErrServerClosed) {
    log.Printf("ListenAndServe: %v", err)
    Quit <-struct{}{}
  }
}
