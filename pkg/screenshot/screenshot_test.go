package screenshot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCapturer tests creating a new capturer
func TestNewCapturer(t *testing.T) {
	capturer := NewCapturer()
	require.NotNil(t, capturer, "NewCapturer should not return nil")
}

// TestGetDisplayCount tests getting display count
func TestGetDisplayCount(t *testing.T) {
	capturer := NewCapturer()
	count := capturer.GetDisplayCount()

	// In headless environment, there might be no displays
	if count == 0 {
		t.Skip("No displays available (headless environment)")
	}

	// Should have at least one display
	assert.GreaterOrEqual(t, count, 1, "Should have at least one display")
	t.Logf("Found %d display(s)", count)
}

// TestGetDisplayBounds tests getting display bounds
func TestGetDisplayBounds(t *testing.T) {
	capturer := NewCapturer()
	count := capturer.GetDisplayCount()

	if count == 0 {
		t.Skip("No displays available for testing")
	}

	// Test valid display index
	bounds, err := capturer.GetDisplayBounds(0)
	require.NoError(t, err, "Getting bounds for display 0 should succeed")
	assert.Greater(t, bounds.Dx(), 0, "Display width should be positive")
	assert.Greater(t, bounds.Dy(), 0, "Display height should be positive")
	t.Logf("Display 0 bounds: %v (size: %dx%d)", bounds, bounds.Dx(), bounds.Dy())

	// Test invalid display index
	_, err = capturer.GetDisplayBounds(-1)
	assert.Error(t, err, "Getting bounds for invalid index should fail")

	_, err = capturer.GetDisplayBounds(count)
	assert.Error(t, err, "Getting bounds for out-of-range index should fail")
}

// TestCaptureScreen tests capturing entire screen
func TestCaptureScreen(t *testing.T) {
	capturer := NewCapturer()
	count := capturer.GetDisplayCount()

	if count == 0 {
		t.Skip("No displays available for testing")
	}

	// Test capturing display 0
	img, err := capturer.CaptureScreen(0)
	require.NoError(t, err, "Capturing screen should succeed")
	require.NotNil(t, img, "Captured image should not be nil")

	bounds := img.Bounds()
	assert.Greater(t, bounds.Dx(), 0, "Captured image width should be positive")
	assert.Greater(t, bounds.Dy(), 0, "Captured image height should be positive")
	t.Logf("Captured screen: %dx%d", bounds.Dx(), bounds.Dy())

	// Test invalid display index
	_, err = capturer.CaptureScreen(-1)
	assert.Error(t, err, "Capturing invalid display should fail")

	_, err = capturer.CaptureScreen(count)
	assert.Error(t, err, "Capturing out-of-range display should fail")
}

// TestCaptureRegion tests capturing a specific region
func TestCaptureRegion(t *testing.T) {
	capturer := NewCapturer()
	count := capturer.GetDisplayCount()

	if count == 0 {
		t.Skip("No displays available for testing")
	}

	// Get display bounds
	displayBounds, err := capturer.GetDisplayBounds(0)
	require.NoError(t, err)

	// Test capturing a 100x100 region
	width, height := 100, 100
	x, y := displayBounds.Min.X, displayBounds.Min.Y

	img, err := capturer.CaptureRegion(0, x, y, width, height)
	require.NoError(t, err, "Capturing region should succeed")
	require.NotNil(t, img, "Captured image should not be nil")

	bounds := img.Bounds()
	assert.Equal(t, width, bounds.Dx(), "Captured region width should match")
	assert.Equal(t, height, bounds.Dy(), "Captured region height should match")
	t.Logf("Captured region: %dx%d at (%d,%d)", width, height, x, y)

	// Test invalid display index
	_, err = capturer.CaptureRegion(-1, x, y, width, height)
	assert.Error(t, err, "Capturing region from invalid display should fail")
}

// BenchmarkCaptureScreen benchmarks screen capture performance
func BenchmarkCaptureScreen(b *testing.B) {
	capturer := NewCapturer()
	count := capturer.GetDisplayCount()

	if count == 0 {
		b.Skip("No displays available for benchmarking")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := capturer.CaptureScreen(0)
		if err != nil {
			b.Fatalf("Capture failed: %v", err)
		}
	}
}

// BenchmarkCaptureRegion benchmarks region capture performance
func BenchmarkCaptureRegion(b *testing.B) {
	capturer := NewCapturer()
	count := capturer.GetDisplayCount()

	if count == 0 {
		b.Skip("No displays available for benchmarking")
	}

	displayBounds, err := capturer.GetDisplayBounds(0)
	if err != nil {
		b.Fatalf("Failed to get display bounds: %v", err)
	}

	x, y := displayBounds.Min.X, displayBounds.Min.Y
	width, height := 100, 100

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := capturer.CaptureRegion(0, x, y, width, height)
		if err != nil {
			b.Fatalf("Capture failed: %v", err)
		}
	}
}
