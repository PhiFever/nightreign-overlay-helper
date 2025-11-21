package detector

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// createTestCircleImage creates a test image with a circle
func createTestCircleImage(width, height, cx, cy, radius int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill background with gray
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{128, 128, 128, 255})
		}
	}

	// Draw a white circle
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dx := float64(x - cx)
			dy := float64(y - cy)
			distance := math.Sqrt(dx*dx + dy*dy)

			// Draw circle edge (thickness ~2 pixels)
			if math.Abs(distance-float64(radius)) < 2.0 {
				img.Set(x, y, color.RGBA{255, 255, 255, 255})
			}
		}
	}

	return img
}

func TestCirclesOverlap(t *testing.T) {
	tests := []struct {
		name     string
		c1       Circle
		c2       Circle
		expected bool
	}{
		{
			name:     "Identical circles",
			c1:       Circle{X: 100, Y: 100, Radius: 50},
			c2:       Circle{X: 100, Y: 100, Radius: 50},
			expected: true,
		},
		{
			name:     "Overlapping circles",
			c1:       Circle{X: 100, Y: 100, Radius: 50},
			c2:       Circle{X: 110, Y: 100, Radius: 52},
			expected: true,
		},
		{
			name:     "Non-overlapping circles",
			c1:       Circle{X: 100, Y: 100, Radius: 50},
			c2:       Circle{X: 200, Y: 100, Radius: 50},
			expected: false,
		},
		{
			name:     "Different sizes, far apart",
			c1:       Circle{X: 100, Y: 100, Radius: 30},
			c2:       Circle{X: 200, Y: 200, Radius: 60},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := circlesOverlap(tt.c1, tt.c2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateCircleScore(t *testing.T) {
	// Create a test image with a circle
	width, height := 200, 200
	cx, cy, radius := 100, 100, 40

	img := createTestCircleImage(width, height, cx, cy, radius)
	gray := RGB2Gray(img)
	edges := EdgeDetect(gray)
	binary := ThresholdImage(edges, 50)

	// Test 1: Score should be high for the actual circle
	score := calculateCircleScore(binary, cx, cy, radius)
	t.Logf("Circle score at correct position: %.4f", score)
	assert.Greater(t, score, 0.3, "Score should be > 0.3 for actual circle")

	// Test 2: Score should be low for wrong position
	wrongScore := calculateCircleScore(binary, cx+20, cy+20, radius)
	t.Logf("Circle score at wrong position: %.4f", wrongScore)
	assert.Less(t, wrongScore, score*0.7, "Score should be lower at wrong position")

	// Test 3: Score should be low for wrong radius
	wrongRadiusScore := calculateCircleScore(binary, cx, cy, radius+20)
	t.Logf("Circle score with wrong radius: %.4f", wrongRadiusScore)
	assert.Less(t, wrongRadiusScore, score*0.7, "Score should be lower with wrong radius")
}

func TestDetectCirclesInRegion(t *testing.T) {
	// Create a test image with a circle in the bottom-left
	width, height := 400, 400
	cx, cy, radius := 80, 320, 40

	img := createTestCircleImage(width, height, cx, cy, radius)

	// Define search region (bottom-left quarter)
	searchRegion := NewRect(0, height/2, width/2, height/2)

	params := CircleDetectParams{
		MinRadius: 35,
		MaxRadius: 45,
		Threshold: 0.5,
	}

	circles := DetectCirclesInRegion(img, searchRegion, params)

	assert.Greater(t, len(circles), 0, "Should detect at least one circle")

	if len(circles) > 0 {
		detected := circles[0]
		t.Logf("Detected circle: X=%d, Y=%d, R=%d, Score=%.4f",
			detected.X, detected.Y, detected.Radius, detected.Score)

		// Check if detected circle is close to actual circle
		dx := detected.X - cx
		dy := detected.Y - cy
		dr := detected.Radius - radius

		assert.Less(t, abs(dx), 10, "X position should be within 10 pixels")
		assert.Less(t, abs(dy), 10, "Y position should be within 10 pixels")
		assert.Less(t, abs(dr), 5, "Radius should be within 5 pixels")
	}
}

func TestCalculateMapRegionFromMiniMap(t *testing.T) {
	screenWidth, screenHeight := 1920, 1080

	// Test with minimap detected
	minimap := &Circle{
		X:      150,
		Y:      950,
		Radius: 50,
	}

	region := CalculateMapRegionFromMiniMap(screenWidth, screenHeight, minimap)

	t.Logf("Map region: X=%d, Y=%d, W=%d, H=%d",
		region.X, region.Y, region.Width, region.Height)

	// Verify region is within screen bounds
	assert.GreaterOrEqual(t, region.X, 0, "Region X should be >= 0")
	assert.GreaterOrEqual(t, region.Y, 0, "Region Y should be >= 0")
	assert.LessOrEqual(t, region.X+region.Width, screenWidth, "Region should fit in screen width")
	assert.LessOrEqual(t, region.Y+region.Height, screenHeight, "Region should fit in screen height")

	// Verify region is reasonable size
	assert.Greater(t, region.Width, 300, "Map should be at least 300px wide")
	assert.Greater(t, region.Height, 300, "Map should be at least 300px tall")
}

func TestCalculateMapRegionFallback(t *testing.T) {
	screenWidth, screenHeight := 1920, 1080

	// Test without minimap (fallback mode)
	region := CalculateMapRegionFromMiniMap(screenWidth, screenHeight, nil)

	t.Logf("Fallback map region: X=%d, Y=%d, W=%d, H=%d",
		region.X, region.Y, region.Width, region.Height)

	// Verify region is within screen bounds
	assert.GreaterOrEqual(t, region.X, 0)
	assert.GreaterOrEqual(t, region.Y, 0)
	assert.LessOrEqual(t, region.X+region.Width, screenWidth)
	assert.LessOrEqual(t, region.Y+region.Height, screenHeight)

	// Fallback should use centered region
	assert.Greater(t, region.X, 0, "Should have left margin")
	assert.Greater(t, region.Y, 0, "Should have top margin")
}

func TestHasSignificantContent(t *testing.T) {
	// Test 1: Uniform image (should fail)
	uniformImg := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			uniformImg.Set(x, y, color.RGBA{128, 128, 128, 255})
		}
	}
	assert.False(t, hasSignificantContent(uniformImg), "Uniform image should not have significant content")

	// Test 2: Varied image (should pass)
	variedImg := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			// Create a gradient
			c := uint8((x + y) % 256)
			variedImg.Set(x, y, color.RGBA{c, c, c, 255})
		}
	}
	assert.True(t, hasSignificantContent(variedImg), "Varied image should have significant content")
}

