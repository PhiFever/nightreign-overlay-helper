//go:build ocr
// +build ocr

package detector

import (
	"fmt"
	"image"
	"image/color"
	"regexp"
	"strconv"
	"strings"

	"github.com/otiai10/gosseract/v2"
)

// OCRAvailable indicates if OCR support is compiled in
const OCRAvailable = true

// OCRResult represents the result of OCR text recognition
type OCRResult struct {
	Text       string  // Recognized text
	Confidence float32 // Confidence score (0-100)
	Found      bool    // Whether text was found
}

// OCRExtractDigits extracts digits from an image using Tesseract OCR
func OCRExtractDigits(img image.Image, language string) (*OCRResult, error) {
	// Convert to grayscale
	gray := RGB2Gray(img)

	// Apply adaptive thresholding
	binary := AdaptiveThreshold(gray)

	// Invert for better OCR
	inverted := InvertImage(binary)

	// Initialize Tesseract
	client := gosseract.NewClient()
	defer client.Close()

	client.SetLanguage(language)
	client.SetPageSegMode(gosseract.PSM_SINGLE_LINE)
	client.SetWhitelist("0123456789DAYday ")

	err := client.SetImage(inverted)
	if err != nil {
		return nil, fmt.Errorf("failed to set image: %w", err)
	}

	text, err := client.Text()
	if err != nil {
		return nil, fmt.Errorf("OCR failed: %w", err)
	}

	text = strings.TrimSpace(text)
	digitRegex := regexp.MustCompile(`\d+`)
	matches := digitRegex.FindAllString(text, -1)

	result := &OCRResult{
		Text:       "",
		Confidence: 0,
		Found:      false,
	}

	if len(matches) > 0 {
		result.Text = matches[0]
		result.Found = true
		result.Confidence = 90.0
	}

	return result, nil
}

// OCRExtractDayNumber extracts day number from image
func OCRExtractDayNumber(img image.Image) (int, error) {
	result, err := OCRExtractDigits(img, "eng")
	if err != nil {
		return -1, err
	}

	if !result.Found || result.Text == "" {
		return -1, fmt.Errorf("no day number found")
	}

	digitRegex := regexp.MustCompile(`(\d+)`)
	matches := digitRegex.FindStringSubmatch(result.Text)

	if len(matches) < 2 {
		return -1, fmt.Errorf("no digit in text: %s", result.Text)
	}

	dayNum, err := strconv.Atoi(matches[1])
	if err != nil {
		return -1, fmt.Errorf("invalid day number: %s", matches[1])
	}

	if dayNum < 1 || dayNum > 3 {
		return -1, fmt.Errorf("day number out of range: %d", dayNum)
	}

	return dayNum, nil
}

// Threshold converts grayscale to binary
func Threshold(img *image.Gray, threshold uint8) *image.Gray {
	bounds := img.Bounds()
	binary := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray := img.GrayAt(x, y).Y
			if gray > threshold {
				binary.SetGray(x, y, color.Gray{Y: 255})
			} else {
				binary.SetGray(x, y, color.Gray{Y: 0})
			}
		}
	}

	return binary
}

// AdaptiveThreshold performs Otsu's method thresholding
func AdaptiveThreshold(img *image.Gray) *image.Gray {
	histogram := make([]int, 256)
	bounds := img.Bounds()
	total := bounds.Dx() * bounds.Dy()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			histogram[img.GrayAt(x, y).Y]++
		}
	}

	sum := 0
	for i := 0; i < 256; i++ {
		sum += i * histogram[i]
	}

	sumB := 0
	wB := 0
	wF := 0
	maxVariance := 0.0
	threshold := uint8(0)

	for t := 0; t < 256; t++ {
		wB += histogram[t]
		if wB == 0 {
			continue
		}

		wF = total - wB
		if wF == 0 {
			break
		}

		sumB += t * histogram[t]
		mB := float64(sumB) / float64(wB)
		mF := float64(sum-sumB) / float64(wF)

		variance := float64(wB) * float64(wF) * (mB - mF) * (mB - mF)
		if variance > maxVariance {
			maxVariance = variance
			threshold = uint8(t)
		}
	}

	return Threshold(img, threshold)
}

// InvertImage inverts grayscale image
func InvertImage(img *image.Gray) *image.Gray {
	bounds := img.Bounds()
	inverted := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray := img.GrayAt(x, y).Y
			inverted.SetGray(x, y, color.Gray{Y: 255 - gray})
		}
	}

	return inverted
}
