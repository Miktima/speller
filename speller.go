package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"golang.org/x/net/html"
)

type SpellOptions struct {
	Article string
	Lang    string
	Options int
	Format  string
}

type SpellError struct {
	Code int
	Pos  int
	Row  int
	Col  int
	Len  int
	Word string
	S    []string
}

func getHtmlPage(url, userAgent string) ([]byte, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Cannot create new request  %s, error: %v\n", url, err)
		return nil, err
	}

	req.Header.Set("User-Agent", "RIA/autotest")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error with GET request: %v\n", err)
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error ReadAll")
		return nil, err
	}
	return body, nil
}

func getArticle(body []byte, tag, keyattr, value string) string {
	tkn := html.NewTokenizer(bytes.NewReader(body))
	depth := 0
	var article string
	block := ""
	errorCode := false

	for !errorCode {
		tt := tkn.Next()
		switch tt {
		case html.ErrorToken:
			errorCode = true
		case html.TextToken:
			if depth > 0 {
				block += string(tkn.Text())
			}
		case html.StartTagToken, html.EndTagToken:
			tn, tAttr := tkn.TagName()
			if string(tn) == tag {
				if tAttr {
					key, attr, _ := tkn.TagAttr()
					if tt == html.StartTagToken && string(key) == keyattr && string(attr) == value {
						depth++
					}
				} else if tt == html.EndTagToken && depth >= 1 {
					depth--
					article += block
					block = ""
				}
			}
		}
	}

	return article
}

func speller(opt SpellOptions) ([]SpellError, error) {
	httpposturl := "https://speller.yandex.net/services/spellservice.json/checkText"

	context := []byte("text=" + url.QueryEscape(opt.Article) + "&lang=" + url.QueryEscape(opt.Lang) + "&options=" + strconv.Itoa(opt.Options) + "&format=" + opt.Format)
	// fmt.Println("Len context: ", len(context))
	request, err := http.NewRequest("POST", httpposturl, bytes.NewBuffer(context))
	if err != nil {
		fmt.Printf("Error NewRequest - %v\n", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// fmt.Println("request before:", request)
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		fmt.Printf("Error doing request - %v\n", err)
	}
	defer response.Body.Close()

	var sperror []SpellError
	// fmt.Println("Article:", opt.Article)
	body, _ := ioutil.ReadAll(response.Body)
	// fmt.Println("response Status:", response.Status)
	// fmt.Println("response Headers:", response.Header)
	// fmt.Println("response Body:", string(body))

	err = json.Unmarshal(body, &sperror)
	if err != nil {
		fmt.Println("error:", err)
	}

	return sperror, err
}

func main() {
	var opt SpellOptions
	var url string
	var urlList string
	// XML structure of RSS
	type RiaRss struct {
		XMLName xml.Name `xml:"rss"`
		Channel struct {
			Title     string `xml:"title"`
			Link      string `xml:"link"`
			Language  string `xml:"language"`
			Copyright string `xml:"copyright"`
			Item      []struct {
				Title     string `xml:"title"`
				Link      string `xml:"link"`
				Guid      string `xml:"guid"`
				Priority  string `xml:"rian:priority"`
				Pubdate   string `xml:"pubDate"`
				Type      string `xml:"rian:type"`
				Category  string `xml:"category"`
				Enclosure string `xml:"enclosure"`
			} `xml:"item"`
		} `xml:"channel"`
	}

	userAgent := `RIA/autotest`
	flag.StringVar(&url, "url", "0", "URL of the article")
	flag.StringVar(&urlList, "xml", "0", "XML with list of the articles")
	flag.StringVar(&opt.Lang, "lang", "ru,en", "Language being tested")
	flag.IntVar(&opt.Options, "options", 14, "Speller options")
	flag.StringVar(&opt.Format, "format", "plain", "Format of the text ")

	flag.Parse()

	if url == "0" && urlList == "0" {
		fmt.Println(("URL or XML must be specified"))
		return
	}

	if url != "0" {
		body, err := getHtmlPage(url, userAgent)
		if err != nil {
			fmt.Printf("Error getHtmlPage - %v\n", err)
		}
		article := getArticle(body, "div", "class", "article__text")
		articleLen := len(article)

		opt.Article = article
		fmt.Println("Article length: ", articleLen)

		sperror, err := speller(opt)
		if err != nil {
			fmt.Printf("Error speller - %v\n", err)
		}
		if len(sperror) > 0 {
			var article_err string
			// fmt.Println("Article: ", article)
			for _, v := range sperror {
				article_err = article[:v.Pos] + "--->" + article[v.Pos+1:v.Pos+v.Len] + "<---" + article[v.Pos+v.Len+1:]
				fmt.Printf("Incorrect world: %v, pos: %v, len: %v\n", v.Word, v.Pos, v.Len)
			}
			fmt.Println("Article with errors: ", article_err)
		}
	} else if urlList != "0" {
		rss := new(RiaRss)
		body, err := getHtmlPage(urlList, userAgent)
		if err != nil {
			fmt.Printf("Error getHtmlPage - %v\n", err)
		}
		err1 := xml.Unmarshal([]byte(body), rss)
		if err != nil {
			fmt.Printf("error: %v", err1)
			return
		}
		for _, value := range rss.Channel.Item {
			fmt.Println("========>", value.Link)
			body, err := getHtmlPage(value.Link, userAgent)
			if err != nil {
				fmt.Printf("Error getHtmlPage - %v\n", err)
			}
			article := getArticle(body, "div", "class", "article__text")
			articleLen := len(article)
			fmt.Println("Total length: ", articleLen)

			opt.Article = article
			sperror, err_sp := speller(opt)
			if len(sperror) > 0 {
				fmt.Println("Article: ", article)
				for _, v := range sperror {
					fmt.Printf("Incoorect world: %v\n", v.Word)
				}
			}
			if err_sp != nil {
				fmt.Printf("Error speller - %v\n", err_sp)
			}
		}
	}
}
