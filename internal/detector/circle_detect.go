package detector

import (
	"image"
	"math"
)

// Circle represents a detected circle
type Circle struct {
	X      int     // Center X
	Y      int     // Center Y
	Radius int     // Radius
	Score  float64 // Detection confidence score (0-1)
}

// CircleDetectParams contains parameters for circle detection
type CircleDetectParams struct {
	MinRadius int     // Minimum radius to search for
	MaxRadius int     // Maximum radius to search for
	Threshold float64 // Circularity threshold (0-1, higher = more strict)
}

// DetectCirclesInRegion detects circles in a specific region of the image
// This is a simplified circle detection without full Hough transform
func DetectCirclesInRegion(img image.Image, region Rect, params CircleDetectParams) []Circle {
	// Extract the search region
	searchImg := CropImage(img, region)

	// Convert to grayscale
	gray := RGB2Gray(searchImg)

	// Apply edge detection
	edges := EdgeDetect(gray)

	// Apply threshold to get binary edge map
	binary := ThresholdImage(edges, 50)

	// Find circles by analyzing edge patterns
	circles := findCirclesByEdgeAnalysis(binary, params)

	// Adjust coordinates to original image space
	for i := range circles {
		circles[i].X += region.X
		circles[i].Y += region.Y
	}

	return circles
}

// findCirclesByEdgeAnalysis finds circles by analyzing edge patterns
func findCirclesByEdgeAnalysis(binary *image.Gray, params CircleDetectParams) []Circle {
	bounds := binary.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var circles []Circle

	// Search for circles by testing candidate centers and radii
	// We use a coarse grid to speed up the search
	stepSize := 5 // Test every 5 pixels

	for cy := params.MaxRadius; cy < height-params.MaxRadius; cy += stepSize {
		for cx := params.MaxRadius; cx < width-params.MaxRadius; cx += stepSize {
			// Test different radii
			for r := params.MinRadius; r <= params.MaxRadius; r++ {
				score := calculateCircleScore(binary, cx, cy, r)

				if score >= params.Threshold {
					circle := Circle{
						X:      cx,
						Y:      cy,
						Radius: r,
						Score:  score,
					}

					// Check if this circle overlaps with existing circles
					// If so, keep only the one with higher score
					overlaps := false
					for i, existing := range circles {
						if circlesOverlap(circle, existing) {
							overlaps = true
							if circle.Score > existing.Score {
								circles[i] = circle
							}
							break
						}
					}

					if !overlaps {
						circles = append(circles, circle)
					}
				}
			}
		}
	}

	return circles
}

// calculateCircleScore calculates how well a circle fits the edge pattern
func calculateCircleScore(binary *image.Gray, cx, cy, radius int) float64 {
	bounds := binary.Bounds()

	// Sample points around the circle perimeter
	numSamples := radius * 8 // More samples for larger circles
	if numSamples < 32 {
		numSamples = 32
	}

	edgePixels := 0
	totalSamples := 0

	for i := 0; i < numSamples; i++ {
		angle := float64(i) * 2.0 * math.Pi / float64(numSamples)
		x := cx + int(float64(radius)*math.Cos(angle))
		y := cy + int(float64(radius)*math.Sin(angle))

		// Check if point is within bounds
		if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
			continue
		}

		// Check if this pixel is an edge pixel
		if binary.GrayAt(x, y).Y > 128 {
			edgePixels++
		}
		totalSamples++
	}

	if totalSamples == 0 {
		return 0.0
	}

	// Score is the ratio of edge pixels on the circle perimeter
	return float64(edgePixels) / float64(totalSamples)
}

// circlesOverlap checks if two circles overlap significantly
func circlesOverlap(c1, c2 Circle) bool {
	dx := float64(c1.X - c2.X)
	dy := float64(c1.Y - c2.Y)
	distance := math.Sqrt(dx*dx + dy*dy)

	// Consider circles overlapping if centers are close
	// and radii are similar
	radiusDiff := math.Abs(float64(c1.Radius - c2.Radius))
	avgRadius := float64(c1.Radius+c2.Radius) / 2.0

	return distance < avgRadius*0.5 && radiusDiff < avgRadius*0.3
}

