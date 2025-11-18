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

// DayDetector detects the current day and phase in the game
type DayDetector struct {
	*BaseDetector
	config     *config.Config
	lastResult *DayResult

	// Detection regions
	dayRegion Rect

	// Template cache
	templates map[string]*DayTemplate

	// Current language
	currentLang string

	// Configuration
	updateInterval    time.Duration
	lastUpdateTime    time.Time
	matchThreshold    float64
	enableTemplateMatch bool
}

// NewDayDetector creates a new day detector
func NewDayDetector(cfg *config.Config) *DayDetector {
	return &DayDetector{
		BaseDetector:      NewBaseDetector("DayDetector"),
		config:            cfg,
		updateInterval:    time.Duration(cfg.UpdateInterval * float64(time.Second)),
		templates:         make(map[string]*DayTemplate),
		currentLang:       "chs", // Default to simplified Chinese
		matchThreshold:    0.8,   // Default threshold
		enableTemplateMatch: false, // Disable by default (use mock mode)
		lastResult: &DayResult{
			IsDetected: false,
		},
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
	// Get the data directory path
	dataDir := "data/day_template"

	// Check if directory exists
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return fmt.Errorf("template directory not found: %s", dataDir)
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

// detectDay detects the current day from the image
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

	// Crop image to day detection region
	dayImg := CropImage(img, d.dayRegion)

	// Try to match each day template
	bestDay := -1
	bestSimilarity := 0.0

	// Match Day 1
	if result, err := TemplateMatch(dayImg, template.Day1, d.matchThreshold); err == nil && result.Found {
		if result.Similarity > bestSimilarity {
			bestSimilarity = result.Similarity
			bestDay = 1
		}
	}

	// Match Day 2
	if result, err := TemplateMatch(dayImg, template.Day2, d.matchThreshold); err == nil && result.Found {
		if result.Similarity > bestSimilarity {
			bestSimilarity = result.Similarity
			bestDay = 2
		}
	}

	// Match Day 3
	if result, err := TemplateMatch(dayImg, template.Day3, d.matchThreshold); err == nil && result.Found {
		if result.Similarity > bestSimilarity {
			bestSimilarity = result.Similarity
			bestDay = 3
		}
	}

	if bestDay > 0 {
		logger.Debugf("[%s] Detected day %d with similarity %.2f", d.Name(), bestDay, bestSimilarity)
		return bestDay
	}

	// If no match found, return -1
	return -1
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
