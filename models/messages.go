package models

import (
  "path/filepath"
  "time"
  "crypto/sha1"
  "encoding/base64"
  "encoding/json"
  "html/template"
  "bytes"
  "fmt"
  "strings"
  "log"
  
  "github.com/yurajp/wx/database"
)

var (
  Addr string
  templ *template.Template
)

type User string

type Message struct {
  Sid string `json:"sid"`
  Time DateTime `json:"time"`
  From User `json:"from"`
  To User `json:"to"`
  Type string `json:"type"`
  Data string `json:"data"`
  Quote string `json:"quote"`
}

type DateTime struct {
  Date string `json:"date"`
  Time string `json:"time"`
}

type HtmlMessage struct {
  Kind string `json:"kind"`
  Date string `json:"date"`
  Content string `json:"content"`
}

func MakeId() string {
	nt := fmt.Sprintf("%d", time.Now().UnixNano())
	hash := sha1.New()
	hash.Write([]byte(nt))
	sha := base64.URLEncoding.EncodeToString(hash.Sum(nil))
	return string(sha[:12])
}

func DateNow() string {
  return time.Now().Format("02 Jan")
}

func MakeTime() DateTime {
  tn := time.Now()
  d := tn.Format("02 Jan")
  t := tn.Format("15:04")
  return DateTime{d, t}
}

func NewMessage() *Message {
  return &Message{Sid: MakeId(), Time: MakeTime()}
}

func (m *Message) IsPrivate() bool {
  return m.To != User("All")
}

func (m *Message) FileAddr() string {
  return "https://" + Addr + "/files/" + m.Data
}

func (m *Message) FileImage() string {
  ext := filepath.Ext(m.Data)
  t := "other"
  switch ext {
    case ".png", ".jpg", ".jpeg":
      t = "image"
    case ".mp3", ".wav", ".ogg", ".m4a":
      t = "audio"
    case ".mp4", ".webm", ".avi", ".mkv", ".mpeg":
      t = "video"
    case ".pdf", ".txt", ".doc", ".docx":
      t = "text"
    case ".zip", ".rar", ".gz", ".tar":
      t = "archive"
    case ".go", ".py", ".css", ".html", ".js", ".sh", ".kt":
      t = "code"
  }
  
  return "static/types/" + t + ".png"
}

func (m *Message) TemplHTML() ([]byte, error) {
  var buf bytes.Buffer
  err := templ.ExecuteTemplate(&buf, m.Type, m)
  if err !=nil {
    return []byte{}, err
  }
  return buf.Bytes(), nil
}

func (m *Message) ToHtmlMessage() ([]byte, error) {
  data, err := m.TemplHTML()
  if err != nil {
    return []byte{}, err
  }
  hm := HtmlMessage{"html", m.Time.Date, string(data)}
  jm, _ := json.Marshal(hm)
  
  return jm, nil
}

func (user User) IsValid() bool {
  return len([]rune(string(user))) > 3
}

func (m Message) DeleteAddr() string {
  return fmt.Sprintf("https://%s/delete?sid=%s", Addr, m.Sid)
}

func (m Message) VoiceAddr() string {
  return fmt.Sprintf("https://%s/files/records/%s.wav", Addr, m.Sid )
}

func Limited(data string) string {
  words := strings.Fields(data)
  lim := []string{}
  for i := 0; i < 9; i++ {
    lim = append(lim, words[i])
    if i == len(words) - 1 {
      break
    }
  }
  sfx := ""
  if len(words) > 9 {
    sfx = "..."
  }
  return strings.Join(lim, " ") + sfx
}

func (m Message) HasQuote() bool {
  return m.Quote != ""
}

func (m Message) Quoted() []string {
  dbm, err := database.S.GetMessage(m.Quote)
  if err != nil {
    log.Printf("cannot get quoted message: %v", err)
    return []string{}
  }
  return []string{string(dbm.From), Limited(dbm.Data)}
}