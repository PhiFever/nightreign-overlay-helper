package detector

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

// Point represents a 2D point
type Point struct {
	X, Y int
}

// Rect represents a rectangle
type Rect struct {
	X, Y, Width, Height int
}

// NewRect creates a new rectangle
func NewRect(x, y, width, height int) Rect {
	return Rect{X: x, Y: y, Width: width, Height: height}
}

// Contains checks if a point is inside the rectangle
func (r Rect) Contains(p Point) bool {
	return p.X >= r.X && p.X < r.X+r.Width &&
		p.Y >= r.Y && p.Y < r.Y+r.Height
}

// ToImageRect converts to image.Rectangle
func (r Rect) ToImageRect() image.Rectangle {
	return image.Rect(r.X, r.Y, r.X+r.Width, r.Y+r.Height)
}

// CropImage crops an image to the specified rectangle
func CropImage(img image.Image, rect Rect) image.Image {
	bounds := rect.ToImageRect()

	// Create a new image with the cropped size
	cropped := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))

	// Copy pixels
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if x >= img.Bounds().Min.X && x < img.Bounds().Max.X &&
				y >= img.Bounds().Min.Y && y < img.Bounds().Max.Y {
				cropped.Set(x-bounds.Min.X, y-bounds.Min.Y, img.At(x, y))
			}
		}
	}

	return cropped
}

// ResizeImage resizes an image to the specified width and height
// Uses nearest neighbor interpolation for simplicity
func ResizeImage(img image.Image, width, height int) image.Image {
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	resized := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Calculate source coordinates
			srcX := x * srcWidth / width
			srcY := y * srcHeight / height

			// Get color from source image
			c := img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY)
			resized.Set(x, y, c)
		}
	}

	return resized
}

// RGB2Gray converts an RGB image to grayscale
func RGB2Gray(img image.Image) *image.Gray {
	bounds := img.Bounds()
	gray := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// Convert to 8-bit values
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)
			// Calculate grayscale using standard formula
			grayValue := uint8(0.299*float64(r8) + 0.587*float64(g8) + 0.114*float64(b8))
			gray.SetGray(x, y, color.Gray{Y: grayValue})
		}
	}

	return gray
}

// RGB2HSV converts RGB color to HSV
func RGB2HSV(r, g, b uint8) (h, s, v float64) {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	delta := max - min

	// Value
	v = max

	// Saturation
	if max == 0 {
		s = 0
	} else {
		s = delta / max
	}

	// Hue
	if delta == 0 {
		h = 0
	} else {
		switch max {
		case rf:
			h = 60 * (((gf - bf) / delta) + 0)
			if h < 0 {
				h += 360
			}
		case gf:
			h = 60 * (((bf - rf) / delta) + 2)
		case bf:
			h = 60 * (((rf - gf) / delta) + 4)
		}
	}

	return h, s, v
}

// RGB2HLS converts RGB color to HLS (Hue, Lightness, Saturation)
func RGB2HLS(r, g, b uint8) (h, l, s float64) {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	delta := max - min

	// Lightness
	l = (max + min) / 2.0

	// Saturation
	if delta == 0 {
		s = 0
	} else {
		if l < 0.5 {
			s = delta / (max + min)
		} else {
			s = delta / (2.0 - max - min)
		}
	}

	// Hue
	if delta == 0 {
		h = 0
	} else {
		switch max {
		case rf:
			h = ((gf - bf) / delta)
			if gf < bf {
				h += 6
			}
		case gf:
			h = ((bf - rf) / delta) + 2
		case bf:
			h = ((rf - gf) / delta) + 4
		}
		h *= 60
	}

	return h, l, s
}

// CountNonZero counts non-zero pixels in a grayscale image
func CountNonZero(img *image.Gray) int {
	count := 0
	bounds := img.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.GrayAt(x, y).Y > 0 {
				count++
			}
		}
	}

	return count
}

// InRange checks if a color is within the specified range
func InRange(c color.Color, lower, upper [3]uint8) bool {
	r, g, b, _ := c.RGBA()
	r8 := uint8(r >> 8)
	g8 := uint8(g >> 8)
	b8 := uint8(b >> 8)

	return r8 >= lower[0] && r8 <= upper[0] &&
		g8 >= lower[1] && g8 <= upper[1] &&
		b8 >= lower[2] && b8 <= upper[2]
}

// CreateMask creates a binary mask based on color range
func CreateMask(img image.Image, lower, upper [3]uint8) *image.Gray {
	bounds := img.Bounds()
	mask := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			if InRange(c, lower, upper) {
				mask.SetGray(x, y, color.Gray{Y: 255})
			} else {
				mask.SetGray(x, y, color.Gray{Y: 0})
			}
		}
	}

	return mask
}

