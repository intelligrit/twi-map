package aggregator

import (
	"github.com/robertmeta/twi-map/internal/model"
	"github.com/robertmeta/twi-map/internal/store"
)

// AssignCoordinates generates initial coordinates for locations that don't have any.
// Uses containment relationships and directional data to estimate positions.
// Manual coordinates (from the DB) are never overwritten.
func AssignCoordinates(s *store.Store, data *model.AggregatedData) error {
	existing, err := s.ReadCoordinates()
	if err != nil {
		existing = nil // fresh start
	}

	coordMap := make(map[string]model.Coordinate)
	for _, c := range existing {
		if c.Manual {
			coordMap[c.LocationID] = c // preserve manual coords
		}
	}

	// Build containment tree
	parentOf := make(map[string]string)
	for _, c := range data.Containment {
		parentOf[normalizeName(c.Child)] = normalizeName(c.Parent)
	}

	// Seed continents matching positions on the base map image ([-512,512] coordinate space)
	// Image is 1024x1024 mapped to [-512,512]. Coords are [x_horizontal, y_vertical].
	seeds := map[string][2]float64{
		"izril":             {200, -20},     // large continent center-right
		"baleros":           {-220, -300},   // lower-left
		"chandrar":          {-80, -120},    // center-left desert
		"terandria":         {-250, 250},    // upper-left
		"rhir":              {350, 300},     // upper-right
		"drath archipelago": {420, 50},      // far right islands
		"drath":             {420, 50},
	}

	// Seed known regions/nations within Izril (southern Izril where V1 action happens)
	izrilSeeds := map[string][2]float64{
		"liscor":                {190, -40},
		"the wandering inn":     {191, -38},
		"the inn":               {191, -38},
		"inn":                   {191, -38},
		"celum":                 {150, 40},
		"esthelm":               {130, 30},
		"wales":                 {120, 50},
		"invrisil":              {140, 80},
		"pallass":               {230, -70},
		"the blood fields":      {180, -60},
		"the high passes":       {250, 20},
		"the floodplains":       {188, -43},
		"flood plains":          {188, -43},
		"floodplains of liscor": {188, -43},
		"first landing":         {110, 110},
		"the northern plains":   {150, 90},
		"the human lands":       {140, 70},
		"the drake lands":       {200, -50},
		"great plains of izril": {180, -90},
		"high passes":           {250, 20},
		"vale forest":           {160, 50},
		"blood fields":          {180, -60},
		"bloodfields":           {180, -60},
		"ruins of liscor":       {192, -41},
		"ruins of albez":        {160, 10},
		"krakk forest":          {170, 30},
	}

	// Chandrar nations (center-left, below Terandria)
	chandrarSeeds := map[string][2]float64{
		"reim":                 {-50, -100},
		"hellios":              {-60, -120},
		"germina":              {-100, -140},
		"nerrhavia":            {-110, -100},
		"nerrhavia's fallen":   {-110, -100},
		"belchan":              {-70, -80},
		"jecrass":              {-40, -130},
		"medain":               {-100, -80},
		"khelt":                {-30, -70},
		"quarass":              {-80, -130},
	}
	for name, pos := range chandrarSeeds {
		izrilSeeds[name] = pos
	}

	for name, pos := range seeds {
		if _, ok := coordMap[name]; !ok {
			coordMap[name] = model.Coordinate{
				LocationID: name, X: pos[0], Y: pos[1],
				Confidence: "estimated",
			}
		}
	}
	for name, pos := range izrilSeeds {
		if _, ok := coordMap[name]; !ok {
			coordMap[name] = model.Coordinate{
				LocationID: name, X: pos[0], Y: pos[1],
				Confidence: "estimated",
			}
		}
	}

	// Place locations with containment parents near their parent
	for _, loc := range data.Locations {
		id := normalizeName(loc.Name)
		if _, ok := coordMap[id]; ok {
			continue
		}

		parent := parentOf[id]
		for parent != "" {
			if pc, ok := coordMap[parent]; ok {
				spread := spreadForType(loc.Type)
				coord := model.Coordinate{
					LocationID: id,
					X:          pc.X + hashFloat(id, "x", spread),
					Y:          pc.Y + hashFloat(id, "y", spread),
					Confidence: "estimated",
				}
				coordMap[id] = coord
				break
			}
			parent = parentOf[parent]
		}
	}

	// Place remaining locations without containment:
	// Group by type and place around a default region
	// Default placement near Liscor/Izril for unplaced locations (most V1 action is here)
	typeDefaults := map[model.LocationType][2]float64{
		model.LocationContinent:   {0, 0},
		model.LocationNation:      {0, 0},
		model.LocationCity:        {80, 0},
		model.LocationTown:        {60, 30},
		model.LocationVillage:     {50, 40},
		model.LocationBuilding:    {92, -28}, // near Liscor
		model.LocationLandmark:    {100, -10},
		model.LocationDungeon:     {110, -20},
		model.LocationBodyOfWater: {70, -30},
		model.LocationForest:      {60, 50},
		model.LocationRoad:        {70, 20},
		model.LocationOther:       {80, 10},
	}

	for _, loc := range data.Locations {
		id := normalizeName(loc.Name)
		if _, ok := coordMap[id]; ok {
			continue
		}

		base, ok := typeDefaults[loc.Type]
		if !ok {
			base = [2]float64{0, 0}
		}

		spread := spreadForType(loc.Type)
		coordMap[id] = model.Coordinate{
			LocationID: id,
			X:          base[0] + hashFloat(id, "x", spread),
			Y:          base[1] + hashFloat(id, "y", spread),
			Confidence: "estimated",
		}
	}

	// Write all non-manual coordinates
	for _, c := range coordMap {
		if err := s.WriteCoordinate(c); err != nil {
			return err
		}
	}

	return nil
}

func spreadForType(t model.LocationType) float64 {
	switch t {
	case model.LocationContinent:
		return 50
	case model.LocationNation:
		return 40
	case model.LocationCity:
		return 25
	case model.LocationTown, model.LocationVillage:
		return 20
	case model.LocationBuilding:
		return 5
	case model.LocationLandmark:
		return 8
	default:
		return 20
	}
}

func hashFloat(name, axis string, spread float64) float64 {
	h := simpleHash(name + ":" + axis)
	// Map to [-spread, +spread]
	return (float64(h%1000)/500.0 - 1.0) * spread
}

func simpleHash(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}
