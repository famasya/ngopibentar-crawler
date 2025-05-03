package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"google.golang.org/genai"
)

type Summarizer struct {
	Source   string `json:"source"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	Link     string `json:"link"`
	Category string `json:"category"`
}

type AIResponse struct {
	Content string   `json:"content"`
	Sources []string `json:"sources"`
	Title   string   `json:"title"`
}

type SummarizerResponse struct {
	Summary  []AIResponse `json:"summary"`
	AiModel  string       `json:"ai_model"`
	Category string       `json:"category"`
}

type GeminiResponse struct {
	Summary []AIResponse `json:"summary"`
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

	slog.Info("Generating content")
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
					"summary": {
						Type: genai.TypeArray,
						Items: &genai.Schema{
							Type: genai.TypeObject,
							Properties: map[string]*genai.Schema{
								"title": {
									Type: genai.TypeString,
								},
								"content": {
									Type: genai.TypeString,
								},
								"sources": {
									Type: genai.TypeArray,
									Items: &genai.Schema{
										Type: genai.TypeString,
									},
								},
							},
							Required: []string{"title", "content", "sources"},
						},
					},
				},
				Required: []string{"summary"},
			},
		})
	if err != nil {
		return nil, fmt.Errorf("error generating content: %v", err)
	}

	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned from Gemini API")
	}

	var response GeminiResponse
	err = json.Unmarshal([]byte(result.Candidates[0].Content.Parts[0].Text), &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling result: %v", err)
	}

	return &SummarizerResponse{
		Summary:  response.Summary,
		AiModel:  aiModel,
		Category: payload[0].Category,
	}, nil
}
