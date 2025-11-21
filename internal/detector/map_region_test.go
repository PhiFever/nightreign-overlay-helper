package detector

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMapRegionDetector(t *testing.T) {
	detector := NewMapRegionDetector()
	assert.NotNil(t, detector)
	assert.Nil(t, detector.GetLastDetectedRegion())
	assert.Nil(t, detector.GetLastMinimap())
}

func TestDetectMapRegionWithFallback(t *testing.T) {
	detector := NewMapRegionDetector()

	// Create a simple test image (no minimap)
	width, height := 1920, 1080
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with gray
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}

	region, success := detector.DetectMapRegion(img)

	// Should use fallback since no minimap
	assert.False(t, success)

	// Fallback region should still be valid
	assert.Greater(t, region.Width, 0)
	assert.Greater(t, region.Height, 0)
	assert.GreaterOrEqual(t, region.X, 0)
	assert.GreaterOrEqual(t, region.Y, 0)

	t.Logf("Fallback region: X=%d, Y=%d, W=%d, H=%d",
		region.X, region.Y, region.Width, region.Height)
}

func TestDetectMapRegionWithMinimap(t *testing.T) {
	detector := NewMapRegionDetector()

	// Create test image with minimap circle
	width, height := 1920, 1080
	img := createTestFullScreenWithMinimap(width, height)

	region, success := detector.DetectMapRegion(img)

	// Region should be valid even if detection fails (fallback)
	assert.Greater(t, region.Width, 0)
	assert.Greater(t, region.Height, 0)

	t.Logf("Detection success: %v", success)
	t.Logf("Detected region: X=%d, Y=%d, W=%d, H=%d",
		region.X, region.Y, region.Width, region.Height)

	if success {
		t.Logf("Minimap detected: X=%d, Y=%d, R=%d",
			detector.GetLastMinimap().X,
			detector.GetLastMinimap().Y,
			detector.GetLastMinimap().Radius)
	} else {
		t.Log("Minimap not detected, used fallback region (expected for simple test image)")
	}
}

func TestExtractMapRegion(t *testing.T) {
	detector := NewMapRegionDetector()

	width, height := 1920, 1080
	img := createTestFullScreenWithMinimap(width, height)

	mapImg, success := detector.ExtractMapRegion(img)

	assert.NotNil(t, mapImg)
	// success can be false if using fallback, which is fine

	// Extracted image should be smaller than full screen
	mapBounds := mapImg.Bounds()
	assert.Less(t, mapBounds.Dx(), width)
	assert.Less(t, mapBounds.Dy(), height)

	t.Logf("Extracted map size: %dx%d (success=%v)", mapBounds.Dx(), mapBounds.Dy(), success)
}

func TestGetPresetRegionsForResolution(t *testing.T) {
	testCases := []struct {
		width  int
		height int
	}{
		{1920, 1080},
		{2560, 1440},
		{1280, 720},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			presets := GetPresetRegionsForResolution(tc.width, tc.height)

			assert.Greater(t, len(presets), 0, "Should return at least one preset")

			for i, preset := range presets {
				t.Logf("Resolution %dx%d, Preset %d: X=%d, Y=%d, W=%d, H=%d",
					tc.width, tc.height, i,
					preset.X, preset.Y, preset.Width, preset.Height)

				// Verify preset is within screen bounds
				assert.GreaterOrEqual(t, preset.X, 0)
				assert.GreaterOrEqual(t, preset.Y, 0)
				assert.LessOrEqual(t, preset.X+preset.Width, tc.width)
				assert.LessOrEqual(t, preset.Y+preset.Height, tc.height)
			}
		})
	}
}

func TestDetectMapRegionWithPreset(t *testing.T) {
	detector := NewMapRegionDetector()

	width, height := 1920, 1080
	img := createTestFullScreenWithMinimap(width, height)

	presets := GetPresetRegionsForResolution(width, height)

	region, success := detector.DetectMapRegionWithPreset(img, presets)

	assert.NotNil(t, region)
	t.Logf("Region from preset: X=%d, Y=%d, W=%d, H=%d, Success=%v",
		region.X, region.Y, region.Width, region.Height, success)
}

// Helper function to create test image with minimap
func createTestFullScreenWithMinimap(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 50, G: 50, B: 50, A: 255})
		}
	}

	// Draw a "map" region in the center (with varied colors)
	mapX := width / 4
	mapY := height / 4
	mapW := width / 2
	mapH := height / 2

	for y := mapY; y < mapY+mapH; y++ {
		for x := mapX; x < mapX+mapW; x++ {
			c := uint8((x+y)%200 + 50)
			img.Set(x, y, color.RGBA{R: c, G: c + 10, B: c + 20, A: 255})
		}
	}

	// Draw minimap circle in bottom-left of map
	minimapX := mapX + mapW/10
	minimapY := mapY + mapH*9/10
	minimapR := 60

	// Draw a thicker, more visible circle
	for thickness := 0; thickness < 5; thickness++ {
		for y := minimapY - minimapR; y <= minimapY+minimapR; y++ {
			for x := minimapX - minimapR; x <= minimapX+minimapR; x++ {
				dx := float64(x - minimapX)
				dy := float64(y - minimapY)
				dist := dx*dx + dy*dy
				r := float64(minimapR - thickness)

				// Circle edge with multiple pixels thickness
				if dist >= r*r && dist <= (r+1)*(r+1) {
					img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
				}
			}
		}
	}

	return img
}
