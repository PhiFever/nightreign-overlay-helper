package screenshot

import (
	"fmt"
	"image"

	"github.com/kbinani/screenshot"
)

// Capturer provides screen capture functionality
type Capturer interface {
	// CaptureScreen captures the entire screen from the specified display
	CaptureScreen(displayIndex int) (image.Image, error)

	// CaptureRegion captures a specific region of the screen
	CaptureRegion(displayIndex int, x, y, width, height int) (image.Image, error)

	// GetDisplayCount returns the number of available displays
	GetDisplayCount() int

	// GetDisplayBounds returns the bounds of the specified display
	GetDisplayBounds(displayIndex int) (image.Rectangle, error)
}

// DefaultCapturer implements screen capture using the screenshot library
type DefaultCapturer struct{}

// NewCapturer creates a new default screen capturer
func NewCapturer() Capturer {
	return &DefaultCapturer{}
}

// CaptureScreen captures the entire screen from the specified display
func (c *DefaultCapturer) CaptureScreen(displayIndex int) (image.Image, error) {
	n := screenshot.NumActiveDisplays()
	if displayIndex < 0 || displayIndex >= n {
		return nil, fmt.Errorf("invalid display index: %d (available: 0-%d)", displayIndex, n-1)
	}

	bounds := screenshot.GetDisplayBounds(displayIndex)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("failed to capture screen: %w", err)
	}

	return img, nil
}

// CaptureRegion captures a specific region of the screen
func (c *DefaultCapturer) CaptureRegion(displayIndex int, x, y, width, height int) (image.Image, error) {
	n := screenshot.NumActiveDisplays()
	if displayIndex < 0 || displayIndex >= n {
		return nil, fmt.Errorf("invalid display index: %d (available: 0-%d)", displayIndex, n-1)
	}

	// Create the rectangle for the region
	rect := image.Rect(x, y, x+width, y+height)

	// Capture the region
	img, err := screenshot.CaptureRect(rect)
	if err != nil {
		return nil, fmt.Errorf("failed to capture region: %w", err)
	}

	return img, nil
}

// GetDisplayCount returns the number of available displays
func (c *DefaultCapturer) GetDisplayCount() int {
	return screenshot.NumActiveDisplays()
}

// GetDisplayBounds returns the bounds of the specified display
func (c *DefaultCapturer) GetDisplayBounds(displayIndex int) (image.Rectangle, error) {
	n := screenshot.NumActiveDisplays()
	if displayIndex < 0 || displayIndex >= n {
		return image.Rectangle{}, fmt.Errorf("invalid display index: %d (available: 0-%d)", displayIndex, n-1)
	}

	bounds := screenshot.GetDisplayBounds(displayIndex)
	return bounds, nil
}
