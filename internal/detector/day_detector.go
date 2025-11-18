package detector

import (
	"fmt"
	"image"
	_ "image/png" // Import PNG decoder
	"os"
	"path/filepath"
	"time"

	"github.com/PhiFever/nightreign-overlay-helper/internal/config"
	"github.com/PhiFever/nightreign-overlay-helper/internal/logger"
)

// DayResult represents the result of day detection
type DayResult struct {
	Day          int           // Current day (0-based)
	Phase        int           // Current phase within the day
	ElapsedTime  time.Duration // Time elapsed since day start
	ShrinkTime   time.Duration // Time until next shrink
	NextPhaseIn  time.Duration // Time until next phase
	IsDetected   bool          // Whether day was successfully detected
	LastUpdateAt time.Time     // When this result was last updated
}

// String returns a string representation of the result
func (r *DayResult) String() string {
	if !r.IsDetected {
		return "Day: Not Detected"
	}
	return fmt.Sprintf("Day %d Phase %d | Elapsed: %v | Shrink in: %v | Next phase in: %v",
		r.Day, r.Phase, r.ElapsedTime, r.ShrinkTime, r.NextPhaseIn)
}

// DayTemplate represents templates for a specific language
type DayTemplate struct {
	Language string
	Day1     image.Image
	Day2     image.Image
	Day3     image.Image
}

// DetectionStrategy represents the detection strategy to use
type DetectionStrategy int

const (
	// StrategyAuto automatically selects the best strategy
	StrategyAuto DetectionStrategy = iota
	// StrategyHotspotCache uses cached hotspot from previous detection
	StrategyHotspotCache
	// StrategyColorFilter uses color-based filtering to find candidates
	StrategyColorFilter
	// StrategyPyramid uses image pyramid for multi-scale search
	StrategyPyramid
	// StrategyPredefined searches in predefined common locations
	StrategyPredefined
	// StrategyFullScan performs full screen scan (slowest, most thorough)
	StrategyFullScan
)

// DetectionStats tracks detection performance metrics
type DetectionStats struct {
	LastStrategy      DetectionStrategy
	LastDetectionTime time.Duration
	CacheHitCount     int
	ColorFilterCount  int
	PyramidCount      int
	PredefinedCount   int
	FullScanCount     int
	TotalDetections   int
}

// DayDetector detects the current day and phase in the game
type DayDetector struct {
	*BaseDetector
	config     *config.Config
	lastResult *DayResult

	// Detection regions (legacy, for backward compatibility)
	dayRegion Rect

	// Template cache
	templates map[string]*DayTemplate

	// Current language
	currentLang string

	// Configuration
	updateInterval      time.Duration
	lastUpdateTime      time.Time
	matchThreshold      float64
	enableTemplateMatch bool

	// Smart detection
	lastMatchLocation *Point            // Cached location from last successful match
	searchRadius      int               // Radius for local search around cached location
	strategy          DetectionStrategy // Current detection strategy
	stats             DetectionStats    // Performance statistics

	// Performance tuning
	colorFilterThreshold float64 // Threshold for bright pixel ratio (0.0-1.0)
	pyramidScales        []float64
	candidateStepSize    int // Step size for candidate region scanning
}

// NewDayDetector creates a new day detector
func NewDayDetector(cfg *config.Config) *DayDetector {
	return &DayDetector{
		BaseDetector:        NewBaseDetector("DayDetector"),
		config:              cfg,
		updateInterval:      time.Duration(cfg.UpdateInterval * float64(time.Second)),
		templates:           make(map[string]*DayTemplate),
		currentLang:         "chs", // Default to simplified Chinese
		matchThreshold:      0.8,   // Default threshold
		enableTemplateMatch: false, // Disable by default (use mock mode)
		lastResult: &DayResult{
			IsDetected: false,
		},
		// Smart detection settings
		searchRadius:         100,            // Search within 100px radius of last match
		strategy:             StrategyAuto,   // Auto-select strategy
		colorFilterThreshold: 0.1,            // 10% bright pixels indicates potential text
		pyramidScales:        []float64{0.25, 0.5, 1.0}, // Multi-scale search
		candidateStepSize:    50,             // Scan every 50px
		stats:                DetectionStats{},
	}
}

// SetLanguage sets the current language for template matching
func (d *DayDetector) SetLanguage(lang string) {
	d.currentLang = lang
}

// EnableTemplateMatching enables or disables template matching
func (d *DayDetector) EnableTemplateMatching(enable bool) {
	d.enableTemplateMatch = enable
}

// SetMatchThreshold sets the similarity threshold for template matching
func (d *DayDetector) SetMatchThreshold(threshold float64) {
	d.matchThreshold = threshold
}

