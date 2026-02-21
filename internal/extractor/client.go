package extractor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const anthropicAPI = "https://api.anthropic.com/v1/messages"

// Client calls the Anthropic Messages API.
type Client struct {
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

// NewClient creates a Client using the ANTHROPIC_API_KEY env var.
func NewClient(model string) (*Client, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}
	return &Client{
		APIKey:     key,
		Model:      model,
		HTTPClient: &http.Client{},
	}, nil
}

type apiRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system"`
	Messages  []apiMessage `json:"messages"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type apiResponse struct {
	Content []apiContentBlock `json:"content"`
	Usage   apiUsage          `json:"usage"`
	Error   *apiError         `json:"error,omitempty"`
}

type apiContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type apiUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Extract sends a chapter to Claude and returns the raw text response.
func (c *Client) Extract(ctx context.Context, chapterTitle, chapterText string) (string, apiUsage, error) {
	prompt := buildExtractionPrompt(chapterTitle, chapterText)

	reqBody := apiRequest{
		Model:     c.Model,
		MaxTokens: 64000,
		System:    systemPrompt,
		Messages: []apiMessage{
			{Role: "user", Content: prompt},
			{Role: "assistant", Content: "{"},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", apiUsage{}, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anthropicAPI, bytes.NewReader(body))
	if err != nil {
		return "", apiUsage{}, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", apiUsage{}, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", apiUsage{}, fmt.Errorf("reading response: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", apiUsage{}, fmt.Errorf("parsing response: %w", err)
	}

	if apiResp.Error != nil {
		return "", apiUsage{}, fmt.Errorf("API error (%s): %s", apiResp.Error.Type, apiResp.Error.Message)
	}

	if resp.StatusCode != 200 {
		return "", apiUsage{}, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	if len(apiResp.Content) == 0 {
		return "", apiUsage{}, fmt.Errorf("empty response from API")
	}

	// Prepend the "{" from the assistant prefill to reconstruct full JSON
	return "{" + apiResp.Content[0].Text, apiResp.Usage, nil
}
