package main

import (
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/godruoyi/go-snowflake"
	"github.com/joho/godotenv"
	"resty.dev/v3"
)

var targets = []string{
	"https://news-api-id.abidf.com/rss/kompas/news",
	"https://news-api-id.abidf.com/rss/cnn/nasional",
	"https://news-api-id.abidf.com/rss/cnn/internasional",
	"https://news-api-id.abidf.com/rss/liputan6/news",
	"https://news-api-id.abidf.com/rss/liputan6/bisnis",
	"https://news-api-id.abidf.com/rss/liputan6/global",
	"https://news-api-id.abidf.com/rss/kumparan/news",
	"https://news-api-id.abidf.com/rss/kumparan/bisnis",
	"https://news-api-id.abidf.com/rss/cnbc/market",
}

func normalizeSource(source string) string {
	switch {
	case strings.Contains(source, "kompas"):
		return "Kompas"
	case strings.Contains(source, "cnn"):
		return "CNN"
	case strings.Contains(source, "liputan6"):
		return "Liputan6"
	case strings.Contains(source, "kumparan"):
		return "Kumparan"
	case strings.Contains(source, "cnbc"):
		return "CNBC"
	default:
		return source
	}
}

var logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelDebug,
}))

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
	for _, target := range targets {
		wg.Add(1)
		go func(target string) {
			defer wg.Done()
			articles, err := StartCrawler(target, client)
			if err != nil {
				logger.Error("Error fetching URL", "url", target, "error", err)
				return
			}
			for _, article := range *articles {
				rawArticles = append(rawArticles, Summarizer{
					Source:  normalizeSource(article.Link),
					Title:   article.Title,
					Content: article.Content,
					Link:    article.Link,
				})
			}
		}(target)
	}

	wg.Wait()
	summarizerResponse, err := Summarize(rawArticles)
	if err != nil {
		logger.Error("Error summarizing articles", "error", err)
		os.Exit(1)
	}

	// save to db
	createdAt := time.Now().Format("2006-01-02 15:04:05")
	for _, article := range summarizerResponse.Articles {
		id := int64(snowflake.ID())
		// normalize sources
		var sources []string
		for _, source := range article.Sources {
			sources = append(sources, normalizeSource(source))
		}

		_, err := db.Exec(`
			INSERT INTO articles (id, title, excerpt, long_content, sources, links, category, ai_model, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, article.Title, article.Excerpt, article.LongContent, strings.Join(sources, ","), strings.Join(article.Sources, ","), article.Category, summarizerResponse.AiModel, createdAt)
		if err != nil {
			logger.Error("Error inserting article", "error", err)
			return
		}
	}

	logger.Info("Done")
	// exit when done
	os.Exit(0)
}
