package config

import (
  "fmt"
  "os"
  "errors"
  
  "github.com/yurajp/confy"
)

type WsConf struct {
  Port string
  CertPath string
  CertKeyPath string
}

var defaults = WsConf{
  Port: ":6555",
  CertPath: "undefined",
  CertKeyPath: "undefined",
}

var Conf *WsConf

func initConf() (bool, error) {
  if _, err := os.Stat("config/conf.ini"); os.IsNotExist(err) {
    err = confy.WriteConfy(defaults)
    if err != nil {
      return false, fmt.Errorf("error when write default config: %s", err)
    }
    return false, nil
  }
  return true, nil
}

func LoadConf() error {
  warn := "You should edit the configuration file 'config/conf.ini' to define path to your certificates"
  undefinedErr := errors.New("path to certificate is not defined yet")
  ok, err := initConf()
  if err != nil {
    return fmt.Errorf("init conf: %s", err)
  }
  if !ok {
    fmt.Println(warn)
    return undefinedErr
  }
  cfg := WsConf{}
  icfg, err := confy.LoadConfy(cfg)
  if err != nil {
    return fmt.Errorf("error when load conf.ini: %s", err)
  }
  cfg, ok = icfg.(WsConf) 
  if !ok {
    return errors.New("corrupted config")
  }
  if cfg.CertPath == "undefined" || cfg.CertKeyPath == "undefined" {
    fmt.Println(warn)
    return undefinedErr
  }
  Conf = &cfg
  
  return nil
}
