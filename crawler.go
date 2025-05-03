package main

import (
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"resty.dev/v3"
)

type Response struct {
	Success bool        `json:"success"`
	Message interface{} `json:"message"`
	Data    struct {
		Link        string `json:"link"`
		Description string `json:"description"`
		Title       string `json:"title"`
		Posts       []struct {
			Link        string    `json:"link"`
			Title       string    `json:"title"`
			PubDate     time.Time `json:"pubDate"`
			Description string    `json:"description"`
			Thumbnail   string    `json:"thumbnail"`
		} `json:"posts"`
	} `json:"data"`
}

type CrawlerResult struct {
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Link        string    `json:"link"`
	PublishedAt time.Time `json:"published_at"`
}

func StartCrawler(url string, client *resty.Client) (*[]CrawlerResult, error) {
	var response Response
	_, err := client.R().
		SetResult(&response).
		Get(url)
	if err != nil {
		slog.Error("Error fetching URL [1]", "url", url, "error", err)
		return nil, err
	}

	if !response.Success {
		slog.Error("Error fetching URL [2]", "url", url, "message", response.Message)
		return nil, err
	}

	var results []CrawlerResult
	for _, post := range response.Data.Posts[:2] {
		var content *string
		switch {
		case strings.Contains(post.Link, "antara"):
			content, err = GetContentAntara(post.Link, client)
			if err != nil {
				slog.Error("Error fetching content [3]", "url", post.Link, "error", err)
				continue
			}
		case strings.Contains(post.Link, "tempo"):
			content, err = GetContentTempo(post.Link, client)
			if err != nil {
				slog.Error("Error fetching content [3]", "url", post.Link, "error", err)
				continue
			}
		case strings.Contains(post.Link, "cnbc"):
			content, err = GetContentCNBC(post.Link, client)
			if err != nil {
				slog.Error("Error fetching content [3]", "url", post.Link, "error", err)
				continue
			}
		case strings.Contains(post.Link, "cnn"):
			content, err = GetContentCNN(post.Link, client)
			if err != nil {
				slog.Error("Error fetching content [3]", "url", post.Link, "error", err)
				continue
			}
		case strings.Contains(post.Link, "sindonews"):
			content, err = GetContentSindonews(post.Link, client)
			if err != nil {
				slog.Error("Error fetching content [3]", "url", post.Link, "error", err)
				continue
			}
		}

		results = append(results, CrawlerResult{
			Title:       post.Title,
			Content:     *content,
			Link:        post.Link,
			PublishedAt: post.PubDate,
		})
	}

	return &results, nil
}

func GetContentAntara(url string, client *resty.Client) (*string, error) {
	slog.Info("--> Processing URL", "url", url)
	response, err := client.R().
		Get(url)
	if err != nil {
		slog.Error("Error fetching content [1]", "url", url, "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		slog.Error("Error fetching content [2]", "url", url, "error", err)
		return nil, err
	}

	var content string
	doc.Find(".post-content p").Each(func(i int, s *goquery.Selection) {
		// ignore if it contains `script` or it contains `.text-muted` class or it contains `span.baca-juga`
		if s.Has("script").Length() > 0 || s.HasClass("text-muted") || s.Find("span.baca-juga").Length() > 0 {
			return
		}
		content += s.Text() + " "
	})

	re := regexp.MustCompile(`\s{2,}`)
	cleaned := re.ReplaceAllString(content, " ")
	trimmed := strings.TrimSpace(cleaned)
	return &trimmed, nil
}

func GetContentTempo(url string, client *resty.Client) (*string, error) {
	slog.Info("--> Processing URL", "url", url)
	response, err := client.R().
		Get(url)
	if err != nil {
		slog.Error("Error fetching content [1]", "url", url, "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		slog.Error("Error fetching content [2]", "url", url, "error", err)
		return nil, err
	}

	var content string
	doc.Find("#content-wrapper").Each(func(i int, s *goquery.Selection) {
		// ignore if it contains "Pilihan Editor" or "berkontribusi dalam penulisan artikel ini"
		if strings.Contains(s.Text(), "Pilihan Editor") || strings.Contains(s.Text(), "berkontribusi dalam penulisan artikel ini") {
			return
		}
		content += s.Text() + " "
	})

	re := regexp.MustCompile(`\s{2,}`)
	cleaned := re.ReplaceAllString(content, " ")
	trimmed := strings.TrimSpace(cleaned)
	return &trimmed, nil
}

func GetContentCNBC(url string, client *resty.Client) (*string, error) {
	slog.Info("--> Processing URL", "url", url)
	response, err := client.R().
		Get(url)
	if err != nil {
		slog.Error("Error fetching content [1]", "url", url, "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		slog.Error("Error fetching content [2]", "url", url, "error", err)
		return nil, err
	}

	var content string
	doc.Find(".detail-text p").Each(func(i int, s *goquery.Selection) {
		// ignore if it contains `linksisip` class
		if s.HasClass("linksisip") {
			return
		}
		content += s.Text() + " "
	})

	re := regexp.MustCompile(`\s{2,}`)
	cleaned := re.ReplaceAllString(content, " ")
	trimmed := strings.TrimSpace(cleaned)
	return &trimmed, nil
}

func GetContentCNN(url string, client *resty.Client) (*string, error) {
	slog.Info("--> Processing URL", "url", url)
	response, err := client.R().
		Get(url)
	if err != nil {
		slog.Error("Error fetching content [1]", "url", url, "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		slog.Error("Error fetching content [2]", "url", url, "error", err)
		return nil, err
	}

	var content string
	doc.Find(".detail-text p").Each(func(i int, s *goquery.Selection) {
		// ignore if it contains `para_caption` class
		if s.HasClass("para_caption") {
			return
		}
		content += s.Text() + " "
	})

	re := regexp.MustCompile(`\s{2,}`)
	cleaned := re.ReplaceAllString(content, " ")
	trimmed := strings.TrimSpace(cleaned)
	return &trimmed, nil
}

func GetContentSindonews(url string, client *resty.Client) (*string, error) {
	slog.Info("--> Processing URL", "url", url)
	response, err := client.R().
		Get(url)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		slog.Error("Error fetching content [2]", "url", url, "error", err)
		return nil, err
	}

	var content string
	doc.Find(".detail-desc").Each(func(i int, s *goquery.Selection) {
		content += s.Text() + " "
	})

	re := regexp.MustCompile(`\s{2,}`)
	cleaned := re.ReplaceAllString(content, " ")
	trimmed := strings.TrimSpace(cleaned)
	return &trimmed, nil
}
