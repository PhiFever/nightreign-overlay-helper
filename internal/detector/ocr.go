//go:build ocr
// +build ocr

package detector

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/otiai10/gosseract/v2"
)

// OCRAvailable 指示是否编译了 OCR 支持
const OCRAvailable = true

// OCRResult 表示 OCR 文本识别的结果
type OCRResult struct {
	Text       string  // 识别的文本
	Confidence float32 // 置信度分数（0-100）
	Found      bool    // 是否找到文本
}

// OCRExtractDigits 使用 Tesseract OCR 从图像中提取数字
func OCRExtractDigits(img image.Image, language string) (*OCRResult, error) {
	// 转换为灰度图
	gray := RGB2Gray(img)

	// 应用自适应阈值化
	binary := AdaptiveThreshold(gray)

	// 反转以获得更好的 OCR 效果
	inverted := InvertImage(binary)

	// 初始化 Tesseract
	client := gosseract.NewClient()
	defer client.Close()

	client.SetLanguage(language)
	client.SetPageSegMode(gosseract.PSM_SINGLE_LINE)
	client.SetWhitelist("0123456789DAYday ")

	// 保存图像到临时文件供 Tesseract 使用
	tmpfile, err := ioutil.TempFile("", "ocr-*.png")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	// 将图像编码为 PNG
	if err := png.Encode(tmpfile, inverted); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	// 从文件设置图像
	if err := client.SetImage(tmpfile.Name()); err != nil {
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

// OCRExtractDayNumber 从图像中提取天数（识别罗马数字 I/II/III）
func OCRExtractDayNumber(img image.Image) (int, error) {
	bounds := img.Bounds()

	// 调试：保存原始图像
	debugDir := "/tmp/ocr_debug"
	os.MkdirAll(debugDir, 0755)

	// 转换为灰度图
	gray := RGB2Gray(img)

	// 保存灰度图
	if f, err := os.Create(fmt.Sprintf("%s/01_gray_%dx%d.png", debugDir, bounds.Dx(), bounds.Dy())); err == nil {
		png.Encode(f, gray)
		f.Close()
	}

	// 应用自适应阈值化
	binary := AdaptiveThreshold(gray)

	// 保存二值图
	if f, err := os.Create(fmt.Sprintf("%s/02_binary_%dx%d.png", debugDir, bounds.Dx(), bounds.Dy())); err == nil {
		png.Encode(f, binary)
		f.Close()
	}

	// 尝试两种方式：不反转和反转
	// 游戏中的罗马数字可能是黑底白字（不需要反转）
	testImages := []struct {
		name string
		img  image.Image
	}{
		{"normal", binary},
		{"inverted", InvertImage(binary)},
	}

	for _, testImg := range testImages {
		// 保存测试图像
		if f, err := os.Create(fmt.Sprintf("%s/03_%s_%dx%d.png", debugDir, testImg.name, bounds.Dx(), bounds.Dy())); err == nil {
			png.Encode(f, testImg.img)
			f.Close()
		}

		// 初始化 Tesseract
		client := gosseract.NewClient()

		client.SetLanguage("eng")
		client.SetPageSegMode(gosseract.PSM_SINGLE_WORD) // 单词模式更适合罗马数字
		client.SetWhitelist("IVX ")

		// 保存到临时文件
		tmpfile, err := ioutil.TempFile("", "ocr-*.png")
		if err != nil {
			client.Close()
			continue
		}

		png.Encode(tmpfile, testImg.img)
		tmpfile.Close()

		client.SetImage(tmpfile.Name())
		text, err := client.Text()
		os.Remove(tmpfile.Name())
		client.Close()

		if err != nil {
			continue
		}

		text = strings.TrimSpace(text)
		text = strings.ReplaceAll(text, " ", "")

		fmt.Printf("[OCR Debug] %s mode: raw='%s'\n", testImg.name, text)

		if text == "" {
			continue
		}

		// 匹配罗马数字
		switch text {
		case "I":
			fmt.Printf("[OCR Success] Recognized: I (Day 1)\n")
			return 1, nil
		case "II":
			fmt.Printf("[OCR Success] Recognized: II (Day 2)\n")
			return 2, nil
		case "III":
			fmt.Printf("[OCR Success] Recognized: III (Day 3)\n")
			return 3, nil
		default:
			// Fallback: 数 I 的个数
			iCount := strings.Count(text, "I")
			if iCount >= 1 && iCount <= 3 {
				fmt.Printf("[OCR Success] Counted %d I's (Day %d)\n", iCount, iCount)
				return iCount, nil
			}
		}
	}

	return -1, fmt.Errorf("no text found (OCR returned empty), check /tmp/ocr_debug/ for debug images")
}

// Threshold 将灰度图转换为二值图
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

// AdaptiveThreshold 执行 Otsu 方法阈值化
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

// InvertImage 反转灰度图像
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
