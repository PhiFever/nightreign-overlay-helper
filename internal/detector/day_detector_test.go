package detector

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/PhiFever/nightreign-overlay-helper/internal/config"
	"github.com/PhiFever/nightreign-overlay-helper/internal/logger"
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
			if err != nil {
				t.Errorf("Failed to open template %s: %v", filename, err)
				continue
			}
			defer file.Close()

			// Decode PNG
			img, err := png.Decode(file)
			if err != nil {
				t.Errorf("Failed to decode template %s: %v", filename, err)
				continue
			}

			// Verify image is not nil and has valid dimensions
			if img == nil {
				t.Errorf("Template %s decoded to nil image", filename)
				continue
			}

			bounds := img.Bounds()
			if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
				t.Errorf("Template %s has invalid dimensions: %dx%d", filename, bounds.Dx(), bounds.Dy())
			}

			t.Logf("Successfully loaded template %s: %dx%d", filename, bounds.Dx(), bounds.Dy())
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
	if detector == nil {
		t.Fatal("NewDayDetector returned nil")
	}

	// Test initialization
	err := detector.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test that detector is enabled by default
	if !detector.IsEnabled() {
		t.Error("Detector should be enabled by default")
	}

	// Test cleanup
	err = detector.Cleanup()
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
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
	if detector == nil {
		t.Fatal("NewDayDetector returned nil")
	}

	err := detector.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	defer detector.Cleanup()

	// Create a dummy image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// Run detection
	result, err := detector.Detect(img)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// Check result type
	dayResult, ok := result.(*DayResult)
	if !ok {
		t.Fatalf("Result is not *DayResult, got %T", result)
	}

	// Verify result has valid data (even if mock)
	if dayResult == nil {
		t.Fatal("DayResult is nil")
	}

	t.Logf("Detection result: %s", dayResult.String())

	// Test that subsequent calls respect rate limiting
	result2, err := detector.Detect(img)
	if err != nil {
		t.Fatalf("Second Detect failed: %v", err)
	}

	dayResult2, ok := result2.(*DayResult)
	if !ok {
		t.Fatalf("Second result is not *DayResult")
	}

	// Since rate limiting is disabled, results might differ
	t.Logf("Second detection result: %s", dayResult2.String())
}

// TestDayDetectorCalculateTimes tests time calculation logic
func TestDayDetectorCalculateTimes(t *testing.T) {
	cfg := &config.Config{
		DayPeriodSeconds: []int{270, 180, 210, 180}, // 4.5min, 3min, 3.5min, 3min
		UpdateInterval:   0.1,
	}

	detector := NewDayDetector(cfg)
	if detector == nil {
		t.Fatal("NewDayDetector returned nil")
	}

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
		elapsed, shrink, nextPhase := detector.calculateTimes(tc.day, tc.phase)

		// Verify all durations are non-negative
		if elapsed < 0 {
			t.Errorf("%s: elapsed time is negative: %v", tc.desc, elapsed)
		}
		if shrink < 0 {
			t.Errorf("%s: shrink time is negative: %v", tc.desc, shrink)
		}
		if nextPhase < 0 {
			t.Errorf("%s: next phase time is negative: %v", tc.desc, nextPhase)
		}

		t.Logf("%s: elapsed=%v, shrink=%v, nextPhase=%v", tc.desc, elapsed, shrink, nextPhase)
	}
}

// TestDayDetectorEnableDisable tests enable/disable functionality
func TestDayDetectorEnableDisable(t *testing.T) {
	cfg := &config.Config{
		DayPeriodSeconds: []int{270, 180, 210, 180},
		UpdateInterval:   0.1,
	}

	detector := NewDayDetector(cfg)
	if detector == nil {
		t.Fatal("NewDayDetector returned nil")
	}

	// Initially enabled
	if !detector.IsEnabled() {
		t.Error("Detector should be enabled by default")
	}

	// Disable
	detector.SetEnabled(false)
	if detector.IsEnabled() {
		t.Error("Detector should be disabled after SetEnabled(false)")
	}

	// Re-enable
	detector.SetEnabled(true)
	if !detector.IsEnabled() {
		t.Error("Detector should be enabled after SetEnabled(true)")
	}
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