func TestVariance(t *testing.T) {
	// Test 1: No variance
	uniform := []float64{5.0, 5.0, 5.0, 5.0}
	v := variance(uniform)
	assert.Equal(t, 0.0, v, "Uniform values should have 0 variance")

	// Test 2: Some variance
	varied := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	v = variance(varied)
	assert.Greater(t, v, 0.0, "Varied values should have positive variance")
	t.Logf("Variance of [1,2,3,4,5]: %.4f", v)

	// Test 3: Empty slice
	empty := []float64{}
	v = variance(empty)
	assert.Equal(t, 0.0, v, "Empty slice should have 0 variance")
}

// Helper function
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Benchmark tests
func BenchmarkCalculateCircleScore(b *testing.B) {
	width, height := 200, 200
	cx, cy, radius := 100, 100, 40

	img := createTestCircleImage(width, height, cx, cy, radius)
	gray := RGB2Gray(img)
	edges := EdgeDetect(gray)
	binary := ThresholdImage(edges, 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateCircleScore(binary, cx, cy, radius)
	}
}

func BenchmarkDetectCirclesInRegion(b *testing.B) {
	width, height := 400, 400
	cx, cy, radius := 80, 320, 40

	img := createTestCircleImage(width, height, cx, cy, radius)
	searchRegion := NewRect(0, height/2, width/2, height/2)
	params := CircleDetectParams{
		MinRadius: 35,
		MaxRadius: 45,
		Threshold: 0.3,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectCirclesInRegion(img, searchRegion, params)
	}
}
