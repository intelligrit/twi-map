# TWI Map - Research Notes

## The Wandering Inn - World Structure

### Known Continents (from general knowledge)

- **Izril** - Main setting. Split north/south. The Wandering Inn is here.
- **Baleros** - Jungle continent, Companies (mercenary groups)
- **Chandrar** - Desert continent, kingdoms and empires
- **Terandria** - "Europe-like" continent, kingdoms and knightly orders
- **Rhir** - Demon-held continent, Blighted Kingdom
- **Drath Archipelago** - Far east islands, rarely visited in story

### Key Location Types to Extract

1. **Continents** - Major landmasses
2. **Nations/Kingdoms** - Political entities with borders
3. **Cities/Towns/Villages** - Named settlements
4. **Landmarks** - Mountains, rivers, forests, dungeons, ruins
5. **Roads/Passes** - Named travel routes
6. **Bodies of Water** - Seas, lakes, rivers
7. **Special Locations** - The Inn itself, Walled Cities, Great Companies HQs
8. **Dungeons** - Named dungeons and their locations

### Relationship Types Between Locations

1. **Distance** - "X miles/leagues from Y"
2. **Travel Time** - "X days travel from Y"
3. **Directional** - "North of", "East of", etc.
4. **Containment** - "City X is in Kingdom Y which is on Continent Z"
5. **Adjacency** - "borders", "near", "next to"
6. **Route** - "The road from X passes through Y"
7. **Relative** - "closer to X than Y", "halfway between"

### Walled Cities of Izril (known major landmarks)

- Liscor - Near The Wandering Inn
- Pallass - Connected via magic door
- Invrisil - Connected via magic door
- Oteslia - Tree city, south
- Zeres - City of Waves
- Manus - War city
- Salazsar - Gem city
- Fissival - City of Magic

### Chapter Content Characteristics

- Chapters are LONG (10,000-40,000+ words each)
- 807 chapters = potentially millions of words
- Location info is scattered throughout narrative prose
- Some chapters are location-heavy (travel chapters, war chapters)
- Some chapters barely mention locations (dialogue-heavy character chapters)
- Author occasionally provides explicit geography (e.g., map descriptions in text)

## Extraction Strategy Notes

### Challenges

1. **Scale**: ~12+ million words total. Can't process all at once.
2. **Ambiguity**: "south" might be relative to character, not absolute
3. **Evolving geography**: Some places change (destroyed, rebuilt, renamed)
4. **Spoiler sensitivity**: Must track when info is first revealed
5. **Character perspective**: Same location described differently by different POVs
6. **Canonical vs fan content**: Must only use chapter prose, not comments/images

### Approach

- Process chapter by chapter, sequentially
- Use LLM to extract structured location data from each chapter's plaintext
- Store with chapter metadata (volume, book, audiobook chapter, ebook chapter)
- Build cumulative graph of locations and relationships
- Allow "as of chapter X" queries for spoiler-free maps
