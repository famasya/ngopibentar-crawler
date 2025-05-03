package main

import (
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"resty.dev/v3"
)

var categories = map[string][]string{
	"recent": {
		"https://api-berita-indonesia.vercel.app/antara/terbaru",
		// "https://api-berita-indonesia.vercel.app/cnbc/terbaru",
		// "https://api-berita-indonesia.vercel.app/cnn/terbaru",
	},
	// "national": {
	// 	"https://api-berita-indonesia.vercel.app/tempo/nasional",
	// 	"https://api-berita-indonesia.vercel.app/sindonews/nasional",
	// 	"https://api-berita-indonesia.vercel.app/cnn/nasional",
	// },
	// "business": {
	// 	"https://api-berita-indonesia.vercel.app/tempo/bisnis",
	// 	"https://api-berita-indonesia.vercel.app/antara/ekonomi",
	// 	"https://api-berita-indonesia.vercel.app/cnn/ekonomi",
	// 	"https://api-berita-indonesia.vercel.app/sindonews/ekbis",
	// },
}

func normalizeSource(source string) string {
	switch {
	case strings.Contains(source, "antara"):
		return "Antara"
	case strings.Contains(source, "cnbc"):
		return "CNBC"
	case strings.Contains(source, "cnn"):
		return "CNN"
	case strings.Contains(source, "tempo"):
		return "Tempo"
	case strings.Contains(source, "sindonews"):
		return "SindoNews"
	default:
		return source
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db, cleanup, err := InitDB()
	if err != nil {
		log.Fatal("Error initializing database")
	}
	defer cleanup()
	var client = resty.New()
	var wg sync.WaitGroup

	var rawArticles []Summarizer
	for category, urls := range categories {
		for _, url := range urls {
			wg.Add(1)
			go func(category, url string) {
				defer wg.Done()
				slog.Info("Processing parent URL", "category", category, "url", url)
				articles, err := StartCrawler(url, client)
				if err != nil {
					slog.Error("Error fetching URL", "url", url, "error", err)
					return
				}
				for _, article := range *articles {
					rawArticles = append(rawArticles, Summarizer{
						Source:   normalizeSource(article.Link),
						Title:    article.Title,
						Content:  article.Content,
						Link:     article.Link,
						Category: category,
					})
				}
			}(category, url)
		}
	}

	wg.Wait()
	summarizerResponse, err := Summarize(rawArticles)
	if err != nil {
		slog.Error("Error summarizing articles", "error", err)
		os.Exit(1)
	}

	// save to db
	createdAt := time.Now().Format("2006-01-02 15:04:05")
	for _, article := range summarizerResponse.Summary {
		_, err := db.Exec(`
			INSERT INTO articles (title, content, sources, category, ai_model, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, article.Title, article.Content, strings.Join(article.Sources, ","), summarizerResponse.Category, summarizerResponse.AiModel, createdAt)
		if err != nil {
			slog.Error("Error inserting article", "error", err)
			return
		}
	}

	slog.Info("Done")
	// exit when done
	os.Exit(0)
}