// SetDetectionStrategy sets the detection strategy
func (d *DayDetector) SetDetectionStrategy(strategy DetectionStrategy) {
	d.strategy = strategy
}

// GetDetectionStats returns the current detection statistics
func (d *DayDetector) GetDetectionStats() DetectionStats {
	return d.stats
}

// SetSearchRadius sets the search radius for hotspot cache
func (d *DayDetector) SetSearchRadius(radius int) {
	d.searchRadius = radius
}

// ResetCache clears the cached hotspot location
func (d *DayDetector) ResetCache() {
	d.lastMatchLocation = nil
	logger.Debugf("[%s] Hotspot cache reset", d.Name())
}

// Initialize initializes the day detector
func (d *DayDetector) Initialize() error {
	logger.Infof("[%s] Initializing...", d.Name())

	// Load templates from data directory
	if err := d.loadTemplates(); err != nil {
		logger.Warningf("[%s] Failed to load templates: %v (using mock mode)", d.Name(), err)
		// Don't return error - we can still run in mock mode
	} else {
		logger.Infof("[%s] Templates loaded successfully", d.Name())
	}

	// Set default detection region (should be calibrated for actual game)
	d.dayRegion = NewRect(100, 50, 200, 50)

	logger.Infof("[%s] Initialized successfully", d.Name())
	return nil
}

// loadTemplates loads day number templates from the data directory
func (d *DayDetector) loadTemplates() error {
	// Get the data directory path, try multiple possible locations
	possiblePaths := []string{
		"data/day_template",         // When running from project root
		"../../data/day_template",   // When running tests
		"../data/day_template",      // Alternative location
	}

	var dataDir string
	var found bool
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			dataDir = path
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("template directory not found in any of: %v", possiblePaths)
	}

	// Languages to load
	languages := []string{"chs", "cht", "eng", "jp"}

	for _, lang := range languages {
		template := &DayTemplate{
			Language: lang,
		}

		// Load day 1, 2, 3 templates
		for day := 1; day <= 3; day++ {
			filename := filepath.Join(dataDir, fmt.Sprintf("%s_%d.png", lang, day))

			img, err := loadImageFromFile(filename)
			if err != nil {
				return fmt.Errorf("failed to load template %s: %w", filename, err)
			}

			// Store template
			switch day {
			case 1:
				template.Day1 = img
			case 2:
				template.Day2 = img
			case 3:
				template.Day3 = img
			}
		}

		d.templates[lang] = template
		logger.Debugf("[%s] Loaded templates for language: %s", d.Name(), lang)
	}

	return nil
}

// loadImageFromFile loads an image from a file
func loadImageFromFile(filename string) (image.Image, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return img, nil
}

// Detect performs day detection on the given image
func (d *DayDetector) Detect(img image.Image) (interface{}, error) {
	if !d.IsEnabled() {
		return d.lastResult, nil
	}

	// Check if we should update (rate limiting)
	now := time.Now()
	if now.Sub(d.lastUpdateTime) < d.updateInterval {
		return d.lastResult, nil
	}
	d.lastUpdateTime = now

	// Detect day and phase from the image
	day := d.detectDay(img)
	phase := d.detectPhase(img)

	// If detection failed, return the last result
	if day < 0 || phase < 0 {
		result := &DayResult{
			IsDetected:   false,
			LastUpdateAt: now,
		}
		d.lastResult = result
		return result, nil
	}

	// Calculate timing information
	elapsed, shrink, nextPhase := d.calculateTimes(day, phase)

	result := &DayResult{
		Day:          day,
		Phase:        phase,
		ElapsedTime:  elapsed,
		ShrinkTime:   shrink,
		NextPhaseIn:  nextPhase,
		IsDetected:   true,
		LastUpdateAt: now,
	}

	d.lastResult = result
	return result, nil
}

// Cleanup releases resources used by the detector
func (d *DayDetector) Cleanup() error {
	logger.Infof("[%s] Cleaning up...", d.Name())

	// Clear templates
	d.templates = nil
	d.lastResult = nil

	logger.Infof("[%s] Cleaned up successfully", d.Name())
	return nil
}

// GetLastResult returns the last detection result
func (d *DayDetector) GetLastResult() *DayResult {
	return d.lastResult
}

