package main

import (
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/godruoyi/go-snowflake"
	"resty.dev/v3"
)

type Response struct {
	AvailableCategories []string  `json:"available_categories"`
	CachedAt            time.Time `json:"cached_at"`
	Items               []struct {
		Title       string    `json:"title"`
		Link        string    `json:"link"`
		Description string    `json:"description"`
		PublishedAt time.Time `json:"published_at"`
		Thumbnail   string    `json:"thumbnail"`
	} `json:"items"`
}

type CrawlerResult struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Link        string    `json:"link"`
	PublishedAt time.Time `json:"published_at"`
	Source      string    `json:"source"`
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

func StartCrawler(url string, client *resty.Client) (*[]CrawlerResult, error) {
	var response Response
	_, err := client.R().
		SetResult(&response).
		Get(url)
	if err != nil {
		logger.Error("Error fetching URL [1]", "url", url, "error", err)
		return nil, err
	}

	var results []CrawlerResult
	for _, post := range response.Items {
		var content *string
		switch {
		case strings.Contains(post.Link, "kompas"):
			content, err = GetContentKompas(post.Link+"?page=all", client)
			if err != nil {
				logger.Error("Error fetching content [3]", "url", post.Link, "error", err)
				continue
			}
		case strings.Contains(post.Link, "liputan6"):
			content, err = GetContentLiputan6(post.Link, client)
			if err != nil {
				logger.Error("Error fetching content [3]", "url", post.Link, "error", err)
				continue
			}
		case strings.Contains(post.Link, "cnbc"):
			content, err = GetContentCNBC(post.Link, client)
			if err != nil {
				logger.Error("Error fetching content [3]", "url", post.Link, "error", err)
				continue
			}
		case strings.Contains(post.Link, "cnn"):
			content, err = GetContentCNN(post.Link, client)
			if err != nil {
				logger.Error("Error fetching content [3]", "url", post.Link, "error", err)
				continue
			}
		case strings.Contains(post.Link, "kumparan"):
			content, err = GetContentKumparan(post.Link+"/full", client)
			if err != nil {
				logger.Error("Error fetching content [3]", "url", post.Link, "error", err)
				continue
			}
		}

		// omit short content
		if len(*content) < 100 {
			continue
		}

		id := int64(snowflake.ID())
		results = append(results, CrawlerResult{
			ID:          id,
			Title:       post.Title,
			Content:     *content,
			Link:        post.Link,
			Source:      normalizeSource(post.Link),
			PublishedAt: post.PublishedAt,
		})
	}

	return &results, nil
}

func GetContentKompas(url string, client *resty.Client) (*string, error) {
	logger.Info("--> Processing URL", "url", url)
	response, err := client.R().
		Get(url)
	if err != nil {
		logger.Error("Error fetching content [1]", "url", url, "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		logger.Error("Error fetching content [2]", "url", url, "error", err)
		return nil, err
	}

	var content string
	doc.Find(".read__content p").Each(func(i int, s *goquery.Selection) {
		if strings.Contains(s.Text(), "Baca juga") {
			return
		}
		content += s.Text() + " "
	})

	re := regexp.MustCompile(`\s{2,}`)
	cleaned := re.ReplaceAllString(content, " ")
	trimmed := strings.TrimSpace(cleaned)
	return &trimmed, nil
}

func GetContentLiputan6(url string, client *resty.Client) (*string, error) {
	logger.Info("--> Processing URL", "url", url)
	response, err := client.R().
		Get(url)
	if err != nil {
		logger.Error("Error fetching content [1]", "url", url, "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		logger.Error("Error fetching content [2]", "url", url, "error", err)
		return nil, err
	}

	var content string
	doc.Find(".article-content-body__item-content p").Each(func(i int, s *goquery.Selection) {
		content += s.Text() + " "
	})

	re := regexp.MustCompile(`\s{2,}`)
	cleaned := re.ReplaceAllString(content, " ")
	trimmed := strings.TrimSpace(cleaned)
	return &trimmed, nil
}

func GetContentCNBC(url string, client *resty.Client) (*string, error) {
	logger.Info("--> Processing URL", "url", url)
	response, err := client.R().
		Get(url)
	if err != nil {
		logger.Error("Error fetching content [1]", "url", url, "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		logger.Error("Error fetching content [2]", "url", url, "error", err)
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
	logger.Info("--> Processing URL", "url", url)
	response, err := client.R().
		Get(url)
	if err != nil {
		logger.Error("Error fetching content [1]", "url", url, "error", err)
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		logger.Error("Error fetching content [2]", "url", url, "error", err)
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

func GetContentKumparan(url string, client *resty.Client) (*string, error) {
	logger.Info("--> Processing URL", "url", url)
	response, err := client.R().
		Get(url)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		logger.Error("Error fetching content [2]", "url", url, "error", err)
		return nil, err
	}

	var content string
	doc.Find("span[data-qa-id=story-paragraph]").Each(func(i int, s *goquery.Selection) {
		content += s.Text() + " "
	})

	re := regexp.MustCompile(`\s{2,}`)
	cleaned := re.ReplaceAllString(content, " ")
	trimmed := strings.TrimSpace(cleaned)
	return &trimmed, nil
}
