package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

func main() {
	// s := `<p>Links:</p><ul><li><a href="foo">Foo!!!</a><li><a href="/bar/baz">This is a test</a></ul>`
	url := `https://ria.ru/20231004/diversant-1900408924.html`
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error opening URL: ", url)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error ReadAll")
	}
	fmt.Println(string(body))
	tkn := html.NewTokenizer(strings.NewReader(string(body)))
	var ftkn func(*html.Tokenizer)
	ftkn = func(t *html.Tokenizer) {
		depth := 0
		for {
			tt := t.Next()
			switch tt {
			case html.ErrorToken:
				return
			case html.TextToken:
				if depth > 0 {
					fmt.Println(string(t.Text()))
				}
			case html.StartTagToken, html.EndTagToken:
				tn, _ := t.TagName()
				if len(tn) == 1 && tn[0] == 'a' {
					if tt == html.StartTagToken {
						depth++
					} else {
						depth--
					}
				}
			}
		}
	}
	ftkn(tkn)
	defer resp.Body.Close()
}
