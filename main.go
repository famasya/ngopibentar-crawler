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
	var client = resty.New().
		SetRetryCount(3).
		SetRetryWaitTime(2 * time.Second).
		SetRetryMaxWaitTime(10 * time.Second)
	var wg sync.WaitGroup

	var rawArticles []CrawlerResult
	for _, target := range targets {
		wg.Add(1)
		go func(target string) {
			defer wg.Done()
			articles, err := StartCrawler(target, client)
			if err != nil {
				logger.Error("Error fetching URL", "url", target, "error", err)
				return
			}
			rawArticles = append(rawArticles, *articles...)
		}(target)
	}

	wg.Wait()
	logger.Debug("Raw articles", "articles", rawArticles)
	groups, err := Grouper(rawArticles)
	if err != nil {
		logger.Error("Error grouping articles", "error", err)
		os.Exit(1)
	}

	// sleep for 3 second
	logger.Debug("Sleeping for 3 second")
	time.Sleep(3 * time.Second)

	// summarize each group
	for _, group := range groups.Groups {
		var articles []Summarizer
		for _, g := range group {
			articles = append(articles, Summarizer{
				Source:  g.Source,
				Title:   g.Title,
				Content: g.Content,
				Link:    g.Link,
			})
		}
		summarizerResponse, err := Summarize(articles)
		if err != nil {
			logger.Error("Error summarizing articles", "error", err)
			continue
		}

		// for each summary, save to db
		createdAt := time.Now().Format("2006-01-02 15:04:05")
		for _, article := range summarizerResponse.Articles {
			id := int64(snowflake.ID())
			// merge sources
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
			logger.Debug("Article saved", "id", id)
		}
	}

	logger.Info("Done")
	// exit when done
	os.Exit(0)
}
