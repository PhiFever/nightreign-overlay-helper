package detector

import (
	"fmt"
	"image"

	"github.com/PhiFever/nightreign-overlay-helper/internal/logger"
)

// MapRegionDetector handles detection of map region from full screen captures
type MapRegionDetector struct {
	lastDetectedRegion *Rect
	lastMinimap        *Circle
}

// NewMapRegionDetector creates a new map region detector
func NewMapRegionDetector() *MapRegionDetector {
	return &MapRegionDetector{}
}

// DetectMapRegion detects the map region from a full screen capture
// Returns the map region rectangle and whether detection was successful
func (d *MapRegionDetector) DetectMapRegion(fullScreenImg image.Image) (Rect, bool) {
	bounds := fullScreenImg.Bounds()
	screenWidth := bounds.Dx()
	screenHeight := bounds.Dy()

	logger.Debug(fmt.Sprintf("[MapRegionDetector] Detecting map region from %dx%d image",
		screenWidth, screenHeight))

	// Step 1: Try to find minimap circle
	minimap, err := FindMiniMapCircle(fullScreenImg)
	if err != nil {
		logger.Warning(fmt.Sprintf("[MapRegionDetector] Error finding minimap: %v", err))
		return d.getFallbackRegion(screenWidth, screenHeight), false
	}

	if minimap == nil {
		logger.Info("[MapRegionDetector] No minimap detected, using fallback region")
		return d.getFallbackRegion(screenWidth, screenHeight), false
	}

	logger.Info(fmt.Sprintf("[MapRegionDetector] Minimap detected: X=%d, Y=%d, R=%d, Score=%.4f",
		minimap.X, minimap.Y, minimap.Radius, minimap.Score))

	// Step 2: Calculate map region based on minimap position
	mapRegion := CalculateMapRegionFromMiniMap(screenWidth, screenHeight, minimap)

	logger.Info(fmt.Sprintf("[MapRegionDetector] Calculated map region: X=%d, Y=%d, W=%d, H=%d",
		mapRegion.X, mapRegion.Y, mapRegion.Width, mapRegion.Height))

	// Step 3: Verify the region looks like a map
	if !VerifyMapRegion(fullScreenImg, mapRegion) {
		logger.Warning("[MapRegionDetector] Map region verification failed, using fallback")
		return d.getFallbackRegion(screenWidth, screenHeight), false
	}

	// Cache successful detection
	d.lastDetectedRegion = &mapRegion
	d.lastMinimap = minimap

	return mapRegion, true
}

// ExtractMapRegion extracts the map region from a full screen image
func (d *MapRegionDetector) ExtractMapRegion(fullScreenImg image.Image) (image.Image, bool) {
	region, success := d.DetectMapRegion(fullScreenImg)
	if !success {
		logger.Warning("[MapRegionDetector] Map detection not successful, using fallback region")
	}

	// Extract the region
	mapImg := CropImage(fullScreenImg, region)
	return mapImg, success
}

// GetLastDetectedRegion returns the last successfully detected region
func (d *MapRegionDetector) GetLastDetectedRegion() *Rect {
	return d.lastDetectedRegion
}

// GetLastMinimap returns the last detected minimap circle
func (d *MapRegionDetector) GetLastMinimap() *Circle {
	return d.lastMinimap
}

// getFallbackRegion returns a fallback region when detection fails
func (d *MapRegionDetector) getFallbackRegion(screenWidth, screenHeight int) Rect {
	// Try to use cached region if available
	if d.lastDetectedRegion != nil {
		logger.Debug("[MapRegionDetector] Using cached region")
		return *d.lastDetectedRegion
	}

	// Otherwise use center-based fallback
	return CalculateMapRegionFromMiniMap(screenWidth, screenHeight, nil)
}

// DetectMapRegionWithPreset tries preset regions first, then auto-detection
func (d *MapRegionDetector) DetectMapRegionWithPreset(fullScreenImg image.Image, presetRegions []Rect) (Rect, bool) {
	// Try each preset region
	for _, preset := range presetRegions {
		// Verify this preset looks like a map
		if VerifyMapRegion(fullScreenImg, preset) {
			logger.Info(fmt.Sprintf("[MapRegionDetector] Using preset region: X=%d, Y=%d, W=%d, H=%d",
				preset.X, preset.Y, preset.Width, preset.Height))
			d.lastDetectedRegion = &preset
			return preset, true
		}
	}

	// Presets failed, try auto-detection
	logger.Info("[MapRegionDetector] Preset regions failed, trying auto-detection")
	return d.DetectMapRegion(fullScreenImg)
}

// GetPresetRegionsForResolution returns preset map regions for common resolutions
func GetPresetRegionsForResolution(width, height int) []Rect {
	// Common resolution presets
	// These are typical map positions for different resolutions
	// Format: X, Y, Width, Height (as percentages that will be converted)

	var presets []Rect

	// Preset 1: Centered with 15% margin
	presets = append(presets, NewRect(
		int(float64(width)*0.15),
		int(float64(height)*0.15),
		int(float64(width)*0.70),
		int(float64(height)*0.70),
	))

	// Preset 2: Centered with 10% margin
	presets = append(presets, NewRect(
		int(float64(width)*0.10),
		int(float64(height)*0.10),
		int(float64(width)*0.80),
		int(float64(height)*0.80),
	))

	// Preset 3: Slightly offset to bottom-left
	presets = append(presets, NewRect(
		int(float64(width)*0.12),
		int(float64(height)*0.12),
		int(float64(width)*0.75),
		int(float64(height)*0.75),
	))

	return presets
}
