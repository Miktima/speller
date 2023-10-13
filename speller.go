package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"golang.org/x/net/html"
)

type SpellOptions struct {
	Article []string
	lang    string
	options int
	format  string
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

func getArticle(body []byte, tag, keyattr, value string) []string {
	tkn := html.NewTokenizer(bytes.NewReader(body))
	depth := 0
	var article []string
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
					article = append(article, block)
					block = ""
				}
			}
		}
	}

	return article
}

func speller(opt SpellOptions) error {
	httpposturl := "https://speller.yandex.net/services/spellservice.json/checkTexts"

	context, err := json.Marshal(opt)
	if err != nil {
		fmt.Printf("Error marshal json context - %v\n", err)
		return err
	}

	request, err := http.NewRequest("POST", httpposturl, bytes.NewBuffer(context))
	if err != nil {
		fmt.Printf("Error NewRequest - %v\n", err)
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		fmt.Printf("Error doing request - %v\n", err)
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)
	fmt.Println("response Status:", response.Status)
	fmt.Println("response Body:", string(body))
	return err
}

func main() {
	var opt SpellOptions
	var url string
	userAgent := `RIA/autotest`
	flag.StringVar(&opt.lang, "lang", "ru", "language being tested")
	flag.StringVar(&url, "url", "0", "URL of the article")
	flag.IntVar(&opt.options, "options", 14, "Publish the article")

	flag.Parse()

	body, err := getHtmlPage(url, userAgent)
	if err != nil {
		fmt.Printf("Error getHtmlPage - %v\n", err)
	}
	article := getArticle(body, "div", "class", "article__text")
	articleLen := 0
	for _, value := range article {
		fmt.Println(value)
		articleLen += len(value)
	}
	fmt.Println("Total length: ", articleLen)

	error := speller(opt)
	if error != nil {
		fmt.Printf("Error speller - %v\n", error)
	}

}
