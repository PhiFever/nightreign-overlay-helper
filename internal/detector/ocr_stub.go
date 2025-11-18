//go:build !ocr
// +build !ocr

package detector

import (
	"fmt"
	"image"
)

// OCRAvailable indicates if OCR support is compiled in
const OCRAvailable = false

// OCRResult represents the result of OCR text recognition
type OCRResult struct {
	Text       string
	Confidence float32
	Found      bool
}

// OCRExtractDigits is a stub when OCR is not available
func OCRExtractDigits(img image.Image, language string) (*OCRResult, error) {
	return nil, fmt.Errorf("OCR support not compiled in (use -tags=ocr to enable)")
}

// OCRExtractDayNumber is a stub when OCR is not available
func OCRExtractDayNumber(img image.Image) (int, error) {
	return -1, fmt.Errorf("OCR support not compiled in (use -tags=ocr to enable)")
}