// FindMiniMapCircle finds the minimap circle in the bottom-left region
func FindMiniMapCircle(img image.Image) (*Circle, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Search in bottom-left corner (typical minimap location)
	// Search region: bottom 30%, left 30%
	searchRegion := NewRect(
		0,
		int(float64(height)*0.70),
		int(float64(width)*0.30),
		int(float64(height)*0.30),
	)

	// Expected minimap size: approximately 8-15% of screen height
	minRadius := int(float64(height) * 0.04)
	maxRadius := int(float64(height) * 0.08)

	params := CircleDetectParams{
		MinRadius: minRadius,
		MaxRadius: maxRadius,
		Threshold: 0.35, // 35% of circle perimeter must be edges (relaxed for real images)
	}

	circles := DetectCirclesInRegion(img, searchRegion, params)

	if len(circles) == 0 {
		return nil, nil
	}

	// Return the circle with highest score
	bestCircle := circles[0]
	for _, c := range circles[1:] {
		if c.Score > bestCircle.Score {
			bestCircle = c
		}
	}

	return &bestCircle, nil
}

// CalculateMapRegionFromMiniMap calculates the full map region based on minimap position
func CalculateMapRegionFromMiniMap(screenWidth, screenHeight int, minimap *Circle) Rect {
	if minimap == nil {
		// Fallback to center region if no minimap found
		return getFallbackMapRegion(screenWidth, screenHeight)
	}

	// Minimap is typically in the bottom-left corner of the full map
	// The full map is usually centered on screen with some margin

	// Estimate map size based on minimap size
	// Typical ratio: minimap radius ≈ 10% of map width
	estimatedMapSize := minimap.Radius * 10

	// Clamp to reasonable size (don't exceed screen dimensions)
	maxSize := int(math.Min(float64(screenWidth)*0.8, float64(screenHeight)*0.8))
	if estimatedMapSize > maxSize {
		estimatedMapSize = maxSize
	}

	// Minimap is usually at position (10%, 90%) relative to map region
	// So we can calculate map top-left corner
	mapX := minimap.X - estimatedMapSize/10
	mapY := minimap.Y - estimatedMapSize*9/10

	// Ensure map region is within screen bounds
	if mapX < 0 {
		mapX = 0
	}
	if mapY < 0 {
		mapY = 0
	}

	mapWidth := estimatedMapSize
	mapHeight := estimatedMapSize

	// Ensure map doesn't exceed screen
	if mapX+mapWidth > screenWidth {
		mapWidth = screenWidth - mapX
	}
	if mapY+mapHeight > screenHeight {
		mapHeight = screenHeight - mapY
	}

	return NewRect(mapX, mapY, mapWidth, mapHeight)
}

// getFallbackMapRegion returns a reasonable fallback map region
// This is used when minimap detection fails
func getFallbackMapRegion(screenWidth, screenHeight int) Rect {
	// Common resolutions and their typical map regions
	// For most cases, map is roughly centered with 10-15% margin

	margin := 0.15 // 15% margin on each side
	size := 0.70   // Map takes ~70% of screen dimension

	mapWidth := int(float64(screenWidth) * size)
	mapHeight := int(float64(screenHeight) * size)
	mapX := int(float64(screenWidth) * margin)
	mapY := int(float64(screenHeight) * margin)

	return NewRect(mapX, mapY, mapWidth, mapHeight)
}

// VerifyMapRegion checks if the detected region looks like a map
func VerifyMapRegion(img image.Image, region Rect) bool {
	// Extract the region
	mapImg := CropImage(img, region)

	// Check 1: The region should have significant content
	// (not just empty/uniform color)
	if !hasSignificantContent(mapImg) {
		return false
	}

	// Check 2: Should have map-like characteristics
	// - Multiple distinct colors (terrain types)
	// - Some edge content (paths, boundaries)

	// For now, just check content density
	// Can be enhanced with more sophisticated checks

	return true
}

// hasSignificantContent checks if image has non-uniform content
func hasSignificantContent(img image.Image) bool {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Sample pixels and calculate color variance
	sampleSize := 100
	if width*height < sampleSize {
		sampleSize = width * height
	}

	var rValues, gValues, bValues []float64

	stepX := width / 10
	stepY := height / 10
	if stepX < 1 {
		stepX = 1
	}
	if stepY < 1 {
		stepY = 1
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			r, g, b, _ := img.At(x, y).RGBA()
			rValues = append(rValues, float64(r>>8))
			gValues = append(gValues, float64(g>>8))
			bValues = append(bValues, float64(b>>8))
		}
	}

	// Calculate variance
	rVariance := variance(rValues)
	gVariance := variance(gValues)
	bVariance := variance(bValues)

	// If variance is too low, image is too uniform
	totalVariance := rVariance + gVariance + bVariance

	// Threshold: at least some color variation
	return totalVariance > 1000
}

// variance calculates variance of a slice of float64 values
func variance(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	// Calculate variance
	varianceSum := 0.0
	for _, v := range values {
		diff := v - mean
		varianceSum += diff * diff
	}

	return varianceSum / float64(len(values))
}
