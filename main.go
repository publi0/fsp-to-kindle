package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/bmaupin/go-epub"
	"github.com/gocolly/colly/v2"
	"github.com/google/uuid"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Article struct {
	text string
	name string
}

type Fivefilters struct {
	Tittle   string `json:"att_title"`
	Body     string `json:"att_body"`
	Btype    string `json:"att_type"`
	Language string `json:"att_lang"`
}

func main() {
	downloadCoverPage()

	fspLinks := findLinks()

	articles := getArticles(fspLinks)

	createEpub(articles)

}

func createEpub(articles []Article) {
	currentTime := time.Now()
	tittle := fmt.Sprintf("Folha de SP - %s", currentTime.Format("02-01-2006"))
	e := epub.NewEpub(tittle)

	image, _ := e.AddImage("img/cover.jpg", "cover.png")
	e.SetCover(image, "")
	e.SetAuthor("Folha de São Paulo")

	for i, article := range articles {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(article.text))
		if err != nil {
			log.Fatal(err)
		}

		link, exists := doc.Find("p").Children().Attr("src")
		if exists {
			filename := fmt.Sprintf("%s.jpg", uuid.New())
			downloadFile(link, filename)
			addImage, _ := e.AddImage("img/"+filename, filename)
			article.text = strings.ReplaceAll(article.text, link, addImage)
		}

		e.AddSection(article.text, fmt.Sprintf("%d - %s", i, article.name), "", "")
	}

	err := e.Write(fmt.Sprintf("%s.epub", tittle))
	if err != nil {
		fmt.Println("Error saving file")
	}
}

func getArticles(fspLinks map[string]string) []Article {
	cArticles := make(chan Article)

	for name, link := range fspLinks {
		fmt.Println("Requesting: ", name)
		go getParsedLink(link, name, cArticles)
		time.Sleep(50 * time.Millisecond)
	}

	var articles []Article

	for i := 1; i < len(fspLinks); i++ {
		articles = append(articles, <-cArticles)
	}
	return articles
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

func downloadCoverPage() {
	c := colly.NewCollector()
	var link string
	c.OnHTML(".edition", func(e *colly.HTMLElement) {
		linkf, exists := e.DOM.Children().Attr("src")
		if exists {
			link = linkf
		}
	})
	c.Visit("https://acervo.folha.uol.com.br/digital/")
	downloadFile(link, "cover.jpg")
}

func getParsedLink(link string, name string, textData chan Article) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://pushtokindle.fivefilters.org/send.php?context=iframe&links=1&url=%s", link), nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authority", "pushtokindle.fivefilters.org")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,pt-BR;q=0.8,pt;q=0.7")
	req.Header.Set("Dnt", "1")
	req.Header.Set("Sec-Ch-Ua", "\"Not?A_Brand\";v=\"8\", \"Chromium\";v=\"108\", \"Google Chrome\";v=\"108\"")
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", "\"Linux\"")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	fmt.Println("Response status:", resp.Status)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var parsed Fivefilters

	bytes, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(bytes, &parsed)
	if err != nil {
		log.Fatal(err)
	}

	textData <- Article{parsed.Body, parsed.Tittle}
}

func downloadFile(URL, fileName string) error {
	fmt.Println("Downloading image: ", URL)
	response, err := http.Get(URL)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return errors.New("Received non 200 response code")
	}
	file, err := os.Create("img/" + fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	return nil
}
