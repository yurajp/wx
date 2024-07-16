package database

import (
  "database/sql"
  "log"
  "encoding/json"
  "os"
  "fmt"
  
  _ "github.com/mattn/go-sqlite3"
)

var (
  S Storage
)

type Storage struct {
  Db *sql.DB
}

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

type DbMessage struct {
  Sid string
  Blob []byte
}

func (dt DateTime) String() string {
  return fmt.Sprintf("%s %s", dt.Date, dt.Time)
}

func (s Storage) Store(m Message) {
  ins := "insert into blobs(sid, message) values(?, ?)"
  sid := m.Sid
  js, _ := json.Marshal(m)
  _, err := s.Db.Exec(ins, sid, js)
  if err != nil {
    log.Printf("adding to Store error: %s", err)
    return
  }
}

func (s Storage) LoadMessagesFor(u User) []*Message {
  q := "select * from blobs"
  ms := []*Message{}
  rows, err := s.Db.Query(q)
	if err != nil {
		log.Printf("storage loading error: %s", err)
		return ms
	}
	defer rows.Close()
	
	for rows.Next() {
		var dr DbMessage
		rows.Scan(&dr.Sid, &dr.Blob)
		var m Message
		json.Unmarshal(dr.Blob, &m)
		if m.From == u || m.To == u || m.To == User("All") {
		  ms = append(ms, &m)
		}
	}
	return ms
}

// returns number of cleared messages and files.
func (s Storage) ClearChat(u string) (int, int, int, error) {
  q := "select * from blobs"
  sids := []string{}
  files := []string{}
  voices := []string{}
  rows, err := s.Db.Query(q)
	if err != nil {
		return 0, 0, 0, err
	}
	defer rows.Close()
	
	user := User(u)
	
	for rows.Next() {
		var dr DbMessage
		rows.Scan(&dr.Sid, &dr.Blob)
		var m Message
		err = json.Unmarshal(dr.Blob, &m)
		if err != nil {
		  return 0, 0, 0, err
		}
		if m.From == user || m.To == user {
		  sids = append(sids, m.Sid)
		  if m.Type == "file" {
		    files = append(files, m.Data)
		  }
		  if m.Type == "voice" {
		    voices = append(voices, m.Data)
		  }
		}
	}
	cntm, cntf, cntv := 0, 0, 0
	for _, sid := range sids {
	  err = s.Delete(sid)
	  if err != nil {
	    return cntm, cntf, cntv, err
	  }
	  cntm++
	}
	for _, f := range files {
	  err = s.RemoveFile(f)
	  if err != nil {
	    return cntm, cntf, cntv, err
	  }
	  cntf++
	}
	for _, v := range voices {
	  err = s.RemoveVoice(v)
	  if err != nil {
	    return cntm, cntf, cntv, err
	  }
	  cntv++
	}
	return cntm, cntf, cntv, nil
}

func SetStorage() error {
  db, err := sql.Open("sqlite3", "data/store.db")
	if err != nil {
		return err
	}
	state, err := db.Prepare(`create table if not exists blobs(sid varchar, message blob)`)
	if err != nil {
		return err
	}
	state.Exec()
	S = Storage{db}
	return nil
}

// returns isFrom, isFile, filename
func (s Storage) IsFromOrToAndFile(sid, u string) (bool, bool, bool, string) {
  m, err := s.GetMessage(sid)
  if err != nil {
    log.Print(err)
    return false, false, false, ""
  }
  isFrom := (m.From == User(u)) || (m.To == User(u))
  isFile := m.Type == "file"
  isVoice := m.Type == "voice"
  fname := ""
  if isFile || isVoice {
    fname = m.Data
  }
  return isFrom, isFile, isVoice, fname
}


func (s Storage) Delete(sid string) error {
  state, err := s.Db.Prepare("delete from blobs where sid = ?")
  if err != nil {
    return err
  }
  state.Exec(sid)
  return nil
}

func (s Storage) RemoveFile(f string) error {
  path := "server/files/" + f
  if _, err := os.Stat(path); err != nil {
    if os.IsNotExist(err) {
      return fmt.Errorf("may be already deleted: %v", err)
    } else {
      return err
    }
  }
  return os.Remove(path)
}

func (s Storage) RemoveVoice(v string) error {
  path := "server/files/records/" + v
  if _, err := os.Stat(path); err != nil {
    if os.IsNotExist(err) || v == "" {
      return fmt.Errorf("may be already deleted: %v", err)
    } else {
      return err
    }
  }
  return os.Remove(path)
}

func (s Storage) GetMessage(sid string) (Message, error) {
  query := "select sid, message from blobs where sid = ?"
  row := s.Db.QueryRow(query, sid)
  var dr DbMessage
  err := row.Scan(&dr.Sid, &dr.Blob)
  if err != nil {
    return Message{}, err
  }
  var m Message
  err = json.Unmarshal(dr.Blob, &m)
  if err != nil {
    return Message{}, err
  }
  return m, nil
}