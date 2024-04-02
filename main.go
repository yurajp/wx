package main

import (
  "github.com/yurajp/wx/server"
)

func main() {
  go server.Start()
  select{}
}