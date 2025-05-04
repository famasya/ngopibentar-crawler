package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"google.golang.org/genai"
)

type Summarizer struct {
	Source  string `json:"source"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Link    string `json:"link"`
}

type AIResponse struct {
	Excerpt     string   `json:"excerpt"`
	LongContent string   `json:"long_content"`
	Sources     []string `json:"sources"`
	Title       string   `json:"title"`
	Category    string   `json:"category"`
}

type SummarizerResponse struct {
	Articles []AIResponse `json:"articles"`
	AiModel  string       `json:"ai_model"`
	Category string       `json:"category"`
}

type GeminiResponse struct {
	Articles []AIResponse `json:"articles"`
}

func Summarize(payload []Summarizer) (*SummarizerResponse, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("no articles to summarize")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	// parse payload to string
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling payload: %v", err)
	}

	logger.Info("Generating content")
	parts := []*genai.Part{
		{Text: string(jsonPayload)},
	}
	aiModel := "gemini-2.0-flash"
	result, err := client.Models.GenerateContent(
		ctx, "gemini-2.0-flash",
		[]*genai.Content{genai.NewContentFromParts(parts, genai.RoleUser)}, &genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{
					{Text: systemInstruction},
				},
			},
			ResponseMIMEType: "application/json",
			ResponseSchema: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"articles": {
						Type: genai.TypeArray,
						Items: &genai.Schema{
							Type: genai.TypeObject,
							Properties: map[string]*genai.Schema{
								"title": {
									Type: genai.TypeString,
								},
								"excerpt": {
									Type: genai.TypeString,
								},
								"long_content": {
									Type: genai.TypeString,
								},
								"sources": {
									Type: genai.TypeArray,
									Items: &genai.Schema{
										Type: genai.TypeString,
									},
								},
								"category": {
									Type: genai.TypeString,
									Enum: []string{"national", "international", "entertainment", "sports", "technology", "business", "politics"},
								},
							},
							Required: []string{"title", "excerpt", "long_content", "sources", "category"},
						},
					},
				},
				Required: []string{"articles"},
			},
		})
	if err != nil {
		return nil, fmt.Errorf("error generating content: %v", err)
	}

	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned from Gemini API")
	}

	var response GeminiResponse
	logger.Debug("summarization result", "result", result.Candidates[0].Content.Parts[0].Text)
	err = json.Unmarshal([]byte(result.Candidates[0].Content.Parts[0].Text), &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling result: %v", err)
	}

	return &SummarizerResponse{
		Articles: response.Articles,
		AiModel:  aiModel,
	}, nil
}
