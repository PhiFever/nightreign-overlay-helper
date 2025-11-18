package updater

import (
	"context"
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/PhiFever/nightreign-overlay-helper/internal/config"
	"github.com/PhiFever/nightreign-overlay-helper/internal/detector"
	"github.com/PhiFever/nightreign-overlay-helper/internal/logger"
)

// DetectorResult represents the result from a detector
type DetectorResult struct {
	DetectorName string
	Result       interface{}
	Error        error
	Timestamp    time.Time
}

// ScreenCapture is a function type for capturing screen
type ScreenCapture func() (image.Image, error)

// Updater coordinates all detectors and manages the detection loop
type Updater struct {
	config   *config.Config
	registry *detector.DetectorRegistry

	// Channels
	resultChan chan DetectorResult
	stopChan   chan struct{}
	doneChan   chan struct{}

	// Screen capture function
	captureFunc ScreenCapture

	// State
	running bool
	mu      sync.RWMutex

	// Statistics
	updateCount    uint64
	lastUpdateTime time.Time

	// Cache for last results to avoid duplicate logging
	lastResults map[string]string
	resultsMu   sync.Mutex
}

// NewUpdater creates a new updater
func NewUpdater(cfg *config.Config, registry *detector.DetectorRegistry) *Updater {
	return &Updater{
		config:      cfg,
		registry:    registry,
		resultChan:  make(chan DetectorResult, 100),
		stopChan:    make(chan struct{}),
		doneChan:    make(chan struct{}),
		captureFunc: mockCapture, // Use mock capture for now
		lastResults: make(map[string]string),
	}
}

// SetCaptureFunc sets the screen capture function
func (u *Updater) SetCaptureFunc(fn ScreenCapture) {
	u.captureFunc = fn
}

// Start starts the updater loop
func (u *Updater) Start(ctx context.Context) error {
	u.mu.Lock()
	if u.running {
		u.mu.Unlock()
		return fmt.Errorf("updater is already running")
	}
	u.running = true
	u.mu.Unlock()

	logger.Info("[Updater] Starting...")

	// Start result processor
	go u.processResults(ctx)

	// Start detection loop
	go u.detectionLoop(ctx)

	logger.Info("[Updater] Started successfully")
	return nil
}

// Stop stops the updater
func (u *Updater) Stop() error {
	u.mu.Lock()
	if !u.running {
		u.mu.Unlock()
		return fmt.Errorf("updater is not running")
	}
	u.running = false
	u.mu.Unlock()

	logger.Info("[Updater] Stopping...")

	// Signal stop
	close(u.stopChan)

	// Wait for done
	<-u.doneChan

	logger.Info("[Updater] Stopped successfully")
	return nil
}

// IsRunning returns whether the updater is running
func (u *Updater) IsRunning() bool {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.running
}

// GetResultChan returns the result channel
func (u *Updater) GetResultChan() <-chan DetectorResult {
	return u.resultChan
}

// detectionLoop runs the main detection loop
func (u *Updater) detectionLoop(ctx context.Context) {
	defer close(u.doneChan)

	interval := time.Duration(u.config.UpdateInterval * float64(time.Second))
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Infof("[Updater] Detection loop started (interval: %v)", interval)

	for {
		select {
		case <-ctx.Done():
			logger.Info("[Updater] Context cancelled, stopping detection loop")
			return

		case <-u.stopChan:
			logger.Info("[Updater] Stop signal received, stopping detection loop")
			return

		case <-ticker.C:
			u.runDetection()
		}
	}
}

// runDetection runs all enabled detectors
func (u *Updater) runDetection() {
	// Capture screen
	img, err := u.captureFunc()
	if err != nil {
		logger.Errorf("[Updater] Failed to capture screen: %v", err)
		return
	}

	// Get all detectors
	detectors := u.registry.GetAll()

	// Run detectors concurrently
	var wg sync.WaitGroup
	for _, d := range detectors {
		if !d.IsEnabled() {
			continue
		}

		wg.Add(1)
		go func(det detector.Detector) {
			defer wg.Done()

			result, err := det.Detect(img)

			// Send result
			select {
			case u.resultChan <- DetectorResult{
				DetectorName: det.Name(),
				Result:       result,
				Error:        err,
				Timestamp:    time.Now(),
			}:
			default:
				// Channel is full, skip
				logger.Warningf("[Updater] Result channel full, dropping result from %s", det.Name())
			}
		}(d)
	}

	// Wait for all detectors to complete
	wg.Wait()

	// Update statistics
	u.updateCount++
	u.lastUpdateTime = time.Now()
}

// processResults processes detector results
func (u *Updater) processResults(ctx context.Context) {
	logger.Info("[Updater] Result processor started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("[Updater] Context cancelled, stopping result processor")
			return

		case <-u.stopChan:
			logger.Info("[Updater] Stop signal received, stopping result processor")
			return

		case result := <-u.resultChan:
			u.handleResult(result)
		}
	}
}

// handleResult handles a single detector result
func (u *Updater) handleResult(result DetectorResult) {
	if result.Error != nil {
		logger.Errorf("[Updater] Detector %s error: %v", result.DetectorName, result.Error)
		return
	}

	// Convert result to string for comparison
	resultStr := fmt.Sprintf("%v", result.Result)

	// Check if result has changed
	u.resultsMu.Lock()
	lastResult, exists := u.lastResults[result.DetectorName]
	shouldLog := !exists || lastResult != resultStr
	if shouldLog {
		u.lastResults[result.DetectorName] = resultStr
	}
	u.resultsMu.Unlock()

	// Only log if result changed
	if shouldLog {
		logger.Infof("[Updater] %s: %v", result.DetectorName, result.Result)
	}

	// TODO: Update UI with result
}

// GetStatistics returns updater statistics
func (u *Updater) GetStatistics() map[string]interface{} {
	u.mu.RLock()
	defer u.mu.RUnlock()

	return map[string]interface{}{
		"running":          u.running,
		"update_count":     u.updateCount,
		"last_update_time": u.lastUpdateTime,
	}
}

// mockCapture is a mock screen capture function for testing
func mockCapture() (image.Image, error) {
	// Return a dummy 1x1 image
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	return img, nil
}
