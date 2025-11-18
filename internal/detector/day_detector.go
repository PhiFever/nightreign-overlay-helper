package detector

import (
	"fmt"
	"image"
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

// DayDetector detects the current day and phase in the game
type DayDetector struct {
	*BaseDetector
	config     *config.Config
	lastResult *DayResult

	// Detection regions
	dayRegion   Rect
	phaseRegion Rect

	// Template cache (will be loaded from assets)
	// templates map[string]*image.Gray

	// Configuration
	updateInterval time.Duration
	lastUpdateTime time.Time
}

// NewDayDetector creates a new day detector
func NewDayDetector(cfg *config.Config) *DayDetector {
	return &DayDetector{
		BaseDetector:   NewBaseDetector("DayDetector"),
		config:         cfg,
		updateInterval: time.Duration(cfg.UpdateInterval * float64(time.Second)),
		lastResult: &DayResult{
			IsDetected: false,
		},
	}
}

// Initialize initializes the day detector
func (d *DayDetector) Initialize() error {
	logger.Infof("[%s] Initializing...", d.Name())

	// TODO: Load templates from assets
	// For now, we'll use placeholder detection regions
	// These should be loaded from config or calibrated
	d.dayRegion = NewRect(100, 50, 200, 50)
	d.phaseRegion = NewRect(100, 100, 200, 50)

	logger.Infof("[%s] Initialized successfully", d.Name())
	return nil
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

	// TODO: Release template resources
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
	// TODO: Implement template matching for day numbers using OCR
	// For now, implement a simple mock detector that cycles through days
	// This is for testing the framework

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
