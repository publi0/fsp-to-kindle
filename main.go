package main

import (
	"fmt"
	"github.com/bmaupin/go-epub"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

type Article struct {
	text string
	name string
}

func main() {

	fspLinks := findLinks()

	articles := getArticles(fspLinks)

	// Create a new EPUB
	currentTime := time.Now()
	tittle := fmt.Sprintf("Folha de SP - %s", currentTime.Format("02-01-2006"))
	e := epub.NewEpub(tittle)

	// Set the author
	e.SetAuthor("Folha de São Paulo")

	for i, article := range articles {
		e.AddSection(strings.ReplaceAll(article.text, "[image]", "\n\n"), fmt.Sprintf("%d - %s", i, article.name), "", "")
	}

	// Write the EPUB
	err := e.Write(fmt.Sprintf("%s.epub", tittle))
	if err != nil {
		fmt.Println("Error saving file")
	}

}

func getArticles(fspLinks map[string]string) []Article {
	cArticles := make(chan Article)

	for name, link := range fspLinks {
		fmt.Println("Requesting: ", name)
		go getTextify(link, name, cArticles)
		time.Sleep(50 * time.Millisecond)
	}

	var articles []Article

	for i := 1; i < len(fspLinks); i++ {
		articles = append(articles, <-cArticles)
	}
	return articles
}

func getTextify(link string, name string, textData chan Article) {
	resp, err := http.Get("https://txtify.it/" + link)
	if err != nil {
		panic(err)
		textData <- Article{resp.Status, name}
		fmt.Println("Response status:", resp.Status)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Response status:", resp.Status)

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	textData <- Article{string(responseData), name}
}

func findLinks() map[string]string {

	c := colly.NewCollector(
		colly.AllowedDomains("www1.folha.uol.com.br"),
	)

	fspLinks := make(map[string]string)
	c.OnHTML(".c-channel__headline", func(e *colly.HTMLElement) {

		link, _ := e.DOM.Children().Attr("href")
		fspLinks[e.Text] = link
	})

	c.Visit("https://www1.folha.uol.com.br/fsp")
	return fspLinks
}