// detectDay detects the current day from the image using intelligent multi-layer search
// Returns -1 if not detected
func (d *DayDetector) detectDay(img image.Image) int {
	// If template matching is disabled, use mock mode
	if !d.enableTemplateMatch {
		return d.detectDayMock()
	}

	// Get template for current language
	template, ok := d.templates[d.currentLang]
	if !ok {
		logger.Warningf("[%s] No template found for language: %s", d.Name(), d.currentLang)
		return d.detectDayMock()
	}

	startTime := time.Now()

	// Use intelligent detection based on strategy
	var day int
	var location *Point

	switch d.strategy {
	case StrategyHotspotCache:
		day, location = d.detectWithHotspotCache(img, template)
	case StrategyColorFilter:
		day, location = d.detectWithColorFilter(img, template)
	case StrategyPyramid:
		day, location = d.detectWithPyramid(img, template)
	case StrategyPredefined:
		day, location = d.detectWithPredefined(img, template)
	case StrategyFullScan:
		day, location = d.detectWithFullScan(img, template)
	default: // StrategyAuto
		day, location = d.detectDayIntelligent(img, template)
	}

	// Update statistics
	d.stats.LastDetectionTime = time.Since(startTime)
	d.stats.TotalDetections++

	// Update cached location if found
	if day > 0 && location != nil {
		d.lastMatchLocation = location
	}

	return day
}

// detectDayIntelligent uses multi-layer intelligent detection (Auto strategy)
func (d *DayDetector) detectDayIntelligent(img image.Image, template *DayTemplate) (int, *Point) {
	// Layer 1: Hotspot cache (fastest, usually hits)
	if d.lastMatchLocation != nil {
		day, loc := d.detectWithHotspotCache(img, template)
		if day > 0 {
			d.stats.LastStrategy = StrategyHotspotCache
			d.stats.CacheHitCount++
			return day, loc
		}
	}

	// Layer 2: Color-based filtering (fast, narrows down search)
	day, loc := d.detectWithColorFilter(img, template)
	if day > 0 {
		d.stats.LastStrategy = StrategyColorFilter
		d.stats.ColorFilterCount++
		return day, loc
	}

	// Layer 3: Image pyramid (medium speed, good coverage)
	day, loc = d.detectWithPyramid(img, template)
	if day > 0 {
		d.stats.LastStrategy = StrategyPyramid
		d.stats.PyramidCount++
		return day, loc
	}

	// Layer 4: Predefined hotspots (fast, common locations)
	day, loc = d.detectWithPredefined(img, template)
	if day > 0 {
		d.stats.LastStrategy = StrategyPredefined
		d.stats.PredefinedCount++
		return day, loc
	}

	// Layer 5: Full scan (slowest, most thorough - last resort)
	logger.Debugf("[%s] Falling back to full scan", d.Name())
	day, loc = d.detectWithFullScan(img, template)
	if day > 0 {
		d.stats.LastStrategy = StrategyFullScan
		d.stats.FullScanCount++
	}

	return day, loc
}

// detectWithHotspotCache searches near the last known location
func (d *DayDetector) detectWithHotspotCache(img image.Image, template *DayTemplate) (int, *Point) {
	if d.lastMatchLocation == nil {
		return -1, nil
	}

	bounds := img.Bounds()
	x := max(0, d.lastMatchLocation.X-d.searchRadius)
	y := max(0, d.lastMatchLocation.Y-d.searchRadius)
	w := min(d.searchRadius*2, bounds.Dx()-x)
	h := min(d.searchRadius*2, bounds.Dy()-y)

	region := NewRect(x, y, w, h)
	day, loc := d.matchDayInRegion(img, template, region)

	if day > 0 && loc != nil {
		logger.Debugf("[%s] Cache hit! Found Day %d near cached location", d.Name(), day)
		return day, loc
	}

	return -1, nil
}

// detectWithColorFilter uses color-based filtering to find candidate regions
func (d *DayDetector) detectWithColorFilter(img image.Image, template *DayTemplate) (int, *Point) {
	// Estimate search window size based on template
	templateBounds := template.Day1.Bounds()
	windowW := templateBounds.Dx() * 3
	windowH := templateBounds.Dy() * 3

	// Find candidate regions with bright pixels
	candidates := FindCandidateRegions(img, windowW, windowH, d.candidateStepSize, d.colorFilterThreshold)

	logger.Debugf("[%s] Color filter found %d candidate regions", d.Name(), len(candidates))

	// Search in candidate regions
	for _, region := range candidates {
		day, loc := d.matchDayInRegion(img, template, region)
		if day > 0 {
			return day, loc
		}
	}

	return -1, nil
}

// detectWithPyramid uses image pyramid for multi-scale search
func (d *DayDetector) detectWithPyramid(img image.Image, template *DayTemplate) (int, *Point) {
	// Try each day template with pyramid search
	templates := []image.Image{template.Day1, template.Day2, template.Day3}

	for dayNum, tmpl := range templates {
		result, err := TemplateMatchPyramid(img, tmpl, d.matchThreshold, d.pyramidScales)
		if err == nil && result.Found {
			day := dayNum + 1
			logger.Debugf("[%s] Pyramid found Day %d at (%d, %d) with similarity %.2f",
				d.Name(), day, result.Location.X, result.Location.Y, result.Similarity)
			return day, &result.Location
		}
	}

	return -1, nil
}

