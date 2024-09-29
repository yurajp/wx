package config

import (
  "fmt"
  "os"
  "errors"
  "net"
  
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

func getLocalIP() string {
    conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        fmt.Println(err)
        return ""
    }
    defer conn.Close()
    localAddr := conn.LocalAddr().(*net.UDPAddr)
    return localAddr.IP.String()
}



func LoadConf() error {
  var err error
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
  locIp := getLocalIP()
  if locIp == "" {
    return errors.New("cannot get local IP address")
  }
  cfg.CertPath = makeCertPath(cfg.CertPath, locIp)
  cfg.CertKeyPath = makeCertKeyPath(cfg.CertKeyPath, locIp)
  
  err = checkCerts(cfg.CertPath, cfg.CertKeyPath)
  if err != nil {
    return err
  }
  Conf = &cfg
  
  return nil
}

func makeCertPath(certPath, addr string) string {
  return certPath + addr + ".pem"
}

func makeCertKeyPath(certPath, addr string) string {
  return certPath + addr + "-key.pem"
}

func checkCerts(crt, crtkey string) error {
  if _, err := os.Stat(crt); err != nil {
    if os.IsNotExist(err) {
      return errors.New("certificate not found for this IP")
    }
  }
  if _, err := os.Stat(crtkey); err != nil {
    if os.IsNotExist(err) {
      return errors.New("certificate key not found for this IP")
    }
  }
  return nil
}
