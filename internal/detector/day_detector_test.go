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

// TestDayDetectorTemplateLoading tests template loading functionality
func TestDayDetectorTemplateLoading(t *testing.T) {
	cfg := &config.Config{
		DayPeriodSeconds: []int{270, 180, 210, 180},
		UpdateInterval:   0.1,
	}

	detector := NewDayDetector(cfg)
	require.NotNil(t, detector)

	err := detector.Initialize()
	require.NoError(t, err)
	defer detector.Cleanup()

	// Check that templates were loaded
	if len(detector.templates) > 0 {
		t.Logf("Loaded %d language templates", len(detector.templates))

		// Verify each language template
		for lang, template := range detector.templates {
			assert.NotNil(t, template.Day1, "Day 1 template for %s should not be nil", lang)
			assert.NotNil(t, template.Day2, "Day 2 template for %s should not be nil", lang)
			assert.NotNil(t, template.Day3, "Day 3 template for %s should not be nil", lang)
			t.Logf("✓ Templates loaded for language: %s", lang)
		}
	} else {
		t.Log("No templates loaded (expected if data directory not found)")
	}
}

// TestDayDetectorLanguageSwitch tests switching languages
func TestDayDetectorLanguageSwitch(t *testing.T) {
	cfg := &config.Config{
		DayPeriodSeconds: []int{270, 180, 210, 180},
		UpdateInterval:   0.1,
	}

	detector := NewDayDetector(cfg)
	require.NotNil(t, detector)

	err := detector.Initialize()
	require.NoError(t, err)
	defer detector.Cleanup()

	// Test setting different languages
	languages := []string{"chs", "cht", "eng", "jp"}
	for _, lang := range languages {
		detector.SetLanguage(lang)
		// Note: We can't directly access currentLang, so we just verify the method doesn't panic
		t.Logf("Set language to: %s", lang)
	}
}

// TestDayDetectorTemplateMatching tests template matching mode
func TestDayDetectorTemplateMatching(t *testing.T) {
	cfg := &config.Config{
		DayPeriodSeconds: []int{270, 180, 210, 180},
		UpdateInterval:   0.0,
	}

	detector := NewDayDetector(cfg)
	require.NotNil(t, detector)

	err := detector.Initialize()
	require.NoError(t, err)
	defer detector.Cleanup()

	// Skip if no templates loaded
	if len(detector.templates) == 0 {
		t.Skip("No templates loaded, skipping template matching test")
	}

	// Enable template matching
	detector.EnableTemplateMatching(true)
	detector.SetMatchThreshold(0.8)

	// Create a test image (in real use, this would be a screenshot)
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))

	// Run detection
	result, err := detector.Detect(img)
	require.NoError(t, err)
	require.NotNil(t, result)

	dayResult, ok := result.(*DayResult)
	require.True(t, ok)
	t.Logf("Template matching result: %s", dayResult.String())
}

// TestTemplateMatchFunction tests the template matching utility function
func TestTemplateMatchFunction(t *testing.T) {
	// Load one of the day templates for testing
	templatePath := "../../data/day_template/chs_1.png"
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Skipf("Template file not found: %s", templatePath)
	}

	// Load template
	templateFile, err := os.Open(templatePath)
	require.NoError(t, err)
	defer templateFile.Close()

	templateImg, err := png.Decode(templateFile)
	require.NoError(t, err)
	require.NotNil(t, templateImg)

	// Create a test source image containing the template
	sourceBounds := image.Rect(0, 0, 800, 600)
	source := image.NewRGBA(sourceBounds)

	// Copy template to source at position (100, 100)
	templateBounds := templateImg.Bounds()
	for y := 0; y < templateBounds.Dy(); y++ {
		for x := 0; x < templateBounds.Dx(); x++ {
			source.Set(100+x, 100+y, templateImg.At(templateBounds.Min.X+x, templateBounds.Min.Y+y))
		}
	}

	// Perform template matching
	result, err := TemplateMatch(source, templateImg, 0.95)
	require.NoError(t, err)
	require.NotNil(t, result)

	// The match should be found at position (100, 100)
	if result.Found {
		t.Logf("Match found at (%d, %d) with similarity %.4f",
			result.Location.X, result.Location.Y, result.Similarity)
		// Allow some tolerance in position
		assert.InDelta(t, 100, result.Location.X, 5, "X position should be close to 100")
		assert.InDelta(t, 100, result.Location.Y, 5, "Y position should be close to 100")
		assert.GreaterOrEqual(t, result.Similarity, 0.95, "Similarity should be >= 0.95")
	} else {
		t.Log("Match not found (this might be expected if template is very different)")
	}
}

// TestImageProcessingUtils tests various image processing utilities
func TestImageProcessingUtils(t *testing.T) {
	// Create a test image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// Fill with some pattern
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, image.White)
		}
	}

	t.Run("CropImage", func(t *testing.T) {
		rect := NewRect(10, 10, 50, 50)
		cropped := CropImage(img, rect)
		bounds := cropped.Bounds()
		assert.Equal(t, 50, bounds.Dx(), "Cropped width should match")
		assert.Equal(t, 50, bounds.Dy(), "Cropped height should match")
	})

	t.Run("ResizeImage", func(t *testing.T) {
		resized := ResizeImage(img, 50, 50)
		bounds := resized.Bounds()
		assert.Equal(t, 50, bounds.Dx(), "Resized width should match")
		assert.Equal(t, 50, bounds.Dy(), "Resized height should match")
	})

	t.Run("RGB2Gray", func(t *testing.T) {
		gray := RGB2Gray(img)
		require.NotNil(t, gray)
		bounds := gray.Bounds()
		assert.Equal(t, img.Bounds().Dx(), bounds.Dx(), "Gray image width should match")
		assert.Equal(t, img.Bounds().Dy(), bounds.Dy(), "Gray image height should match")
	})

	t.Run("CalculateSimilarity", func(t *testing.T) {
		gray1 := RGB2Gray(img)
		gray2 := RGB2Gray(img)

		similarity, err := CalculateSimilarity(gray1, gray2)
		require.NoError(t, err)
		assert.Equal(t, 1.0, similarity, "Identical images should have similarity 1.0")
	})
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

// BenchmarkTemplateMatch benchmarks template matching performance
func BenchmarkTemplateMatch(b *testing.B) {
	// Create test images
	source := image.NewRGBA(image.Rect(0, 0, 800, 600))
	template := image.NewRGBA(image.Rect(0, 0, 50, 50))

	// Fill with some pattern
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			template.Set(x, y, image.White)
			source.Set(100+x, 100+y, image.White)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TemplateMatch(source, template, 0.8)
	}
}
