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
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// Структура для отправки статей в Яндекс.Спеллер https://yandex.ru/dev/speller/
type SpellOptions struct {
	Article string
	Lang    string
	Options int
	Format  string
}

// Структура получения результата проверки в JSON-интерфейсе https://yandex.ru/dev/speller/doc/ru/reference/checkText
type SpellError struct {
	Code int
	Pos  int
	Row  int
	Col  int
	Len  int
	Word string
	S    []string
}

//Структура для данных статей
type NewsDataSp struct {
	URL, Article string
}

func getHtmlPage(url, userAgent string) ([]byte, error) {
	// функция получения ресурсов по указанному адресу url с использованием User-Agent
	// возвращает загруженный HTML контент
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Cannot create new request  %s, error: %v\n", url, err)
		return nil, err
	}

	// с User-agent по умолчанию контент не отдается, используем свой
	req.Header.Set("User-Agent", userAgent)

	// Отправляем запрос
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error with GET request: %v\n", err)
		return nil, err
	}

	defer resp.Body.Close()

	// Получаем контент и возвращаем его, как результат работы функции
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error ReadAll")
		return nil, err
	}
	return body, nil
}

func getArticle(body []byte, tag, keyattr, value string) string {
	// Функция получения текста статьи из html контента
	// Текст достается из тега tag с атрибутом keyattr, значение атрибута value
	// Возвращает содержание статьи
	tkn := html.NewTokenizer(bytes.NewReader(body))
	depth := 0
	var article string
	block := ""
	errorCode := false

	// Проходим по всему дереву тегов (пока не встретится html.ErrorToken)
	for !errorCode {
		tt := tkn.Next()
		switch tt {
		case html.ErrorToken:
			errorCode = true
		case html.TextToken:
			if depth > 0 {
				block += string(tkn.Text()) // Если внутри нужного тега, забираем текст из блока
			}
		case html.StartTagToken, html.EndTagToken:
			tn, tAttr := tkn.TagName()
			if string(tn) == tag { // выбираем нужный tag
				if tAttr {
					key, attr, _ := tkn.TagAttr()
					if tt == html.StartTagToken && string(key) == keyattr && string(attr) == value {
						depth++ // нужный тег открывается
					}
				} else if tt == html.EndTagToken && depth >= 1 {
					depth--
					article += block // Когда блок закрывается, добавляем текст из блока в основной текст
					block = ""
				}
			}
		}
	}
	return article
}

func speller(opt SpellOptions) ([]SpellError, error) {
	// Функция отправки запроса в Яндекс.Спеллер
	// На входе структура для отпраке статей
	// На выходе - структура описания ошибок

	// Адрес сервиса
	httpposturl := "https://speller.yandex.net/services/spellservice.json/checkText"

	// Формируем текст запроса в urlencoded
	context := []byte("text=" + url.QueryEscape(opt.Article) + "&lang=" + url.QueryEscape(opt.Lang) + "&options=" + strconv.Itoa(opt.Options) + "&format=" + opt.Format)
	// fmt.Println("Len context: ", len(context))

	// Отправляем POST запрос
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

	// Получаем результат
	body, _ := ioutil.ReadAll(response.Body)
	// fmt.Println("response Status:", response.Status)
	// fmt.Println("response Headers:", response.Header)
	// fmt.Println("response Body:", string(body))

	// Переводим результат в структуру
	err = json.Unmarshal(body, &sperror)
	if err != nil {
		fmt.Println("error:", err)
	}

	return sperror, err
}

func addtags(article string, subs []string, sperror []SpellError) string {
	// Функция добавления тегов к найденным в статье ошибкам
	// на входе: содержание статьи, срез с начальным и конечным тегами и структура с ошибками
	// на выходе - статья с тегами вокруг ошибок
	article_err := ""
	// Требуется использовать руны, иначе положение слова не определить
	ar := []rune(article)
	startPos := 0
	for _, v := range sperror {
		article_err += string(ar[startPos:v.Pos]) + subs[0] + string(ar[v.Pos:v.Pos+v.Len]) + subs[1]
		startPos = v.Pos + v.Len
	}
	article_err += string(ar[startPos:])
	return article_err
}

