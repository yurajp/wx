package models

import (
  "net/http"
  "log"
  "regexp"
  "strings"
  "errors"
  "time"
  
	"github.com/PuerkitoBio/goquery"
)

type Link struct {
  Title string
  Href string
}

func TitleScrape(url string) (string, error) {
  cl :=http.Client{Timeout: 15 * time.Second}
	res, err := cl.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", errors.New("URL is not responding")
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", err
	}
	var title string
	doc.Find("head").Each(func(_ int, s *goquery.Selection) {
		title = s.Find("title").Text()
	})
	return title, nil
}

func FindLinks(text string) []string {
  text = strings.Replace(text, ">", " ", -1)
  text = strings.Replace(text, "<", " ", -1)
  url := regexp.MustCompile(`http(s)?://*`)
  res := []string{}
  for _, w := range strings.Fields(text) {
    if url.MatchString(w) {
      res = append(res, w)
    }
  }
  return res
}

func MakeTitleLink(url string) Link {
  t, err := TitleScrape(url)
  if err != nil {
    log.Printf("scrape title: %s", err)
    return Link{}
  }
  if t == "" {
    return Link{}
  }
  return Link{t, url}
}

func (m Message) LinkList() []Link {
  if m.Type != "text" {
    return []Link{}
  }
  res := []Link{}
  text := m.Data
  urls := FindLinks(text)
  for _, u := range urls {
    lnk := MakeTitleLink(u)
    if lnk.Title != "" {
      res = append(res, lnk)
    }
  }
  return res
}

