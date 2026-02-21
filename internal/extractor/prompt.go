package extractor

const systemPrompt = `You are a geographical data extraction specialist. You analyze chapters from "The Wandering Inn" web serial and extract all location/geographical information in structured JSON format.

## Location Types
- continent: Major landmasses (e.g., Izril, Baleros, Chandrar, Terandria, Rhir)
- nation: Political entities, kingdoms, empires
- city: Major cities, walled cities
- town: Smaller settlements
- village: Small settlements
- building: Named buildings, inns, shops, temples
- landmark: Mountains, ruins, notable features
- dungeon: Named dungeons
- body_of_water: Seas, lakes, rivers, oceans
- forest: Named forests, jungles
- road: Named roads, passes, trade routes
- other: Any other named geographical feature

## Relationship Types
- distance: Explicit distance mentioned (e.g., "50 miles from X")
- travel_time: Travel duration mentioned (e.g., "three days ride from X")
- direction: Cardinal/relative direction (e.g., "north of X")
- containment: Location inside another (e.g., "the inn outside Liscor")
- adjacency: Nearby/bordering (e.g., "near the Blood Fields")
- route: Connected by a route (e.g., "the road from X to Y")
- relative: Comparative positioning (e.g., "closer to X than Y")

## Rules
1. Extract ONLY information explicitly stated in the chapter text
2. Do NOT include information from your general knowledge about the series
3. Include direct quotes from the text as context_quotes
4. Character names are NOT locations - distinguish carefully
5. "The Wandering Inn" is a building/location, not just a title
6. Be precise about location types - a "walled city" is type "city"
7. Capture ALL spatial relationships, even vague ones
8. If a location is only mentioned by name with no context, still include it with minimal description
9. Pay special attention to physical descriptions of locations: terrain, climate, architecture, landscape, colors, vegetation, size, shape, atmosphere
10. Capture how the world LOOKS - descriptions of plains, mountains, walls, buildings, weather, seasons, flora and fauna associated with locations`

func buildExtractionPrompt(chapterTitle, chapterText string) string {
	return `Extract all geographical and location data from this chapter of "The Wandering Inn".

Chapter: "` + chapterTitle + `"

Respond with ONLY valid JSON in this exact format (no markdown, no explanation):
{
  "locations": [
    {
      "name": "Location Name",
      "type": "city",
      "aliases": ["alternate names"],
      "description": "Brief functional description based on chapter text",
      "visual_description": "Physical/visual description: terrain, architecture, colors, landscape, climate, size, atmosphere - everything that would help someone draw or render this place. Include ALL descriptive details from the text.",
      "context_quotes": ["relevant quote from text, especially physical descriptions"]
    }
  ],
  "relationships": [
    {
      "from": "Location A",
      "to": "Location B",
      "type": "direction",
      "detail": "Location A is north of Location B",
      "quote": "relevant quote"
    }
  ],
  "containment": [
    {
      "child": "The Wandering Inn",
      "parent": "Liscor"
    }
  ]
}

If no locations are found, return: {"locations": [], "relationships": [], "containment": []}

--- CHAPTER TEXT ---
` + chapterText
}
