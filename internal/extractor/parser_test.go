package extractor

import (
	"testing"
)

func TestParseExtraction_Direct(t *testing.T) {
	input := `{"locations":[{"name":"Liscor","type":"city","aliases":[],"description":"A walled city","context_quotes":["near the inn"]}],"relationships":[],"containment":[]}`

	result, err := ParseExtraction(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(result.Locations))
	}
	if result.Locations[0].Name != "Liscor" {
		t.Errorf("expected name Liscor, got %s", result.Locations[0].Name)
	}
}

func TestParseExtraction_WithPreamble(t *testing.T) {
	input := `Here is the extraction:
{
  "locations": [
    {"name": "The Wandering Inn", "type": "building", "description": "An inn"}
  ],
  "relationships": [],
  "containment": []
}
Some trailing text.`

	result, err := ParseExtraction(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(result.Locations))
	}
}

func TestParseExtraction_CodeBlock(t *testing.T) {
	input := "```json\n{\"locations\":[],\"relationships\":[],\"containment\":[]}\n```"

	result, err := ParseExtraction(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Locations) != 0 {
		t.Fatalf("expected 0 locations, got %d", len(result.Locations))
	}
}

func TestParseExtraction_Empty(t *testing.T) {
	input := `{"locations":[],"relationships":[],"containment":[]}`

	result, err := ParseExtraction(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Locations) != 0 {
		t.Fatalf("expected 0 locations, got %d", len(result.Locations))
	}
}

func TestParseExtraction_Invalid(t *testing.T) {
	_, err := ParseExtraction("not json at all")
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func TestParseExtraction_WithRelationships(t *testing.T) {
	input := `{
  "locations": [
    {"name": "Izril", "type": "continent", "description": "Main continent"},
    {"name": "Liscor", "type": "city", "description": "Walled city"}
  ],
  "relationships": [
    {"from": "Liscor", "to": "Izril", "type": "containment", "detail": "Liscor is on Izril"}
  ],
  "containment": [
    {"child": "Liscor", "parent": "Izril"}
  ]
}`

	result, err := ParseExtraction(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Locations) != 2 {
		t.Fatalf("expected 2 locations, got %d", len(result.Locations))
	}
	if len(result.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(result.Relationships))
	}
	if len(result.Containment) != 1 {
		t.Fatalf("expected 1 containment, got %d", len(result.Containment))
	}
}
