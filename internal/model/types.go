package model

// Chapter represents a single chapter from the TOC.
type Chapter struct {
	WebTitle         string `json:"web_title"`
	URL              string `json:"url"`
	Volume           string `json:"volume"`
	BookNumber       int    `json:"book_number"`
	AudiobookChapter string `json:"audiobook_chapter"`
	EbookChapter     string `json:"ebook_chapter"`
	Slug             string `json:"slug"`
	Index            int    `json:"index"`
}

// TOC is the full table of contents.
type TOC struct {
	Chapters  []Chapter `json:"chapters"`
	ScrapedAt string    `json:"scraped_at"`
}

// LocationType classifies extracted locations.
type LocationType string

const (
	LocationContinent   LocationType = "continent"
	LocationNation      LocationType = "nation"
	LocationCity        LocationType = "city"
	LocationTown        LocationType = "town"
	LocationVillage     LocationType = "village"
	LocationBuilding    LocationType = "building"
	LocationLandmark    LocationType = "landmark"
	LocationDungeon     LocationType = "dungeon"
	LocationBodyOfWater LocationType = "body_of_water"
	LocationForest      LocationType = "forest"
	LocationRoad        LocationType = "road"
	LocationOther       LocationType = "other"
)

// RelationshipType classifies spatial relationships between locations.
type RelationshipType string

const (
	RelDistance    RelationshipType = "distance"
	RelTravelTime  RelationshipType = "travel_time"
	RelDirection   RelationshipType = "direction"
	RelContainment RelationshipType = "containment"
	RelAdjacency   RelationshipType = "adjacency"
	RelRoute       RelationshipType = "route"
	RelRelative    RelationshipType = "relative"
)

// ExtractedLocation is a location found in a single chapter.
type ExtractedLocation struct {
	Name              string       `json:"name"`
	Type              LocationType `json:"type"`
	Aliases           []string     `json:"aliases,omitempty"`
	Description       string       `json:"description"`
	VisualDescription string       `json:"visual_description,omitempty"`
	ContextQuotes     []string     `json:"context_quotes,omitempty"`
}

// ExtractedRelationship is a spatial relationship found in a single chapter.
type ExtractedRelationship struct {
	From   string           `json:"from"`
	To     string           `json:"to"`
	Type   RelationshipType `json:"type"`
	Detail string           `json:"detail"`
	Quote  string           `json:"quote,omitempty"`
}

// Containment represents a parent-child containment relationship.
type Containment struct {
	Child  string `json:"child"`
	Parent string `json:"parent"`
}

// ChapterExtraction is the full extraction result for one chapter.
type ChapterExtraction struct {
	ChapterIndex  int                     `json:"chapter_index"`
	ChapterTitle  string                  `json:"chapter_title"`
	Locations     []ExtractedLocation     `json:"locations"`
	Relationships []ExtractedRelationship `json:"relationships"`
	Containment   []Containment           `json:"containment"`
	Model         string                  `json:"model"`
	ExtractedAt   string                  `json:"extracted_at"`
}

// AggregatedLocation is a deduplicated location with cross-chapter data.
type AggregatedLocation struct {
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	Type              LocationType `json:"type"`
	Aliases           []string     `json:"aliases,omitempty"`
	Description       string       `json:"description"`
	VisualDescription string       `json:"visual_description,omitempty"`
	FirstChapterIndex int          `json:"first_chapter_index"`
	MentionCount      int          `json:"mention_count"`
	ChapterIndices    []int        `json:"chapter_indices"`
}

// AggregatedRelationship is a deduplicated relationship.
type AggregatedRelationship struct {
	From              string           `json:"from"`
	To                string           `json:"to"`
	Type              RelationshipType `json:"type"`
	Detail            string           `json:"detail"`
	FirstChapterIndex int              `json:"first_chapter_index"`
}

// AggregatedData is the full aggregated dataset.
type AggregatedData struct {
	Locations     []AggregatedLocation     `json:"locations"`
	Relationships []AggregatedRelationship `json:"relationships"`
	Containment   []Containment            `json:"containment"`
	AggregatedAt  string                   `json:"aggregated_at"`
}

// Coordinate holds map coordinates for a location.
type Coordinate struct {
	LocationID string  `json:"location_id"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Confidence string  `json:"confidence"` // "high", "medium", "low", "estimated"
	Manual     bool    `json:"manual"`
}

// CoordinateData is the full coordinate file.
type CoordinateData struct {
	Coordinates []Coordinate `json:"coordinates"`
	UpdatedAt   string       `json:"updated_at"`
}