// CalculateSimilarity calculates similarity between two grayscale images
// Returns a value between 0 and 1, where 1 means identical
func CalculateSimilarity(img1, img2 *image.Gray) (float64, error) {
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()

	if bounds1.Dx() != bounds2.Dx() || bounds1.Dy() != bounds2.Dy() {
		return 0, fmt.Errorf("images must have the same dimensions")
	}

	totalPixels := bounds1.Dx() * bounds1.Dy()
	if totalPixels == 0 {
		return 0, fmt.Errorf("image has zero pixels")
	}

	sumSquaredDiff := 0.0

	for y := 0; y < bounds1.Dy(); y++ {
		for x := 0; x < bounds1.Dx(); x++ {
			v1 := float64(img1.GrayAt(bounds1.Min.X+x, bounds1.Min.Y+y).Y)
			v2 := float64(img2.GrayAt(bounds2.Min.X+x, bounds2.Min.Y+y).Y)
			diff := v1 - v2
			sumSquaredDiff += diff * diff
		}
	}

	// Calculate MSE (Mean Squared Error)
	mse := sumSquaredDiff / float64(totalPixels)

	// Convert MSE to similarity score (0-1)
	// Max MSE is 255^2 = 65025
	maxMSE := 255.0 * 255.0
	similarity := 1.0 - (mse / maxMSE)

	return similarity, nil
}

// MatchResult represents the result of template matching
type MatchResult struct {
	Location   Point   // Top-left corner of the match
	Similarity float64 // Similarity score (0-1)
	Found      bool    // Whether a match was found
}

// TemplateMatch performs template matching on a source image
// Returns the location and similarity of the best match
func TemplateMatch(source, template image.Image, threshold float64) (*MatchResult, error) {
	return TemplateMatchWithStride(source, template, threshold, 1)
}

// TemplateMatchWithStride performs template matching with configurable stride for speed
// stride > 1 performs a coarse search first, then refines around the best match
func TemplateMatchWithStride(source, template image.Image, threshold float64, stride int) (*MatchResult, error) {
	// Convert images to grayscale
	srcGray := RGB2Gray(source)
	tmplGray := RGB2Gray(template)

	srcBounds := srcGray.Bounds()
	tmplBounds := tmplGray.Bounds()

	tmplWidth := tmplBounds.Dx()
	tmplHeight := tmplBounds.Dy()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	if tmplWidth > srcWidth || tmplHeight > srcHeight {
		return &MatchResult{Found: false}, fmt.Errorf("template is larger than source image")
	}

	bestMatch := &MatchResult{
		Similarity: 0.0,
		Found:      false,
	}

	// OPTIMIZATION: Use stride for coarse search
	if stride < 1 {
		stride = 1
	}

	// Coarse search with stride
	for y := 0; y <= srcHeight-tmplHeight; y += stride {
		for x := 0; x <= srcWidth-tmplWidth; x += stride {
			// Extract region of interest from source
			roi := extractROI(srcGray, x, y, tmplWidth, tmplHeight)

			// Calculate similarity
			similarity, err := CalculateSimilarity(roi, tmplGray)
			if err != nil {
				continue
			}

			// Update best match
			if similarity > bestMatch.Similarity {
				bestMatch.Similarity = similarity
				bestMatch.Location = Point{X: x, Y: y}
			}
		}
	}

	// If stride > 1 and we found a good candidate, refine search around it
	if stride > 1 && bestMatch.Similarity > threshold*0.9 {
		refineX := bestMatch.Location.X
		refineY := bestMatch.Location.Y

		// Search in a small region around the best match with stride=1
		startX := max(0, refineX-stride)
		startY := max(0, refineY-stride)
		endX := min(srcWidth-tmplWidth, refineX+stride)
		endY := min(srcHeight-tmplHeight, refineY+stride)

		for y := startY; y <= endY; y++ {
			for x := startX; x <= endX; x++ {
				roi := extractROI(srcGray, x, y, tmplWidth, tmplHeight)
				similarity, err := CalculateSimilarity(roi, tmplGray)
				if err != nil {
					continue
				}

				if similarity > bestMatch.Similarity {
					bestMatch.Similarity = similarity
					bestMatch.Location = Point{X: x, Y: y}
				}
			}
		}
	}

	// Check if we found a match above threshold
	if bestMatch.Similarity >= threshold {
		bestMatch.Found = true
	}

	return bestMatch, nil
}

// extractROI extracts a region of interest from a grayscale image
func extractROI(img *image.Gray, x, y, width, height int) *image.Gray {
	bounds := img.Bounds()
	roi := image.NewGray(image.Rect(0, 0, width, height))

	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			srcX := bounds.Min.X + x + dx
			srcY := bounds.Min.Y + y + dy

			if srcX < bounds.Max.X && srcY < bounds.Max.Y {
				roi.SetGray(dx, dy, img.GrayAt(srcX, srcY))
			}
		}
	}

	return roi
}

