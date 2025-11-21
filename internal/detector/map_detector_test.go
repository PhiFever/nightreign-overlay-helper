package detector

import (
	"testing"

	"github.com/PhiFever/nightreign-overlay-helper/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestLoadMapInfo(t *testing.T) {
	// Load map info from CSV files
	info, err := LoadMapInfo(
		utils.GetDataPath("csv/map_patterns.csv"),
		utils.GetDataPath("csv/constructs.csv"),
		utils.GetDataPath("csv/names.csv"),
		utils.GetDataPath("csv/positions.csv"),
	)

	assert.NoError(t, err, "Should load map info without error")
	assert.NotNil(t, info, "Map info should not be nil")

	// Check patterns loaded
	assert.Greater(t, len(info.Patterns), 300, "Should load at least 300 patterns")
	t.Logf("Loaded %d map patterns", len(info.Patterns))

	// Check names loaded
	assert.Greater(t, len(info.NameDict), 50, "Should load at least 50 names")
	t.Logf("Loaded %d names", len(info.NameDict))

	// Check positions loaded
	assert.Greater(t, len(info.PosDict), 100, "Should load at least 100 positions")
	t.Logf("Loaded %d positions", len(info.PosDict))

	// Check POI data
	assert.Greater(t, len(info.AllPOIPos), 20, "Should have at least 20 POI positions")
	assert.Greater(t, len(info.AllPOIConstructs), 10, "Should have at least 10 POI types")
	t.Logf("Found %d POI positions, %d POI types", len(info.AllPOIPos), len(info.AllPOIConstructs))

	// Verify first pattern structure
	if len(info.Patterns) > 0 {
		p := info.Patterns[0]
		assert.NotNil(t, p.PosConstructs, "Pattern should have constructs map")
		t.Logf("Pattern #%d: EarthShifting=%d, Day1Boss=%d, Day2Boss=%d, Constructs=%d",
			p.ID, p.EarthShifting, p.Day1Boss, p.Day2Boss, len(p.PosConstructs))
	}

	// Verify earth shifting distribution
	earthShiftingCount := make(map[int]int)
	for _, p := range info.Patterns {
		earthShiftingCount[p.EarthShifting]++
	}
	t.Logf("Earth shifting distribution: %v", earthShiftingCount)

	// There should be patterns for earth shifting 0, 1, 2, 3, 5 (skip 4)
	for _, es := range []int{0, 1, 2, 3, 5} {
		assert.Greater(t, earthShiftingCount[es], 0, "Should have patterns for earth shifting %d", es)
	}
}

func TestNewMapDetector(t *testing.T) {
	detector, err := NewMapDetector()
	assert.NoError(t, err, "Should create detector without error")
	assert.NotNil(t, detector, "Detector should not be nil")

	// Verify earth maps loaded
	assert.Equal(t, 5, len(detector.earthMaps), "Should load 5 earth maps")

	// Verify detector state
	assert.True(t, detector.IsEnabled(), "Detector should be enabled by default")
	assert.NotNil(t, detector.info, "Detector should have map info")
	assert.Greater(t, len(detector.info.Patterns), 300, "Should have patterns loaded")
}

func TestFilterPatternsByEarthShifting(t *testing.T) {
	detector, err := NewMapDetector()
	assert.NoError(t, err)

	// Test filtering for each earth shifting type
	for _, es := range []int{0, 1, 2, 3, 5} {
		candidates := detector.filterPatternsByEarthShifting(es)
		assert.Greater(t, len(candidates), 0, "Should have candidates for earth shifting %d", es)
		t.Logf("Earth shifting %d: %d candidates", es, len(candidates))

		// Verify all candidates have correct earth shifting
		for _, p := range candidates {
			assert.Equal(t, es, p.EarthShifting, "Candidate should have correct earth shifting")
		}
	}
}

func TestGetName(t *testing.T) {
	detector, err := NewMapDetector()
	assert.NoError(t, err)

	// Test some known names (from CSV analysis)
	testCases := []struct {
		id   int
		name string
	}{
		{45510, "黑刀刺客"},
		{46510, "红狼"},
		{46520, "龙装"},
	}

	for _, tc := range testCases {
		name := detector.info.GetName(tc.id)
		if name != "" {
			assert.Equal(t, tc.name, name, "Name should match for ID %d", tc.id)
		}
	}
}

func TestIsPOIConstruct(t *testing.T) {
	testCases := []struct {
		ctype    int
		expected bool
	}{
		{30301, true},  // 结晶人要塞-魔
		{32101, true},  // 红狮子营地-火
		{34001, true},  // 鲜血遗迹-出血
		{38100, true},  // 火焰教堂-火
		{40008, true},  // 法师塔
		{41000, true},  // 雕像
		{45510, false}, // BOSS (not POI)
		{46510, false}, // BOSS (not POI)
		{49420, false}, // 主城 (not POI)
	}

	for _, tc := range testCases {
		result := isPOIConstruct(tc.ctype)
		assert.Equal(t, tc.expected, result, "isPOIConstruct(%d) should be %v", tc.ctype, tc.expected)
	}
}

// Benchmark tests
func BenchmarkLoadMapInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := LoadMapInfo(
			utils.GetDataPath("csv/map_patterns.csv"),
			utils.GetDataPath("csv/constructs.csv"),
			utils.GetDataPath("csv/names.csv"),
			utils.GetDataPath("csv/positions.csv"),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewMapDetector(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewMapDetector()
		if err != nil {
			b.Fatal(err)
		}
	}
}
