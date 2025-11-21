package screenshot

import (
	"fmt"
	"image"

	"github.com/kbinani/screenshot"
)

// Capturer 提供屏幕捕获功能
type Capturer interface {
	// CaptureScreen 从指定的显示器捕获整个屏幕
	CaptureScreen(displayIndex int) (image.Image, error)

	// CaptureRegion 捕获屏幕的特定区域
	CaptureRegion(displayIndex int, x, y, width, height int) (image.Image, error)

	// GetDisplayCount 返回可用显示器的数量
	GetDisplayCount() int

	// GetDisplayBounds 返回指定显示器的边界
	GetDisplayBounds(displayIndex int) (image.Rectangle, error)
}

// DefaultCapturer 使用 screenshot 库实现屏幕捕获
type DefaultCapturer struct{}

// NewCapturer 创建一个新的默认屏幕捕获器
func NewCapturer() Capturer {
	return &DefaultCapturer{}
}

// CaptureScreen 从指定的显示器捕获整个屏幕
func (c *DefaultCapturer) CaptureScreen(displayIndex int) (image.Image, error) {
	n := screenshot.NumActiveDisplays()
	if displayIndex < 0 || displayIndex >= n {
		return nil, fmt.Errorf("invalid display index: %d (available: 0-%d)", displayIndex, n-1)
	}

	bounds := screenshot.GetDisplayBounds(displayIndex)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("failed to capture screen: %w", err)
	}

	return img, nil
}

// CaptureRegion 捕获屏幕的特定区域
func (c *DefaultCapturer) CaptureRegion(displayIndex int, x, y, width, height int) (image.Image, error) {
	n := screenshot.NumActiveDisplays()
	if displayIndex < 0 || displayIndex >= n {
		return nil, fmt.Errorf("invalid display index: %d (available: 0-%d)", displayIndex, n-1)
	}

	// 创建区域的矩形
	rect := image.Rect(x, y, x+width, y+height)

	// 捕获区域
	img, err := screenshot.CaptureRect(rect)
	if err != nil {
		return nil, fmt.Errorf("failed to capture region: %w", err)
	}

	return img, nil
}

// GetDisplayCount 返回可用显示器的数量
func (c *DefaultCapturer) GetDisplayCount() int {
	return screenshot.NumActiveDisplays()
}

// GetDisplayBounds 返回指定显示器的边界
func (c *DefaultCapturer) GetDisplayBounds(displayIndex int) (image.Rectangle, error) {
	n := screenshot.NumActiveDisplays()
	if displayIndex < 0 || displayIndex >= n {
		return image.Rectangle{}, fmt.Errorf("invalid display index: %d (available: 0-%d)", displayIndex, n-1)
	}

	bounds := screenshot.GetDisplayBounds(displayIndex)
	return bounds, nil
}