// TemplateMatchMultiple matches a template against multiple source regions
// Returns the best match across all regions
func TemplateMatchMultiple(source image.Image, template image.Image, regions []Rect, threshold float64) (*MatchResult, error) {
	bestMatch := &MatchResult{
		Similarity: 0.0,
		Found:      false,
	}

	for _, region := range regions {
		// Crop source to region
		cropped := CropImage(source, region)

		// Perform template matching
		result, err := TemplateMatch(cropped, template, threshold)
		if err != nil {
			continue
		}

		// Adjust location to account for region offset
		if result.Found && result.Similarity > bestMatch.Similarity {
			bestMatch.Similarity = result.Similarity
			bestMatch.Location = Point{
				X: region.X + result.Location.X,
				Y: region.Y + result.Location.Y,
			}
			bestMatch.Found = true
		}
	}

	return bestMatch, nil
}

// ColorRange represents a color range for filtering
type ColorRange struct {
	Lower [3]uint8 // RGB lower bounds
	Upper [3]uint8 // RGB upper bounds
}

// HasBrightPixels checks if a region has enough bright/white pixels (for text detection)
// Uses sampling to improve performance
func HasBrightPixels(img image.Image, region Rect, threshold float64, sampleStep int) bool {
	if sampleStep < 1 {
		sampleStep = 1
	}

	brightCount := 0
	totalSamples := 0

	bounds := img.Bounds()
	startX := max(region.X, bounds.Min.X)
	startY := max(region.Y, bounds.Min.Y)
	endX := min(region.X+region.Width, bounds.Max.X)
	endY := min(region.Y+region.Height, bounds.Max.Y)

	for y := startY; y < endY; y += sampleStep {
		for x := startX; x < endX; x += sampleStep {
			r, g, b, _ := img.At(x, y).RGBA()
			// Convert to 8-bit
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			// Check if pixel is bright (white or light colored)
			if r8 > 200 && g8 > 200 && b8 > 200 {
				brightCount++
			}
			totalSamples++
		}
	}

	if totalSamples == 0 {
		return false
	}

	ratio := float64(brightCount) / float64(totalSamples)
	return ratio >= threshold
}

// FindCandidateRegions finds potential text regions using color-based filtering
// This is much faster than template matching and can narrow down search areas
func FindCandidateRegions(img image.Image, windowWidth, windowHeight, stepSize int, brightThreshold float64) []Rect {
	candidates := []Rect{}
	bounds := img.Bounds()

	// Scan image with sliding window
	for y := bounds.Min.Y; y < bounds.Max.Y-windowHeight; y += stepSize {
		for x := bounds.Min.X; x < bounds.Max.X-windowWidth; x += stepSize {
			region := NewRect(x, y, windowWidth, windowHeight)

			// Quick color check with sampling
			if HasBrightPixels(img, region, brightThreshold, 5) {
				candidates = append(candidates, region)
			}
		}
	}

	return candidates
}

// TemplateMatchPyramid performs multi-scale template matching using image pyramid
// This is faster than full-resolution matching for large images
func TemplateMatchPyramid(source, template image.Image, threshold float64, scales []float64) (*MatchResult, error) {
	if len(scales) == 0 {
		scales = []float64{0.25, 0.5, 1.0} // Default scales
	}

	bestMatch := &MatchResult{
		Similarity: 0.0,
		Found:      false,
	}

	srcBounds := source.Bounds()
	tmplBounds := template.Bounds()

	for i, scale := range scales {
		// Resize images according to scale
		scaledSrcW := int(float64(srcBounds.Dx()) * scale)
		scaledSrcH := int(float64(srcBounds.Dy()) * scale)
		scaledTmplW := int(float64(tmplBounds.Dx()) * scale)
		scaledTmplH := int(float64(tmplBounds.Dy()) * scale)

		if scaledSrcW < scaledTmplW || scaledSrcH < scaledTmplH {
			continue
		}

		scaledSrc := ResizeImage(source, scaledSrcW, scaledSrcH)
		scaledTmpl := ResizeImage(template, scaledTmplW, scaledTmplH)

		// For coarse scales, use lower threshold
		adjustedThreshold := threshold
		if i < len(scales)-1 {
			adjustedThreshold = threshold * 0.85 // Lower threshold for coarse search
		}

		// Perform template matching at this scale
		result, err := TemplateMatch(scaledSrc, scaledTmpl, adjustedThreshold)
		if err != nil {
			continue
		}

		if result.Found {
			// Scale location back to original coordinates
			result.Location.X = int(float64(result.Location.X) / scale)
			result.Location.Y = int(float64(result.Location.Y) / scale)

			if i == len(scales)-1 {
				// Last scale (full resolution), return directly
				return result, nil
			}

			// For coarse scales, refine the search in next iteration
			// by focusing on the found region
			if result.Similarity > bestMatch.Similarity {
				bestMatch = result
			}
		}
	}

	return bestMatch, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
