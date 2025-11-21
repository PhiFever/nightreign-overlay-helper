package detector

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

// Position represents a 2D coordinate on the map
type Position struct {
	X int
	Y int
}

// Construct represents a building or structure on the map
type Construct struct {
	Type      int
	Pos       Position
	IsDisplay bool
}

// MapPattern represents a specific map configuration
type MapPattern struct {
	ID             int
	NightLord      int
	EarthShifting  int
	StartPos       Position
	Day1Boss       int
	Day1ExtraBoss  int
	Day1Pos        Position
	Day2Boss       int
	Day2ExtraBoss  int
	Day2Pos        Position
	Treasure       int
	RotRew         int
	EventValue     int
	EventFlag      int
	EvpatValue     int
	EvpatFlag      int
	PosConstructs  map[Position]*Construct
}

// MapInfo contains all map-related data
type MapInfo struct {
	NameDict          map[int]string
	PosDict           map[int]Position
	Patterns          []*MapPattern
	AllPOIPos         []Position
	AllPOIConstructs  []int
}

// GetName returns the name for a given ID
func (m *MapInfo) GetName(id int) string {
	return m.NameDict[id]
}

// Standard map size
const (
	StdMapWidth  = 750
	StdMapHeight = 750
)

// POI construct types
var POIConstructs = []int{30, 32, 34, 37, 38, 40, 41}

// LoadMapInfo loads all map data from CSV files
func LoadMapInfo(mapPatternsPath, constructsPath, namesPath, positionsPath string) (*MapInfo, error) {
	info := &MapInfo{
		NameDict: make(map[int]string),
		PosDict:  make(map[int]Position),
	}

	// Load names
	if err := loadNames(namesPath, info); err != nil {
		return nil, fmt.Errorf("failed to load names: %w", err)
	}

	// Load positions
	if err := loadPositions(positionsPath, info); err != nil {
		return nil, fmt.Errorf("failed to load positions: %w", err)
	}

	// Load constructs
	mapConstructs, allPOIPos, allPOITypes, err := loadConstructs(constructsPath, info)
	if err != nil {
		return nil, fmt.Errorf("failed to load constructs: %w", err)
	}
	info.AllPOIPos = allPOIPos
	info.AllPOIConstructs = allPOITypes

	// Load map patterns
	if err := loadMapPatterns(mapPatternsPath, info, mapConstructs); err != nil {
		return nil, fmt.Errorf("failed to load map patterns: %w", err)
	}

	return info, nil
}

// loadNames loads the names CSV
func loadNames(path string, info *MapInfo) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1 // Allow variable number of fields
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	// Skip header
	for i := 1; i < len(records); i++ {
		if len(records[i]) < 2 {
			continue
		}
		id, err := strconv.Atoi(records[i][0])
		if err != nil {
			continue
		}
		info.NameDict[id] = records[i][1]
	}

	return nil
}

// loadPositions loads the positions CSV and converts game coordinates to map pixels
func loadPositions(path string, info *MapInfo) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	// Skip header
	for i := 1; i < len(records); i++ {
		if len(records[i]) < 9 {
			continue
		}
		id, err := strconv.Atoi(records[i][0])
		if err != nil {
			continue
		}

		// Parse game coordinates (picX, picY)
		picX, err := strconv.ParseFloat(records[i][7], 64)
		if err != nil {
			continue
		}
		picY, err := strconv.ParseFloat(records[i][8], 64)
		if err != nil {
			continue
		}

		// Convert to map pixel coordinates (from Python: map_info.py:64-66)
		x := int((picX-907.5537109)/6.045 + 127.26920918617023)
		y := int((picY-1571.031006)/6.045 + 242.71771372340424)

		info.PosDict[id] = Position{X: x, Y: y}
	}

	return nil
}

// loadConstructs loads the constructs CSV
func loadConstructs(path string, info *MapInfo) (map[int][]*Construct, []Position, []int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, nil, err
	}

	mapConstructs := make(map[int][]*Construct)
	poiPosSet := make(map[Position]bool)
	poiTypeSet := make(map[int]bool)

	// Skip header
	for i := 1; i < len(records); i++ {
		if len(records[i]) < 4 {
			continue
		}

		mapID, err := strconv.Atoi(records[i][1])
		if err != nil {
			continue
		}

		cType, err := strconv.Atoi(records[i][2])
		if err != nil {
			continue
		}

		isDisplay := records[i][3] == "1"

		posID, err := strconv.Atoi(records[i][4])
		if err != nil {
			continue
		}

		pos, ok := info.PosDict[posID]
		if !ok {
			continue
		}

		construct := &Construct{
			Type:      cType,
			Pos:       pos,
			IsDisplay: isDisplay,
		}

		mapConstructs[mapID] = append(mapConstructs[mapID], construct)

		// Track POI positions and types
		if isPOIConstruct(cType) {
			poiPosSet[pos] = true
			poiTypeSet[cType] = true
		}
	}

	// Add empty construct type
	poiTypeSet[0] = true

	// Convert sets to slices
	allPOIPos := make([]Position, 0, len(poiPosSet))
	for pos := range poiPosSet {
		allPOIPos = append(allPOIPos, pos)
	}

	allPOITypes := make([]int, 0, len(poiTypeSet))
	for t := range poiTypeSet {
		allPOITypes = append(allPOITypes, t)
	}

	return mapConstructs, allPOIPos, allPOITypes, nil
}

// loadMapPatterns loads the map patterns CSV
func loadMapPatterns(path string, info *MapInfo, mapConstructs map[int][]*Construct) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	// Skip header
	for i := 1; i < len(records); i++ {
		if len(records[i]) < 16 {
			continue
		}

		pattern := &MapPattern{
			PosConstructs: make(map[Position]*Construct),
		}

		// Parse all fields
		pattern.ID, _ = strconv.Atoi(records[i][0])
		pattern.NightLord, _ = strconv.Atoi(records[i][1])
		pattern.EarthShifting, _ = strconv.Atoi(records[i][2])

		startPosID, _ := strconv.Atoi(records[i][3])
		pattern.StartPos = info.PosDict[startPosID]

		pattern.Treasure, _ = strconv.Atoi(records[i][4])
		pattern.EventValue, _ = strconv.Atoi(records[i][5])
		pattern.EventFlag, _ = strconv.Atoi(records[i][6])
		pattern.EvpatValue, _ = strconv.Atoi(records[i][7])
		pattern.EvpatFlag, _ = strconv.Atoi(records[i][8])
		pattern.RotRew, _ = strconv.Atoi(records[i][9])

		pattern.Day1Boss, _ = strconv.Atoi(records[i][10])
		day1LocID, _ := strconv.Atoi(records[i][11])
		pattern.Day1Pos = info.PosDict[day1LocID]

		pattern.Day2Boss, _ = strconv.Atoi(records[i][12])
		day2LocID, _ := strconv.Atoi(records[i][13])
		pattern.Day2Pos = info.PosDict[day2LocID]

		pattern.Day1ExtraBoss, _ = strconv.Atoi(records[i][14])
		pattern.Day2ExtraBoss, _ = strconv.Atoi(records[i][15])

		// Map constructs to this pattern
		for _, c := range mapConstructs[pattern.ID] {
			pattern.PosConstructs[c.Pos] = c
		}

		info.Patterns = append(info.Patterns, pattern)
	}

	return nil
}

// isPOIConstruct checks if a construct type is a POI
func isPOIConstruct(cType int) bool {
	mainType := cType / 1000
	for _, poiType := range POIConstructs {
		if mainType == poiType {
			return true
		}
	}
	return false
}
