package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/samber/lo"
	"google.golang.org/genai"
)

type NewsItem struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type GeminiGroupResponse struct {
	Groups [][]int64 `json:"groups"`
}

type GrouperResponse struct {
	Groups [][]CrawlerResult `json:"groups"`
}

func Grouper(payload []CrawlerResult) (*GrouperResponse, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	// rewrite payload
	var newsItem []NewsItem
	for _, item := range payload {
		newsItem = append(newsItem, NewsItem{
			ID:    item.ID,
			Title: item.Title,
		})
	}

	jsonPayload, err := json.Marshal(newsItem)
	if err != nil {
		return nil, fmt.Errorf("error marshaling payload: %v", err)
	}
	logger.Info("Grouping news")
	parts := []*genai.Part{
		{Text: string(jsonPayload)},
	}
	result, err := client.Models.GenerateContent(
		ctx, "gemini-2.0-flash",
		[]*genai.Content{genai.NewContentFromParts(parts, genai.RoleUser)}, &genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{
					{Text: systemInstructionGrouper},
				},
			},
			ResponseMIMEType: "application/json",
			ResponseSchema: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"groups": {
						Type: genai.TypeArray,
						Items: &genai.Schema{
							Type: genai.TypeArray,
							Items: &genai.Schema{
								Type: genai.TypeNumber,
							},
						},
					},
				},
				Required: []string{"groups"},
			},
		})
	if err != nil {
		return nil, fmt.Errorf("error generating content: %v", err)
	}

	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned from Gemini API")
	}

	var response GeminiGroupResponse
	logger.Debug("grouping result", "result", result.Candidates[0].Content.Parts[0].Text)
	err = json.Unmarshal([]byte(result.Candidates[0].Content.Parts[0].Text), &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling result: %v", err)
	}

	// convert response to GrouperResponse
	var grouperResponse GrouperResponse
	for _, group := range response.Groups {
		var crawlerResults []CrawlerResult
		for _, id := range group {
			// find appropriate crawler result based on id
			result, _ := lo.Find(payload, func(item CrawlerResult) bool {
				return item.ID == id
			})
			crawlerResults = append(crawlerResults, result)
		}
		grouperResponse.Groups = append(grouperResponse.Groups, crawlerResults)
	}

	return &grouperResponse, nil
}
