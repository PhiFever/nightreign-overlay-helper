package detector

import (
	"fmt"
	"image"
	"math"
	"path/filepath"
	"sync"
	"time"

	"github.com/PhiFever/nightreign-overlay-helper/internal/logger"
	"github.com/PhiFever/nightreign-overlay-helper/pkg/utils"
)

// MapDetector handles map detection and pattern matching
type MapDetector struct {
	info          *MapInfo
	earthMaps     map[int]image.Image
	earthMapsLock sync.RWMutex
	enabled       bool
	mu            sync.RWMutex
}

// MapDetectResult contains the detection result
type MapDetectResult struct {
	EarthShifting      int
	EarthShiftingScore float64
	Pattern            *MapPattern
	PatternScore       int
	// OverlayImage will be added later when implementing drawing
}

// Detection parameters
const (
	// Earth shifting detection
	earthShiftingStdSize   = 100
	earthShiftingROIX      = 20  // 20%
	earthShiftingROIY      = 20  // 20%
	earthShiftingROIW      = 60  // 60%
	earthShiftingROIH      = 60  // 60%
	earthShiftingOffset    = 5
	earthShiftingStride    = 1
	earthShiftingMinScale  = 0.95
	earthShiftingMaxScale  = 1.05
	earthShiftingScaleSteps = 7

	// POI detection
	stdPOISize      = 45
	poiDownsample   = 16
	poiMaxOffset    = 6
	poiOffsetStride = 2
)

// NewMapDetector creates a new map detector
func NewMapDetector() (*MapDetector, error) {
	// Load map info from CSV files
	info, err := LoadMapInfo(
		utils.GetDataPath("csv/map_patterns.csv"),
		utils.GetDataPath("csv/constructs.csv"),
		utils.GetDataPath("csv/names.csv"),
		utils.GetDataPath("csv/positions.csv"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load map info: %w", err)
	}

	logger.Info(fmt.Sprintf("Loaded %d map patterns, %d POI positions, %d POI types",
		len(info.Patterns), len(info.AllPOIPos), len(info.AllPOIConstructs)))

	detector := &MapDetector{
		info:      info,
		earthMaps: make(map[int]image.Image),
		enabled:   true,
	}

	// Load earth shifting maps (0, 1, 2, 3, 5 - skip 4)
	if err := detector.loadEarthMaps(); err != nil {
		return nil, fmt.Errorf("failed to load earth maps: %w", err)
	}

	return detector, nil
}

// loadEarthMaps loads the background maps for earth shifting detection
func (d *MapDetector) loadEarthMaps() error {
	earthIDs := []int{0, 1, 2, 3, 5} // Skip 4 as per Python code

	for _, id := range earthIDs {
		path := utils.GetDataPath(filepath.Join("maps", fmt.Sprintf("%d.jpg", id)))
		img, err := LoadImageFromFile(path)
		if err != nil {
			return fmt.Errorf("failed to load earth map %d: %w", id, err)
		}
		d.earthMaps[id] = img
		logger.Debug(fmt.Sprintf("Loaded earth map %d from %s", id, path))
	}

	return nil
}

// Name returns the detector name
func (d *MapDetector) Name() string {
	return "MapDetector"
}

// IsEnabled returns whether the detector is enabled
func (d *MapDetector) IsEnabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.enabled
}

// SetEnabled sets the detector enabled state
func (d *MapDetector) SetEnabled(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = enabled
}

// Initialize initializes the detector
func (d *MapDetector) Initialize() error {
	return nil
}

// Cleanup cleans up resources
func (d *MapDetector) Cleanup() error {
	return nil
}

// Detect performs map detection on the given image
func (d *MapDetector) Detect(img image.Image) (interface{}, error) {
	if !d.IsEnabled() {
		return nil, nil
	}

	startTime := time.Now()

	// Resize to standard size
	stdImg := ResizeImage(img, StdMapWidth, StdMapHeight)

	result := &MapDetectResult{}

	// Step 1: Detect earth shifting (special terrain)
	earthShifting, score, err := d.detectEarthShifting(stdImg)
	if err != nil {
		return nil, fmt.Errorf("earth shifting detection failed: %w", err)
	}
	result.EarthShifting = earthShifting
	result.EarthShiftingScore = score

	logger.Info(fmt.Sprintf("[MapDetector] Earth shifting: %d (score: %.4f) in %.3fs",
		earthShifting, score, time.Since(startTime).Seconds()))

	// Step 2: Filter patterns by earth shifting
	candidates := d.filterPatternsByEarthShifting(earthShifting)
	logger.Info(fmt.Sprintf("[MapDetector] Filtered to %d candidates by earth shifting", len(candidates)))

	if len(candidates) == 0 {
		return result, fmt.Errorf("no matching patterns for earth shifting %d", earthShifting)
	}

	// Step 3: Match pattern by POI (to be implemented)
	// For now, just return the first candidate
	result.Pattern = candidates[0]
	result.PatternScore = 0

	logger.Info(fmt.Sprintf("[MapDetector] Total detection time: %.3fs", time.Since(startTime).Seconds()))

	return result, nil
}