// detectWithPredefined searches in predefined common UI locations
func (d *DayDetector) detectWithPredefined(img image.Image, template *DayTemplate) (int, *Point) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Common UI locations based on typical game layouts
	predefinedRegions := []Rect{
		// Top-left corner (most common for game info)
		NewRect(int(float64(w)*0.02), int(float64(h)*0.02), int(float64(w)*0.20), int(float64(h)*0.15)),
		// Top-right corner
		NewRect(int(float64(w)*0.78), int(float64(h)*0.02), int(float64(w)*0.20), int(float64(h)*0.15)),
		// Top-center
		NewRect(int(float64(w)*0.40), int(float64(h)*0.02), int(float64(w)*0.20), int(float64(h)*0.15)),
		// Left-center
		NewRect(int(float64(w)*0.02), int(float64(h)*0.40), int(float64(w)*0.20), int(float64(h)*0.20)),
	}

	for _, region := range predefinedRegions {
		day, loc := d.matchDayInRegion(img, template, region)
		if day > 0 {
			logger.Debugf("[%s] Found in predefined region at (%d, %d)", d.Name(), loc.X, loc.Y)
			return day, loc
		}
	}

	return -1, nil
}

// detectWithFullScan performs full screen template matching (slowest)
func (d *DayDetector) detectWithFullScan(img image.Image, template *DayTemplate) (int, *Point) {
	bounds := img.Bounds()
	fullRegion := NewRect(bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy())

	return d.matchDayInRegion(img, template, fullRegion)
}

// matchDayInRegion tries to match day templates in a specific region
func (d *DayDetector) matchDayInRegion(img image.Image, template *DayTemplate, region Rect) (int, *Point) {
	// Crop to region
	regionImg := CropImage(img, region)

	// Try each day template
	templates := []image.Image{template.Day1, template.Day2, template.Day3}
	bestDay := -1
	bestSimilarity := 0.0
	var bestLocation *Point

	for dayNum, tmpl := range templates {
		result, err := TemplateMatch(regionImg, tmpl, d.matchThreshold)
		if err == nil && result.Found && result.Similarity > bestSimilarity {
			bestSimilarity = result.Similarity
			bestDay = dayNum + 1
			// Adjust location to account for region offset
			bestLocation = &Point{
				X: region.X + result.Location.X,
				Y: region.Y + result.Location.Y,
			}
		}
	}

	if bestDay > 0 {
		logger.Debugf("[%s] Matched Day %d with similarity %.2f", d.Name(), bestDay, bestSimilarity)
		return bestDay, bestLocation
	}

	return -1, nil
}

// detectDayMock provides mock day detection for testing
func (d *DayDetector) detectDayMock() int {
	// Simulate detecting Day 1-3 based on time
	seconds := time.Now().Unix() % 30
	if seconds < 10 {
		return 1
	} else if seconds < 20 {
		return 2
	} else {
		return 3
	}
}

// detectPhase detects the current phase from the image
// Returns -1 if not detected
func (d *DayDetector) detectPhase(img image.Image) int {
	// TODO: Implement template matching for phase markers
	// For now, simulate phase detection that cycles 0-3

	// Simulate detecting phases 0-3 based on time
	seconds := time.Now().Unix() % 20
	return int(seconds / 5) // Returns 0, 1, 2, or 3
}

// calculateTimes calculates elapsed time, shrink time, and next phase time
func (d *DayDetector) calculateTimes(day, phase int) (elapsed, shrink, nextPhase time.Duration) {
	// Calculate based on game configuration
	if day < 0 || phase < 0 || phase >= len(d.config.DayPeriodSeconds) {
		return 0, 0, 0
	}

	// Calculate elapsed time from start of day
	elapsedSeconds := 0
	for i := 0; i < phase; i++ {
		if i < len(d.config.DayPeriodSeconds) {
			elapsedSeconds += d.config.DayPeriodSeconds[i]
		}
	}

	// Add current phase elapsed time (mock - in real version this would be detected)
	currentPhaseElapsed := int(time.Now().Unix() % int64(d.config.DayPeriodSeconds[phase]))
	elapsedSeconds += currentPhaseElapsed

	elapsed = time.Duration(elapsedSeconds) * time.Second

	// Calculate time until next shrink
	if phase < len(d.config.DayPeriodSeconds) {
		shrinkSeconds := d.config.DayPeriodSeconds[phase] - currentPhaseElapsed
		shrink = time.Duration(shrinkSeconds) * time.Second
	}

	// Next phase is the same as shrink time for now
	nextPhase = shrink

	return elapsed, shrink, nextPhase
}
