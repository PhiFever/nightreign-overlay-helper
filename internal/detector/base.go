package detector

import (
	"image"
	"sync"
)

// Detector is the base interface for all detectors
type Detector interface {
	// Name returns the name of the detector
	Name() string

	// Detect performs detection on the given image
	// Returns detection result and any error encountered
	Detect(img image.Image) (interface{}, error)

	// Initialize initializes the detector with necessary resources
	Initialize() error

	// Cleanup releases resources used by the detector
	Cleanup() error

	// IsEnabled returns whether the detector is enabled
	IsEnabled() bool

	// SetEnabled sets the enabled state of the detector
	SetEnabled(enabled bool)
}

// BaseDetector provides common functionality for all detectors
type BaseDetector struct {
	name    string
	enabled bool
	mu      sync.RWMutex
}

// NewBaseDetector creates a new BaseDetector
func NewBaseDetector(name string) *BaseDetector {
	return &BaseDetector{
		name:    name,
		enabled: true,
	}
}

// Name returns the name of the detector
func (b *BaseDetector) Name() string {
	return b.name
}

// IsEnabled returns whether the detector is enabled
func (b *BaseDetector) IsEnabled() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.enabled
}

// SetEnabled sets the enabled state of the detector
func (b *BaseDetector) SetEnabled(enabled bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.enabled = enabled
}

// DetectorRegistry manages all detectors
type DetectorRegistry struct {
	detectors map[string]Detector
	mu        sync.RWMutex
}

// NewDetectorRegistry creates a new detector registry
func NewDetectorRegistry() *DetectorRegistry {
	return &DetectorRegistry{
		detectors: make(map[string]Detector),
	}
}

// Register registers a detector
func (r *DetectorRegistry) Register(detector Detector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.detectors[detector.Name()] = detector
}

// Get retrieves a detector by name
func (r *DetectorRegistry) Get(name string) (Detector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	detector, ok := r.detectors[name]
	return detector, ok
}

// GetAll returns all registered detectors
func (r *DetectorRegistry) GetAll() []Detector {
	r.mu.RLock()
	defer r.mu.RUnlock()

	detectors := make([]Detector, 0, len(r.detectors))
	for _, detector := range r.detectors {
		detectors = append(detectors, detector)
	}
	return detectors
}

// InitializeAll initializes all registered detectors
func (r *DetectorRegistry) InitializeAll() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, detector := range r.detectors {
		if err := detector.Initialize(); err != nil {
			return err
		}
	}
	return nil
}

// CleanupAll cleans up all registered detectors
func (r *DetectorRegistry) CleanupAll() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, detector := range r.detectors {
		if err := detector.Cleanup(); err != nil {
			return err
		}
	}
	return nil
}