// detectEarthShifting detects the special terrain type
func (d *MapDetector) detectEarthShifting(img image.Image) (int, float64, error) {
	startTime := time.Now()

	// Resize to standard detection size
	resized := ResizeImage(img, earthShiftingStdSize, earthShiftingStdSize)

	// Extract ROI (region of interest)
	roiX := earthShiftingStdSize * earthShiftingROIX / 100
	roiY := earthShiftingStdSize * earthShiftingROIY / 100
	roiW := earthShiftingStdSize * earthShiftingROIW / 100
	roiH := earthShiftingStdSize * earthShiftingROIH / 100

	roi := CropImage(resized, NewRect(roiX, roiY, roiW, roiH))

	// Convert to RGB array for comparison
	roiRGB := imageToRGBArray(roi)

	bestMapID := -1
	bestScore := math.Inf(1)

	// Try each earth map
	d.earthMapsLock.RLock()
	defer d.earthMapsLock.RUnlock()

	type result struct {
		mapID int
		score float64
	}
	results := make(chan result, len(d.earthMaps))

	var wg sync.WaitGroup
	for mapID, mapImg := range d.earthMaps {
		wg.Add(1)
		go func(id int, mImg image.Image) {
			defer wg.Done()
			score := d.matchEarthMap(roiRGB, mImg, roiX, roiY, roiW, roiH)
			results <- result{mapID: id, score: score}
		}(mapID, mapImg)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for res := range results {
		if res.score < bestScore {
			bestScore = res.score
			bestMapID = res.mapID
		}
	}

	logger.Info(fmt.Sprintf("[MapDetector] Earth shifting detection: best map %d, score %.4f, time %.3fs",
		bestMapID, bestScore, time.Since(startTime).Seconds()))

	return bestMapID, bestScore, nil
}

// matchEarthMap matches the ROI against an earth map with multi-scale search
func (d *MapDetector) matchEarthMap(roiRGB [][][3]float64, mapImg image.Image, roiX, roiY, roiW, roiH int) float64 {
	bestScore := math.Inf(1)

	// Generate scales
	scales := make([]float64, earthShiftingScaleSteps)
	for i := 0; i < earthShiftingScaleSteps; i++ {
		scales[i] = earthShiftingMinScale + float64(i)*(earthShiftingMaxScale-earthShiftingMinScale)/float64(earthShiftingScaleSteps-1)
	}

	for _, scale := range scales {
		// Resize map image
		newW := int(float64(earthShiftingStdSize) * scale)
		newH := int(float64(earthShiftingStdSize) * scale)
		scaledMap := ResizeImage(mapImg, newW, newH)
		scaledMapRGB := imageToRGBArray(scaledMap)

		// Try different offsets
		for dx := -earthShiftingOffset; dx <= earthShiftingOffset; dx += earthShiftingStride {
			for dy := -earthShiftingOffset; dy <= earthShiftingOffset; dy += earthShiftingStride {
				// Extract shifted ROI from scaled map
				mapROI := extractRGBROI(scaledMapRGB, roiX+dx, roiY+dy, roiW, roiH)

				if mapROI == nil {
					continue
				}

				// Calculate difference
				score := calculateImageDifference(roiRGB, mapROI)
				if score < bestScore {
					bestScore = score
				}
			}
		}
	}

	return bestScore
}

// filterPatternsByEarthShifting filters patterns by earth shifting type
func (d *MapDetector) filterPatternsByEarthShifting(earthShifting int) []*MapPattern {
	var candidates []*MapPattern
	for _, pattern := range d.info.Patterns {
		if pattern.EarthShifting == earthShifting {
			candidates = append(candidates, pattern)
		}
	}
	return candidates
}

// Helper functions

// imageToRGBArray converts an image to a 3D array [y][x][rgb]
func imageToRGBArray(img image.Image) [][][3]float64 {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	arr := make([][][3]float64, h)
	for y := 0; y < h; y++ {
		arr[y] = make([][3]float64, w)
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			arr[y][x] = [3]float64{
				float64(r >> 8),
				float64(g >> 8),
				float64(b >> 8),
			}
		}
	}
	return arr
}

// extractRGBROI extracts a region from an RGB array
func extractRGBROI(arr [][][3]float64, x, y, w, h int) [][][3]float64 {
	if y+h > len(arr) || y < 0 {
		return nil
	}
	if len(arr) > 0 && (x+w > len(arr[0]) || x < 0) {
		return nil
	}

	roi := make([][][3]float64, h)
	for i := 0; i < h; i++ {
		roi[i] = make([][3]float64, w)
		for j := 0; j < w; j++ {
			roi[i][j] = arr[y+i][x+j]
		}
	}
	return roi
}

// calculateImageDifference calculates the median difference between two RGB arrays
func calculateImageDifference(img1, img2 [][][3]float64) float64 {
	h := len(img1)
	if h == 0 || h != len(img2) {
		return math.Inf(1)
	}

	w := len(img1[0])
	if w == 0 || w != len(img2[0]) {
		return math.Inf(1)
	}

	// Calculate L2 norm of differences
	diffs := make([]float64, 0, h*w)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			diff := [3]float64{
				img1[y][x][0] - img2[y][x][0],
				img1[y][x][1] - img2[y][x][1],
				img1[y][x][2] - img2[y][x][2],
			}

			// L2 norm
			norm := math.Sqrt(diff[0]*diff[0] + diff[1]*diff[1] + diff[2]*diff[2])

			// Threshold at 100 (as per Python code)
			if norm > 100 {
				norm = 0
			}

			diffs = append(diffs, norm)
		}
	}

	// Return median
	return median(diffs)
}

// median calculates the median of a float64 slice
func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Simple median calculation (not sorting, using approximate)
	// For performance, we use a simple average of min/max/mean
	sum := 0.0
	min := values[0]
	max := values[0]

	for _, v := range values {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	mean := sum / float64(len(values))
	// Approximate median as weighted average
	return (min + mean + max) / 3.0
}
