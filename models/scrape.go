package models

import (
  "net/http"
  "log"
  "regexp"
  "strings"
  "io"
  "time"
)

type Link struct {
  Title string
  ImgUrl string
  Href string
}

func GetOg(url string) (Link, error) {
  cl :=http.Client{Timeout: 5 * time.Second}
	resp, err := cl.Get(url)
	if err != nil {
		return Link{}, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Link{}, err
	}
	defer resp.Body.Close()
	
	sl := strings.Split(string(body), "<head>")
	if len(sl) < 2 {
		return Link{}, err
	}
	head := strings.Split(sl[1], "</head>")[0]
	lines := strings.Split(head, "<")
	var title, img string
	prefimg := `meta property="og:image" content=`
	prefttl := `meta property="og:title" content=`
	for _, line := range lines {
		if strings.HasPrefix(line, prefimg) {
			img = strings.TrimPrefix(line, prefimg)
		}
		if strings.HasPrefix(line, prefttl) {
			title = strings.TrimPrefix(line, prefttl)
		}
	}
	img = strings.TrimSpace(img)
	img = strings.Trim(img, "\" />")
	title = strings.Split(title, " | ")[0]
	title = strings.TrimSpace(title)
	title = strings.Trim(title, "\" />")
	
	return Link{title, img, url}, nil
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

func (m Message) LinkList() []Link {
  if m.Type != "text" {
    return []Link{}
  }
  res := []Link{}
  
  text := deselectQuotes(m.Data)
  urls := FindLinks(text)
  for _, u := range urls {
    lnk, err := GetOg(u)
    if err != nil {
      log.Printf("scrape: %s", err)
    }
    if lnk.Title != "" {
      if lnk.ImgUrl == "" {
        lnk.ImgUrl = "static/img/earth.png"
      }
      res = append(res, lnk)
    }
  }
  return res
}

func deselectQuotes(ht string) string {
	divs := strings.Split(ht, "<div")
	idx := 0
	for _, div := range divs {
	  if strings.HasPrefix(div, ` class="quoted`) {
		  break
	  }
	  idx++
	}
	tails := strings.Split(ht, "</div>")
	head := strings.Join(divs[:idx], " ")
  tail := ""
	if idx == len(divs) - 1 {
	  tail = strings.Join(tails[len(tails)-idx:], " ")
	}

	return head + tail
}
