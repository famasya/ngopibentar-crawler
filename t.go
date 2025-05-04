package main

// import (
// 	"log"

// 	"github.com/PuerkitoBio/goquery"
// 	"resty.dev/v3"
// )

// func main() {
// 	var client = resty.New()

// 	log.Print("hello")
// 	res, err := client.R().
// 		Get("https://kumparan.com/kumparanhits/ryan-adriandhy-ungkap-alasan-don-di-film-jumbo-tak-punya-lubang-telinga-24zyCJDcWG3/full")
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	doc, err := goquery.NewDocumentFromReader(res.Body)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	var content string
// 	doc.Find("span[data-qa-id=story-paragraph]").Each(func(i int, s *goquery.Selection) {
// 		content += s.Text() + " "
// 	})

// 	log.Print(content)
// }
