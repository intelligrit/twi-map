package aggregator

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/robertmeta/twi-map/internal/model"
	"github.com/robertmeta/twi-map/internal/store"
)

const (
	// minMentions is the minimum number of chapter mentions required for a location to be included.
	minMentions = 3
	// maxContainmentDepth is how many levels of parent containment to walk when checking traceability.
	maxContainmentDepth = 10
)

// Aggregate loads all per-chapter extractions and merges them into a unified dataset.
func Aggregate(s *store.Store) (*model.AggregatedData, error) {
	toc, err := s.ReadTOC()
	if err != nil {
		return nil, fmt.Errorf("reading TOC: %w", err)
	}

	type locEntry struct {
		loc     model.AggregatedLocation
		indices map[int]bool
	}

	// Canonical name mapping for well-known locations with many variants
	canonicalNames := map[string]string{
		"the inn":                 "the wandering inn",
		"inn":                     "the wandering inn",
		"the wandering inn":       "the wandering inn",
		"bloodfields":             "blood fields",
		"the blood fields":        "blood fields",
		"the bloodfields":         "blood fields",
		"high passes":             "the high passes",
		"the high passes":         "the high passes",
		"floodplains":             "floodplains of liscor",
		"the floodplains":         "floodplains of liscor",
		"flood plains":            "floodplains of liscor",
		"antinium hive":           "antinium hive",
		"the antinium hive":       "antinium hive",
		"the hive":                "antinium hive",
		"hive":                    "antinium hive",
		"drath archipelago":       "drath",
		"the ruins":               "ruins of albez",
		"ruins":                   "ruins of albez",
		"garden of sanctuary":     "garden of sanctuary",
		"the garden of sanctuary": "garden of sanctuary",
		"the garden":              "garden of sanctuary",
		"great plains of izril":   "great plains",
		"the great plains":        "great plains",
		"liscor's dungeon":        "liscor's dungeon",
		"dungeon":                 "liscor's dungeon",
		"the dungeon":             "liscor's dungeon",
		"new lands of izril":      "new lands",
	}

	// Earth locations to exclude — TWI characters are transported from modern Earth
	// to Innworld, so real-world place names appear in dialogue but aren't map locations.
	earthLocations := map[string]bool{
		"earth": true, "new york": true, "michigan": true, "london": true,
		"california": true, "oakland": true, "america": true, "japan": true,
		"china": true, "korea": true, "india": true, "france": true,
		"germany": true, "england": true, "united states": true,
		"los angeles": true, "san francisco": true, "chicago": true,
		"tokyo": true, "paris": true, "rome": true, "boston": true,
		"seattle": true, "texas": true, "florida": true, "ohio": true,
		"colorado": true, "europe": true, "asia": true, "africa": true,
		"south america": true, "north america": true, "australia": true,
		"canada": true, "mexico": true, "russia": true, "brazil": true,
		"spain": true, "italy": true, "greece": true,
	}

	// Simple map: normalized name -> entry. No alias pointer tricks.
	locMap := make(map[string]*locEntry)

	var allRels []model.AggregatedRelationship
	relSeen := make(map[string]bool)

	var allContainment []model.Containment
	contSeen := make(map[string]bool)

	for _, ch := range toc.Chapters {
		if !s.ExtractionExists(ch.Index) {
			continue
		}

		ext, err := s.ReadExtraction(ch.Index)
		if err != nil {
			continue
		}

		for _, loc := range ext.Locations {
			key := normalizeName(loc.Name)
			if earthLocations[key] {
				continue // skip Earth locations
			}
			if canon, ok := canonicalNames[key]; ok {
				key = canon
			}

			if entry, ok := locMap[key]; ok {
				entry.indices[ch.Index] = true
				entry.loc.MentionCount++
				if len(loc.Description) > len(entry.loc.Description) {
					entry.loc.Description = loc.Description
				}
				if len(loc.VisualDescription) > len(entry.loc.VisualDescription) {
					entry.loc.VisualDescription = loc.VisualDescription
				}
				for _, a := range loc.Aliases {
					if !containsNorm(entry.loc.Aliases, a) {
						entry.loc.Aliases = append(entry.loc.Aliases, a)
					}
				}
			} else {
				locMap[key] = &locEntry{
					loc: model.AggregatedLocation{
						ID:                key,
						Name:              toDisplayName(key),
						Type:              loc.Type,
						Aliases:           loc.Aliases,
						Description:       loc.Description,
						VisualDescription: loc.VisualDescription,
						FirstChapterIndex: ch.Index,
						MentionCount:      1,
					},
					indices: map[int]bool{ch.Index: true},
				}
			}
		}

		for _, rel := range ext.Relationships {
			fromKey := canonicalize(normalizeName(rel.From), canonicalNames)
			toKey := canonicalize(normalizeName(rel.To), canonicalNames)
			rKey := fmt.Sprintf("%s|%s|%s", fromKey, toKey, rel.Type)
			if !relSeen[rKey] {
				relSeen[rKey] = true
				allRels = append(allRels, model.AggregatedRelationship{
					From:              toDisplayName(fromKey),
					To:                toDisplayName(toKey),
					Type:              rel.Type,
					Detail:            rel.Detail,
					FirstChapterIndex: ch.Index,
				})
			}
		}

		for _, c := range ext.Containment {
			childKey := canonicalize(normalizeName(c.Child), canonicalNames)
			parentKey := canonicalize(normalizeName(c.Parent), canonicalNames)
			cKey := childKey + "|" + parentKey
			if !contSeen[cKey] {
				contSeen[cKey] = true
				allContainment = append(allContainment, model.Containment{
					Child:  toDisplayName(childKey),
					Parent: toDisplayName(parentKey),
				})
			}
		}
	}

	// Build containment parent lookup for traceability check
	parentOf := make(map[string]string)
	for _, c := range allContainment {
		parentOf[normalizeName(c.Child)] = normalizeName(c.Parent)
	}

	// Seed names that count as "traceable" (locations we have known positions for)
	seededNames := map[string]bool{
		"izril": true, "baleros": true, "chandrar": true, "terandria": true,
		"rhir": true, "drath archipelago": true, "drath": true,
		"liscor": true, "the wandering inn": true, "celum": true,
		"esthelm": true, "wales": true, "invrisil": true, "pallass": true,
		"the blood fields": true, "the high passes": true, "high passes": true,
		"the floodplains": true, "flood plains": true, "floodplains of liscor": true,
		"first landing": true, "the northern plains": true, "the human lands": true,
		"the drake lands": true, "great plains of izril": true, "vale forest": true,
		"blood fields": true, "bloodfields": true, "ruins of liscor": true,
		"ruins of albez": true, "krakk forest": true,
		"reim": true, "hellios": true, "germina": true, "nerrhavia": true,
		"nerrhavia's fallen": true, "belchan": true, "jecrass": true,
		"medain": true, "khelt": true, "quarass": true,
		"riverfarm": true, "magnolia's estate": true, "lady magnolia's estate": true,
		"calanfer": true, "noelictus": true, "ailendamus": true,
		"oteslia": true, "zeres": true, "manus": true, "reizmelt": true,
		"hectval": true, "wistram academy": true, "wistram": true,
		"tiqr": true, "pomle": true, "roshal": true, "savere": true,
		"talenqual": true, "elvallian": true, "gaiil-drome": true,
		"blighted kingdom": true, "pheislant": true, "desonis": true,
		"kaliv": true, "erribathe": true, "dawn concordat": true,
		"house of minos": true, "new lands": true, "great plains": true,
		"garden of sanctuary": true, "liscor's dungeon": true,
		"a'ctelios salash": true, "zeikhal": true, "paeth": true,
		"claiven earth": true, "az'kerash's castle": true,
		"remendia": true, "albez": true, "runner's guild": true,
		"windrest": true, "unseen empire": true, "laken's empire": true,
		"tails and scales": true, "nombernaught": true,
		"salazsar": true, "fissival": true, "drake lands": true,
		"human lands": true, "gnoll plains": true,
		"kasignel": true, "shifthold": true,
		"walled cities": true, "market street": true,
		"adventurer's guild": true, "hivelands": true,
	}

	// Core place keywords - if a containment chain mentions one of these, it's traceable
	seedKeywords := []string{
		"izril", "baleros", "chandrar", "terandria", "rhir", "drath",
		"liscor", "celum", "esthelm", "invrisil", "pallass", "wales",
		"reim", "riverfarm", "magnolia", "calanfer", "ailendamus",
		"oteslia", "zeres", "manus", "wistram", "talenqual", "khelt",
		"noelictus", "pheislant", "hectval", "reizmelt",
		"remendia", "albez", "pomle", "tiqr", "roshal", "savere",
		"jecrass", "hellios", "germina", "medain", "belchan",
		"gaiil-drome", "elvallian", "paeth", "claiven",
		"blighted", "nerrhavia", "desonis", "kaliv", "erribathe",
		"laken", "unseen empire", "riverfarm",
		"nombernaught", "dwarven", "salazsar", "fissival", "drake",
		"human", "gnoll", "antinium", "goblin",
	}

	matchesSeed := func(name string) bool {
		if seededNames[name] {
			return true
		}
		for _, kw := range seedKeywords {
			if strings.Contains(name, kw) {
				return true
			}
		}
		return false
	}

	// Check if a location can trace back to a known position
	isTraceable := func(id string) bool {
		if matchesSeed(id) {
			return true
		}
		cur := id
		for i := 0; i < maxContainmentDepth; i++ {
			p, ok := parentOf[cur]
			if !ok {
				return false
			}
			if matchesSeed(p) {
				return true
			}
			cur = p
		}
		return false
	}

	var locations []model.AggregatedLocation
	for _, entry := range locMap {
		if entry.loc.MentionCount < minMentions {
			continue // not referenced enough
		}
		if !isTraceable(entry.loc.ID) {
			continue // can't place on map
		}
		for idx := range entry.indices {
			entry.loc.ChapterIndices = append(entry.loc.ChapterIndices, idx)
		}
		sort.Ints(entry.loc.ChapterIndices)
		locations = append(locations, entry.loc)
	}

	// Sort by first appearance
	sort.Slice(locations, func(i, j int) bool {
		return locations[i].FirstChapterIndex < locations[j].FirstChapterIndex
	})

	return &model.AggregatedData{
		Locations:     locations,
		Relationships: allRels,
		Containment:   allContainment,
		AggregatedAt:  time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// canonicalize applies the canonical name map, returning the canonical form or the original.
func canonicalize(name string, canonicalNames map[string]string) string {
	if canon, ok := canonicalNames[name]; ok {
		return canon
	}
	return name
}

func normalizeName(name string) string {
	// Strip square brackets — LLM sometimes wraps location names in them
	name = strings.NewReplacer("[", "", "]", "").Replace(name)
	return strings.ToLower(strings.TrimSpace(name))
}

// toDisplayName converts a normalized (lowercase) name to title case for display.
func toDisplayName(name string) string {
	words := strings.Fields(name)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func containsNorm(slice []string, s string) bool {
	norm := normalizeName(s)
	for _, item := range slice {
		if normalizeName(item) == norm {
			return true
		}
	}
	return false
}
