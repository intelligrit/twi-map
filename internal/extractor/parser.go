package extractor

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/intelligrit/twi-map/internal/model"
)

// extractionResponse mirrors the JSON structure returned by the LLM.
type extractionResponse struct {
	Locations     []model.ExtractedLocation     `json:"locations"`
	Relationships []model.ExtractedRelationship `json:"relationships"`
	Containment   []model.Containment           `json:"containment"`
}

// ParseExtraction attempts to parse the LLM response text as JSON.
// Tries multiple strategies: direct parse, brace extraction, code block extraction.
func ParseExtraction(text string) (*extractionResponse, error) {
	text = strings.TrimSpace(text)

	// Strategy 1: direct parse
	var result extractionResponse
	if err := json.Unmarshal([]byte(text), &result); err == nil {
		return &result, nil
	}

	// Strategy 2: extract from first { to last }
	if start := strings.Index(text, "{"); start >= 0 {
		if end := strings.LastIndex(text, "}"); end > start {
			jsonStr := text[start : end+1]
			if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
				return &result, nil
			}
		}
	}

	// Strategy 3: extract from code blocks
	if idx := strings.Index(text, "```json"); idx >= 0 {
		after := text[idx+7:]
		if end := strings.Index(after, "```"); end >= 0 {
			jsonStr := strings.TrimSpace(after[:end])
			if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
				return &result, nil
			}
		}
	}
	if idx := strings.Index(text, "```"); idx >= 0 {
		after := text[idx+3:]
		if end := strings.Index(after, "```"); end >= 0 {
			jsonStr := strings.TrimSpace(after[:end])
			if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
				return &result, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to parse extraction response as JSON: %.200s...", text)
}
