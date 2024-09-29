package main

import (
  "log"
  "fmt"
  
  "github.com/yurajp/wx/server"
  "github.com/yurajp/wx/config"
)

func main() {
  err := config.LoadConf()
  if err != nil {
    log.Printf("Config load error: %v", err)
    fmt.Println("Enter any to quit")
    var q string
    fmt.Scanf("%s", &q)
    return 
  }
  go server.Start()
  
  <-server.Quit
}