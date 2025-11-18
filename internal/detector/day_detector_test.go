package detector

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/PhiFever/nightreign-overlay-helper/internal/config"
	"github.com/PhiFever/nightreign-overlay-helper/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Initialize logger with INFO level to avoid blocking
	logger.Setup(logger.INFO)

	// Run tests
	code := m.Run()

	// Exit
	os.Exit(code)
}

// TestDayTemplateLoading tests loading day templates from disk
func TestDayTemplateLoading(t *testing.T) {
	// Get template path
	templateDir := "../../data/day_template"
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		t.Skipf("Template directory not found: %s", templateDir)
	}

	// Test loading all language templates
	languages := []string{"chs", "cht", "eng", "jp"}
	days := []int{1, 2, 3}

	for _, lang := range languages {
		for _, day := range days {
			filename := filepath.Join(templateDir, lang+"_"+string(rune('0'+day))+".png")

			// Try to load the template
			file, err := os.Open(filename)
			require.NoError(t, err, "Failed to open template %s", filename)
			defer file.Close()

			// Decode PNG
			img, err := png.Decode(file)
			require.NoError(t, err, "Failed to decode template %s", filename)
			require.NotNil(t, img, "Template %s decoded to nil image", filename)

			// Verify image has valid dimensions
			bounds := img.Bounds()
			assert.Greater(t, bounds.Dx(), 0, "Template %s has invalid width", filename)
			assert.Greater(t, bounds.Dy(), 0, "Template %s has invalid height", filename)

			t.Logf("✓ Loaded %s: %dx%d", filename, bounds.Dx(), bounds.Dy())
		}
	}
}

// TestDayDetectorInitialization tests detector initialization
func TestDayDetectorInitialization(t *testing.T) {
	// Create a minimal config
	cfg := &config.Config{
		DayPeriodSeconds: []int{270, 180, 210, 180},
		UpdateInterval:   0.1,
	}

	// Create detector
	detector := NewDayDetector(cfg)
	require.NotNil(t, detector, "NewDayDetector should not return nil")

	// Test initialization
	err := detector.Initialize()
	require.NoError(t, err, "Initialize should succeed")

	// Test that detector is enabled by default
	assert.True(t, detector.IsEnabled(), "Detector should be enabled by default")

	// Test cleanup
	err = detector.Cleanup()
	require.NoError(t, err, "Cleanup should succeed")
}

// TestDayDetectorDetect tests the detect method
func TestDayDetectorDetect(t *testing.T) {
	// Create a minimal config
	cfg := &config.Config{
		DayPeriodSeconds: []int{270, 180, 210, 180},
		UpdateInterval:   0.0, // No rate limiting for tests
	}

	// Create detector
	detector := NewDayDetector(cfg)
	require.NotNil(t, detector)

	err := detector.Initialize()
	require.NoError(t, err)
	defer detector.Cleanup()

	// Create a dummy image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// Run detection
	result, err := detector.Detect(img)
	require.NoError(t, err)

	// Check result type
	dayResult, ok := result.(*DayResult)
	require.True(t, ok, "Result should be *DayResult, got %T", result)
	require.NotNil(t, dayResult)

	t.Logf("Detection result: %s", dayResult.String())

	// Test that subsequent calls work correctly
	result2, err := detector.Detect(img)
	require.NoError(t, err)

	dayResult2, ok := result2.(*DayResult)
	require.True(t, ok, "Second result should be *DayResult")
	require.NotNil(t, dayResult2)

	t.Logf("Second detection result: %s", dayResult2.String())
}

// TestDayDetectorCalculateTimes tests time calculation logic
func TestDayDetectorCalculateTimes(t *testing.T) {
	cfg := &config.Config{
		DayPeriodSeconds: []int{270, 180, 210, 180}, // 4.5min, 3min, 3.5min, 3min
		UpdateInterval:   0.1,
	}

	detector := NewDayDetector(cfg)
	require.NotNil(t, detector)

	// Test various day/phase combinations
	testCases := []struct {
		day   int
		phase int
		desc  string
	}{
		{1, 0, "Day 1 Phase 0"},
		{1, 1, "Day 1 Phase 1"},
		{1, 2, "Day 1 Phase 2"},
		{1, 3, "Day 1 Phase 3"},
		{2, 0, "Day 2 Phase 0"},
		{3, 2, "Day 3 Phase 2"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			elapsed, shrink, nextPhase := detector.calculateTimes(tc.day, tc.phase)

			// Verify all durations are non-negative
			assert.GreaterOrEqual(t, elapsed.Seconds(), 0.0, "elapsed time should be non-negative")
			assert.GreaterOrEqual(t, shrink.Seconds(), 0.0, "shrink time should be non-negative")
			assert.GreaterOrEqual(t, nextPhase.Seconds(), 0.0, "next phase time should be non-negative")

			t.Logf("elapsed=%v, shrink=%v, nextPhase=%v", elapsed, shrink, nextPhase)
		})
	}
}

// TestDayDetectorEnableDisable tests enable/disable functionality
func TestDayDetectorEnableDisable(t *testing.T) {
	cfg := &config.Config{
		DayPeriodSeconds: []int{270, 180, 210, 180},
		UpdateInterval:   0.1,
	}

	detector := NewDayDetector(cfg)
	require.NotNil(t, detector)

	// Initially enabled
	assert.True(t, detector.IsEnabled(), "Detector should be enabled by default")

	// Disable
	detector.SetEnabled(false)
	assert.False(t, detector.IsEnabled(), "Detector should be disabled after SetEnabled(false)")

	// Re-enable
	detector.SetEnabled(true)
	assert.True(t, detector.IsEnabled(), "Detector should be enabled after SetEnabled(true)")
}

// BenchmarkDayDetectorDetect benchmarks the detection performance
func BenchmarkDayDetectorDetect(b *testing.B) {
	cfg := &config.Config{
		DayPeriodSeconds: []int{270, 180, 210, 180},
		UpdateInterval:   0.0, // No rate limiting for benchmark
	}

	detector := NewDayDetector(cfg)
	detector.Initialize()
	defer detector.Cleanup()

	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(img)
	}
}
