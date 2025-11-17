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

	// TODO: Actual detection logic
	// For now, return a mock result
	result := &DayResult{
		Day:          1,
		Phase:        2,
		ElapsedTime:  5 * time.Minute,
		ShrinkTime:   2 * time.Minute,
		NextPhaseIn:  30 * time.Second,
		IsDetected:   true,
		LastUpdateAt: now,
	}

	d.lastResult = result
	logger.Debugf("[%s] %s", d.Name(), result.String())

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
	// TODO: Implement template matching for day numbers
	// Crop the day region
	// dayImg := CropImage(img, d.dayRegion)
	// Match against day templates
	// Return detected day number
	return -1
}

// detectPhase detects the current phase from the image
// Returns -1 if not detected
func (d *DayDetector) detectPhase(img image.Image) int {
	// TODO: Implement template matching for phase markers
	// Crop the phase region
	// phaseImg := CropImage(img, d.phaseRegion)
	// Match against phase templates
	// Return detected phase number
	return -1
}

// calculateTimes calculates elapsed time, shrink time, and next phase time
func (d *DayDetector) calculateTimes(day, phase int) (elapsed, shrink, nextPhase time.Duration) {
	// TODO: Implement time calculations based on game mechanics
	// This requires knowledge of:
	// - Phase durations
	// - Shrink timings
	// - Day progression rules

	// For now, return mock values
	elapsed = time.Duration(day*20+phase*5) * time.Minute
	shrink = 2 * time.Minute
	nextPhase = 30 * time.Second

	return elapsed, shrink, nextPhase
}