func inSlice(tSlice []NewsDataSp, url string) bool {
	// Проверка содержится ли URL в срезе
	for _, v := range tSlice {
		if v.URL == url {
			return true
		}
	}
	return false
}

func main() {
	var opt SpellOptions
	var url string
	var urlList string
	var userAgent string
	var collect int
	var listDataNews []NewsDataSp
	var dataNews NewsDataSp

	errorFile := "error.html"
	dataFile := "datanewssp.json"

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

	// Error codes of the Yandex speller (https://yandex.ru/dev/speller/doc/ru/reference/error-codes)
	errorCode := map[int]string{
		1: "Слова нет в словаре",
		2: "Повтор слова",
		3: "Неверное употребление прописных и строчных букв",
		4: "Текст содержит слишком много ошибок",
	}

	// Теги для выделения ошибок
	subs_cl := []string{"<mark>", "</mark>"}

	// Голова HTML для вывода результата в файл
	html_head := `<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
		<title>Article errors</title>
	</head>
	<body>`

	// Ключи для командной строки
	flag.StringVar(&url, "url", "0", "URL of the article")
	flag.StringVar(&urlList, "xml", "0", "XML with list of the articles")
	flag.StringVar(&opt.Lang, "lang", "ru,en", "Language being tested")
	flag.IntVar(&opt.Options, "options", 14, "Speller options")
	flag.StringVar(&opt.Format, "format", "plain", "Format of the text ")
	flag.StringVar(&userAgent, "uagent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit", "User Agent")
	flag.IntVar(&collect, "collect", 0, "Collect articles with codes 2 and 3 (=1) or not (=0)")

	flag.Parse()

	path, _ := os.Executable()
	path = path[:strings.LastIndex(path, "/")+1]

	if collect == 1 {
		// Читаем файл с сохраненными статьями
		if _, err := os.Stat(path + dataFile); err == nil {
			// Open our jsonFile
			byteValue, err := os.ReadFile(path + dataFile)
			// if we os.ReadFile returns an error then handle it
			if err != nil {
				fmt.Println(err)
			}
			// defer the closing of our jsonFile so that we can parse it later on
			// var listHash []ArticleH
			err = json.Unmarshal(byteValue, &listDataNews)
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	// Если не указан url и xml выходим: проверять нечего
	if url == "0" && urlList == "0" {
		fmt.Println(("URL or XML must be specified"))
		return
	}

	var htmlerr string
	colErr := 0

	// Проверка для единичного адреса
	if url != "0" {
		// Получаем html контент
		body, err := getHtmlPage(url, userAgent)
		if err != nil {
			fmt.Printf("Error getHtmlPage - %v\n", err)
		}
		// Получаем текст статьи
		article := getArticle(body, "div", "class", "article__title") + "\n"
		article += getArticle(body, "div", "class", "article__text")
		articleLen := len(article)

		// Сведения о статье
		opt.Article = article
		htmlerr = "<p>Link to the article: <a href='" + url + "'>" + url + "</a></p>\n"
		fmt.Println("Article length: ", articleLen)
		htmlerr += "<p>Article length: " + strconv.Itoa(articleLen) + "</p>\n"

		sperror, err := speller(opt)
		if err != nil {
			fmt.Printf("Error speller - %v\n", err)
		}
		// Если есть ошибки, готовим сведения о них
		if len(sperror) > 0 {
			article_err := addtags(article, subs_cl, sperror)
			for _, v := range sperror {
				fmt.Printf("Incorrect world: %v, pos: %v, len: %v, error: %v\n", v.Word, v.Pos, v.Len, errorCode[v.Code])
				htmlerr += fmt.Sprintf("<p>Incorrect world: %v, pos: %v, len: %v, error: %v</p>\n", v.Word, v.Pos, v.Len, errorCode[v.Code])
				if v.Code == 2 || v.Code == 3 {
					colErr = 1
				}
			}
			fmt.Println("Article with errors: ", article_err)
			htmlerr += "<p>" + article_err + "</p>\n"
			// Выводим результаты проверки в файл
			err := os.WriteFile(errorFile, []byte(html_head+htmlerr+"</body>"), 0644)
			if err != nil {
				fmt.Printf("Error write HTML file - %v\n", err)
			}
		}
		if colErr == 1 && collect == 1 {
			if !inSlice(listDataNews, url) {
				dataNews.URL = url
				dataNews.Article = article
				listDataNews = append(listDataNews, dataNews)
			}
			colErr = 0
		}
		htmlerr += "<br><br>\n"
	} else if urlList != "0" {
		// Проверка ссылок на статьи из xml файла
		rss := new(RiaRss)
		// Получаем текст RSS
		body, err := getHtmlPage(urlList, userAgent)
		if err != nil {
			fmt.Printf("Error getHtmlPage - %v\n", err)
		}
		// Разбираем полученный RSS
		err1 := xml.Unmarshal([]byte(body), rss)
		if err != nil {
			fmt.Printf("error: %v", err1)
			return
		}

		var article_err string
		totalLng := 0
		// Пребираем все ссылки в RSS
		for _, value := range rss.Channel.Item {
			fmt.Println("========>", value.Link)
			htmlerr += "<p>Link to the article: <a href='" + value.Link + "'>" + value.Link + "</a></p>\n"
			// Получаем HTML контент
			body, err := getHtmlPage(value.Link, userAgent)
			if err != nil {
				fmt.Printf("Error getHtmlPage - %v\n", err)
			}
			// Получаем текст статьи
			article := getArticle(body, "div", "class", "article__title") + "\n"
			article = getArticle(body, "div", "class", "article__text")
			articleLen := len(article)
			fmt.Println("Total length: ", articleLen)
			htmlerr += "<p>Article length: " + strconv.Itoa(articleLen) + "</p>\n"
			totalLng += articleLen
			opt.Article = article
			sperror, err_sp := speller(opt)

			// Если есть ошибки в тексте, готовим вывод результата
			if len(sperror) > 0 {
				article_err = addtags(article, subs_cl, sperror)
				for _, v := range sperror {
					fmt.Printf("Incorrect world: %v, pos: %v, len: %v, error: %v\n", v.Word, v.Pos, v.Len, errorCode[v.Code])
					htmlerr += fmt.Sprintf("<p>Incorrect world: %v, pos: %v, len: %v, error: %v</p>\n", v.Word, v.Pos, v.Len, errorCode[v.Code])
					if v.Code == 2 || v.Code == 3 {
						colErr = 1
					}
				}
				fmt.Println("Article with errors:", article_err)
				htmlerr += "<p>" + article_err + "</p>\n"
			}
			if err_sp != nil {
				fmt.Printf("Error speller - %v\n", err_sp)
			}
			if colErr == 1 && collect == 1 {
				if !inSlice(listDataNews, url) {
					dataNews.URL = value.Link
					dataNews.Article = article
					listDataNews = append(listDataNews, dataNews)
				}
				colErr = 0
			}
			htmlerr += "<br><br>\n"
		}
		htmlerr += "<p>Total article length: " + strconv.Itoa(totalLng) + "</p>\n"
		err = os.WriteFile(path+errorFile, []byte(html_head+htmlerr+"</body>"), 0644)
		if err != nil {
			fmt.Printf("Error write HTML file - %v\n", err)
		}
		if collect == 1 {
			//записываем данные в файл, если они есть
			if len(listDataNews) > 0 {
				f, err := os.OpenFile(path+dataFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
				if err != nil {
					fmt.Printf("Error opening to %v file - %v\n", dataFile, err)
				}
				defer f.Close()

				arData, _ := json.MarshalIndent(listDataNews, "", " ")
				_, err = f.Write(arData)
				if err != nil {
					fmt.Printf("Error write Article data - %v\n", err)
				}
			}
		}
	}
}
