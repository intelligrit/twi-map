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

	// Seed continents with wide separation so landmasses don't overlap.
	// Coordinate space is [-512,512]. Each continent gets its own well-separated region.
	// Layout (approximate):
	//   Terandria (upper-left)     Rhir (upper-right)
	//        Wistram (center)    Izril (center-right)
	//   Chandrar (lower-left)      Drath (far right)
	//        Baleros (bottom)
	seeds := map[string][2]float64{
		"izril":             {250, 0},       // center-right, large continent
		"chandrar":          {-250, -100},   // lower-left, desert continent
		"terandria":         {-250, 300},    // upper-left
		"baleros":           {-150, -400},   // bottom-center
		"rhir":              {350, 350},     // upper-right
		"drath archipelago": {480, 0},       // far right islands
		"drath":             {480, 0},
	}

	// Izril locations (center-right region, ~100-350 x, ~-120 to 150 y)
	izrilSeeds := map[string][2]float64{
		"liscor":                {240, -20},
		"the wandering inn":     {241, -18},
		"the inn":               {241, -18},
		"inn":                   {241, -18},
		"celum":                 {200, 50},
		"esthelm":               {180, 40},
		"wales":                 {170, 60},
		"invrisil":              {190, 90},
		"pallass":               {280, -50},
		"the blood fields":      {230, -40},
		"the high passes":       {300, 30},
		"the floodplains":       {238, -23},
		"flood plains":          {238, -23},
		"floodplains of liscor": {238, -23},
		"first landing":         {160, 120},
		"the northern plains":   {200, 100},
		"the human lands":       {190, 80},
		"the drake lands":       {250, -30},
		"great plains of izril": {230, -70},
		"high passes":           {300, 30},
		"vale forest":           {210, 60},
		"blood fields":          {230, -40},
		"bloodfields":           {230, -40},
		"ruins of liscor":       {242, -21},
		"ruins of albez":        {210, 20},
		"krakk forest":          {220, 40},
	}

	// Chandrar nations (lower-left, ~-350 to -150 x, ~-200 to 0 y)
	chandrarSeeds := map[string][2]float64{
		"reim":                 {-220, -80},
		"hellios":              {-230, -100},
		"germina":              {-270, -120},
		"nerrhavia":            {-280, -80},
		"nerrhavia's fallen":   {-280, -80},
		"belchan":              {-240, -60},
		"jecrass":              {-210, -110},
		"medain":               {-270, -60},
		"khelt":                {-200, -50},
		"quarass":              {-250, -110},
		"tiqr":                 {-300, -110},
		"pomle":                {-230, -130},
		"roshal":               {-310, -90},
		"savere":               {-260, -140},
		"a'ctelios salash":     {-290, -70},
		"zeikhal":              {-240, -90},
	}
	for name, pos := range chandrarSeeds {
		izrilSeeds[name] = pos
	}

	// Terandria nations (upper-left, ~-350 to -150 x, ~200 to 400 y)
	terandriaSeeds := map[string][2]float64{
		"ailendamus":       {-280, 330},
		"calanfer":         {-230, 310},
		"pheislant":        {-260, 280},
		"noelictus":        {-220, 350},
		"dawn concordat":   {-240, 300},
		"desonis":          {-270, 310},
		"kaliv":            {-250, 320},
		"erribathe":        {-210, 320},
	}
	for name, pos := range terandriaSeeds {
		izrilSeeds[name] = pos
	}

	// Baleros locations (bottom, ~-250 to -50 x, ~-500 to -300 y)
	balerosSeeds := map[string][2]float64{
		"talenqual":        {-120, -380},
		"elvallian":        {-170, -420},
		"gaiil-drome":      {-190, -390},
		"claiven earth":    {-160, -360},
		"paeth":            {-140, -410},
	}
	for name, pos := range balerosSeeds {
		izrilSeeds[name] = pos
	}

	// Rhir (upper-right, ~280-420 x, ~280-420 y)
	rhirSeeds := map[string][2]float64{
		"blighted kingdom": {360, 340},
	}
	for name, pos := range rhirSeeds {
		izrilSeeds[name] = pos
	}

	// More Izril cities (within Izril's region)
	moreIzril := map[string][2]float64{
		"oteslia":              {270, -60},
		"zeres":                {290, -40},
		"manus":                {310, -20},
		"salazsar":             {280, -30},
		"fissival":             {300, -50},
		"reizmelt":             {185, 70},
		"hectval":              {250, -35},
		"riverfarm":            {150, 80},
		"windrest":             {155, 75},
		"wistram academy":      {-30, 150},    // island between continents
		"wistram":              {-30, 150},
		"az'kerash's castle":   {220, 10},
		"garden of sanctuary":  {241, -17},
		"liscor's dungeon":     {242, -22},
		"new lands":            {200, -100},
		"house of minos":       {400, -150},   // island nation far east
		"great plains":         {230, -70},
		"remendia":             {195, 30},
		"albez":                {210, 20},
		"unseen empire":        {145, 85},
		"laken's empire":       {145, 85},
		"nombernaught":         {350, -150},   // undersea city
		"kasignel":             {0, 480},      // land of the dead (far above)
		"shifthold":            {210, -10},
	}
	for name, pos := range moreIzril {
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
	// Default to Izril region since most story action is there.
	// Centered around (220, 0) which is mid-Izril, avoiding overlap with other continents.
	typeDefaults := map[model.LocationType][2]float64{
		model.LocationContinent:   {220, 0},
		model.LocationNation:      {220, 10},
		model.LocationCity:        {210, -10},
		model.LocationTown:        {200, 20},
		model.LocationVillage:     {195, 30},
		model.LocationBuilding:    {235, -20}, // near Liscor
		model.LocationLandmark:    {230, 0},
		model.LocationDungeon:     {240, -25},
		model.LocationBodyOfWater: {215, -15},
		model.LocationForest:      {205, 40},
		model.LocationRoad:        {210, 15},
		model.LocationOther:       {220, 5},
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
