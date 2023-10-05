package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/net/html"
)

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

func speller(artcile []string)

func main() {
	// s := `<p>Links:</p><ul><li><a href="foo">Foo!!!</a><li><a href="/bar/baz">This is a test</a></ul>`
	url := `https://ria.ru/20231005/moldaviya-1900611273.html`
	userAgent := `RIA/autotest`

	body, err := getHtmlPage(url, userAgent)
	if err != nil {
		fmt.Printf("Error getHtmlPage - %v\n", err)
	}
	article := getArticle(body, "div", "class", "article__text")
	for _, value := range article {
		fmt.Println(value)
	}
}
