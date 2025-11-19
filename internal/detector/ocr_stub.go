//go:build !ocr
// +build !ocr

package detector

import (
	"fmt"
	"image"
)

// OCRAvailable 指示是否编译了 OCR 支持
const OCRAvailable = false

// OCRResult 表示 OCR 文本识别的结果
type OCRResult struct {
	Text       string
	Confidence float32
	Found      bool
}

// OCRExtractDigits 是 OCR 不可用时的存根
func OCRExtractDigits(img image.Image, language string) (*OCRResult, error) {
	return nil, fmt.Errorf("OCR support not compiled in (use -tags=ocr to enable)")
}

// OCRExtractDayNumber 是 OCR 不可用时的存根
func OCRExtractDayNumber(img image.Image) (int, error) {
	return -1, fmt.Errorf("OCR support not compiled in (use -tags=ocr to enable)")
}
