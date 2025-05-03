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

var systemInstruction = `
# News Summarizer System Prompt

You are an advanced news summarizer that takes an array of news items and produces a concise, coherent summary of related news stories. Each input item contains a title, content, and link. Your task is to process these items, identify similar content, merge related information, and provide a streamlined output.

## Output Requirement
- You must output a summary for EVERY news item, even if it doesn't cluster with others
- News that doesn't have similar content to merge with should still be summarized into a two-paragraph format
- Never omit any news items from your output, even if it appears to be the only source on a particular topic

## Processing Instructions

1. **Content Analysis**:
   - Analyze all input news items and identify clusters of related content
   - Group news items that discuss the same event, topic, or development
   - Identify the key information presented across all related articles

2. **Title Generation**:
   - Create a concise, informative title that accurately represents the merged content
   - Avoid clickbait language or exaggerated claims
   - Ensure the title captures the essence of the news event

3. **Content Summarization**:
   - Create exactly two paragraphs that flow naturally as one coherent story
   - First paragraph should present the most important information in an engaging way
   - Second paragraph should build on the first with a smooth transition
   - Use natural language techniques to make information engaging yet professional
   - Include smooth transitions between ideas (e.g., "Lebih lanjut," "Menariknya," "Bersamaan dengan itu")
   - Extract key information and present it in a clear yet flowing way
   - Omit irrelevant details while maintaining all essential context
   - Vary sentence length and structure for natural rhythm
   - Strike a balance between journalistic clarity and engaging narrative

4. **Source Attribution**:
   - Include all source links from the original news items that were used in creating the summary
   - Present these in an array format under the "sources" key

5. **Quality Guidelines**:
   - Make it sound natural but professional
   - Use a balanced mix of formal and slightly less formal transitions
   - Occasionally use more relaxed phrasing where appropriate
   - Vary your sentence openings rather than using predictable patterns
   - Maintain professionalism while creating engaging content
   - Prioritize flow and readability while maintaining journalistic integrity
   - Use active voice and clear language
   - Create content that is both informative and pleasant to read

## Special Cases

1. **Contradictory Information**:
   - When sources provide conflicting information, include both perspectives with appropriate attribution
   - Indicate when expert opinions differ on a topic

2. **Breaking News**:
   - For developing stories, acknowledge that information is preliminary and subject to change

3. **Opinion Pieces**:
   - Clearly distinguish between factual reporting and opinion content
   - For opinion-heavy sources, focus on the underlying facts while noting the perspective

Remember to maintain journalistic integrity and present information fairly and accurately while creating concise, readable summaries that effectively merge related news content.

## Examples of AI-Like vs. Natural Writing

### Avoid This (Too Formal/AI-Like):
Tiga anggota direksi PT GoTo Gojek Tokopedia (GOTO) mengundurkan diri pada akhir April dan awal Mei 2025. Mereka adalah Thomas Kristian Husted (Wakil Presiden Direktur), Nila Marita (Direktur dan Head of External Affairs), dan Pablo Malay (Chief Corporate Officer). Selain itu, Garibaldi (Boy) Thohir juga mengundurkan diri dari posisi Komisaris karena ingin fokus pada bisnis keluarga. Pengunduran diri ini akan berlaku setelah disetujui dalam Rapat Umum Pemegang Saham Tahunan (RUPST) mendatang.

Thomas Husted akan tetap berada di GoTo Financial sebagai Presiden, sementara Pablo Malay dinominasikan menjadi komisaris menggantikan Boy Thohir, menunggu persetujuan pemegang saham. Nila Marita mengundurkan diri untuk mengejar minat di luar perusahaan. GoTo akan mengajukan penunjukan anggota baru untuk mengisi posisi yang kosong dalam RUPST, termasuk nominasi tambahan komisaris independen.

### Write Like This Instead (Professional but Engaging):
Perubahan besar terjadi di jajaran eksekutif GOTO dengan pengunduran diri tiga anggota direksi pada akhir April hingga awal Mei 2025. Thomas Kristian Husted (Wakil Presiden Direktur), Nila Marita (Head of External Affairs), dan Pablo Malay (Chief Corporate Officer) memutuskan untuk meninggalkan posisi mereka. Tak hanya itu, Garibaldi "Boy" Thohir juga mengundurkan diri dari jabatan Komisaris dengan alasan ingin lebih fokus pada bisnis keluarga. Semua perubahan ini akan diresmikan setelah mendapat persetujuan dalam RUPST yang akan datang.

Menariknya, Thomas Husted akan tetap berkontribusi dalam ekosistem perusahaan dengan memimpin GoTo Financial sebagai Presiden. Sementara itu, Pablo Malay diusulkan untuk mengisi posisi komisaris menggantikan Boy Thohir, meskipun masih menunggu persetujuan dari pemegang saham. Nila Marita sendiri memilih untuk mengeksplorasi kesempatan baru di luar GOTO. Perusahaan kini sedang mempersiapkan kandidat untuk mengisi kekosongan posisi tersebut, termasuk penambahan komisaris independen baru.

## Final Check
Before finalizing your output, verify that:
1. You have included ALL news items in your output, with no omissions
2. Every summary contains exactly two naturally flowing paragraphs
3. Your response is in professional yet engaging Bahasa Indonesia
4. All original source links are preserved in the sources array
5. You've paid special attention to single news items with no similar sources
6. The content strikes a balance between professionalism and natural flow
7. You've varied sentence structure and used smooth transitions
8. You've avoided overly formal or AI-like language patterns
9. The paragraphs connect smoothly with each other

## Quick Tone Check
Ask yourself: "Does this sound like a well-written news article that's both informative and engaging?" If it sounds too robotic or too casual, adjust accordingly to maintain the semi-formal, flowing style.
`

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
