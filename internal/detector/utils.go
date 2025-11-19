package detector

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

// Point 表示一个二维点
type Point struct {
	X, Y int
}

// Rect 表示一个矩形
type Rect struct {
	X, Y, Width, Height int
}

// NewRect 创建一个新的矩形
func NewRect(x, y, width, height int) Rect {
	return Rect{X: x, Y: y, Width: width, Height: height}
}

// Contains 检查点是否在矩形内
func (r Rect) Contains(p Point) bool {
	return p.X >= r.X && p.X < r.X+r.Width &&
		p.Y >= r.Y && p.Y < r.Y+r.Height
}

// ToImageRect 转换为 image.Rectangle
func (r Rect) ToImageRect() image.Rectangle {
	return image.Rect(r.X, r.Y, r.X+r.Width, r.Y+r.Height)
}

// CropImage 将图像裁剪到指定的矩形
func CropImage(img image.Image, rect Rect) image.Image {
	bounds := rect.ToImageRect()

	// 创建指定裁剪大小的新图像
	cropped := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))

	// 复制像素
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

// ResizeImage 将图像调整到指定的宽度和高度
// 为简单起见使用最近邻插值
func ResizeImage(img image.Image, width, height int) image.Image {
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	resized := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// 计算源坐标
			srcX := x * srcWidth / width
			srcY := y * srcHeight / height

			// 从源图像获取颜色
			c := img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY)
			resized.Set(x, y, c)
		}
	}

	return resized
}

// RGB2Gray 将 RGB 图像转换为灰度图
func RGB2Gray(img image.Image) *image.Gray {
	bounds := img.Bounds()
	gray := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// 转换为 8 位值
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)
			// 使用标准公式计算灰度值
			grayValue := uint8(0.299*float64(r8) + 0.587*float64(g8) + 0.114*float64(b8))
			gray.SetGray(x, y, color.Gray{Y: grayValue})
		}
	}

	return gray
}

// RGB2HSV 将 RGB 颜色转换为 HSV
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

// RGB2HLS 将 RGB 颜色转换为 HLS（色调、亮度、饱和度）
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

// CountNonZero 计数灰度图像中的非零像素
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

// InRange 检查颜色是否在指定范围内
func InRange(c color.Color, lower, upper [3]uint8) bool {
	r, g, b, _ := c.RGBA()
	r8 := uint8(r >> 8)
	g8 := uint8(g >> 8)
	b8 := uint8(b >> 8)

	return r8 >= lower[0] && r8 <= upper[0] &&
		g8 >= lower[1] && g8 <= upper[1] &&
		b8 >= lower[2] && b8 <= upper[2]
}

// CreateMask 基于颜色范围创建二值掩码
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

// CalculateSimilarity 计算两个灰度图像之间的相似度
// 返回 0 到 1 之间的值，其中 1 表示完全相同
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

	// 计算 MSE（均方误差）
	mse := sumSquaredDiff / float64(totalPixels)

	// 将 MSE 转换为相似度分数（0-1）
	// 最大 MSE 为 255^2 = 65025
	maxMSE := 255.0 * 255.0
	similarity := 1.0 - (mse / maxMSE)

	return similarity, nil
}

// MatchResult 表示模板匹配的结果
type MatchResult struct {
	Location   Point   // 匹配的左上角
	Similarity float64 // 相似度分数（0-1）
	Found      bool    // 是否找到匹配
}

// TemplateMatch 对源图像执行模板匹配
// 返回最佳匹配的位置和相似度
func TemplateMatch(source, template image.Image, threshold float64) (*MatchResult, error) {
	return TemplateMatchWithStride(source, template, threshold, 1)
}

// TemplateMatchWithStride 执行具有可配置步长的模板匹配以提高速度
// stride > 1 先执行粗略搜索，然后在最佳匹配周围细化
func TemplateMatchWithStride(source, template image.Image, threshold float64, stride int) (*MatchResult, error) {
	// 将图像转换为灰度图
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

	// 优化：使用步长进行粗略搜索
	if stride < 1 {
		stride = 1
	}

	// 使用步长进行粗略搜索
	for y := 0; y <= srcHeight-tmplHeight; y += stride {
		for x := 0; x <= srcWidth-tmplWidth; x += stride {
			// 从源图像中提取感兴趣区域
			roi := extractROI(srcGray, x, y, tmplWidth, tmplHeight)

			// 计算相似度
			similarity, err := CalculateSimilarity(roi, tmplGray)
			if err != nil {
				continue
			}

			// 更新最佳匹配
			if similarity > bestMatch.Similarity {
				bestMatch.Similarity = similarity
				bestMatch.Location = Point{X: x, Y: y}
			}
		}
	}

	// 如果 stride > 1 且找到了一个好的候选，在其周围细化搜索
	if stride > 1 && bestMatch.Similarity > threshold*0.9 {
		refineX := bestMatch.Location.X
		refineY := bestMatch.Location.Y

		// 在最佳匹配周围的小区域内使用 stride=1 搜索
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

	// 检查是否找到高于阈值的匹配
	if bestMatch.Similarity >= threshold {
		bestMatch.Found = true
	}

	return bestMatch, nil
}

// extractROI 从灰度图像中提取感兴趣区域
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

// TemplateMatchMultiple 将模板与多个源区域进行匹配
// 返回所有区域中的最佳匹配
func TemplateMatchMultiple(source image.Image, template image.Image, regions []Rect, threshold float64) (*MatchResult, error) {
	bestMatch := &MatchResult{
		Similarity: 0.0,
		Found:      false,
	}

	for _, region := range regions {
		// 将源裁剪到区域
		cropped := CropImage(source, region)

		// 执行模板匹配
		result, err := TemplateMatch(cropped, template, threshold)
		if err != nil {
			continue
		}

		// 调整位置以考虑区域偏移
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

// ColorRange 表示用于过滤的颜色范围
type ColorRange struct {
	Lower [3]uint8 // RGB 下限
	Upper [3]uint8 // RGB 上限
}

// HasBrightPixels 检查区域是否有足够的亮/白色像素（用于文本检测）
// 使用采样来提高性能
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
			// 转换为 8 位
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			// 检查像素是否明亮（白色或浅色）
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

// FindCandidateRegions 使用基于颜色的过滤查找潜在的文本区域
// 这比模板匹配快得多，可以缩小搜索区域
func FindCandidateRegions(img image.Image, windowWidth, windowHeight, stepSize int, brightThreshold float64) []Rect {
	candidates := []Rect{}
	bounds := img.Bounds()

	// 使用滑动窗口扫描图像
	for y := bounds.Min.Y; y < bounds.Max.Y-windowHeight; y += stepSize {
		for x := bounds.Min.X; x < bounds.Max.X-windowWidth; x += stepSize {
			region := NewRect(x, y, windowWidth, windowHeight)

			// 使用采样进行快速颜色检查
			if HasBrightPixels(img, region, brightThreshold, 5) {
				candidates = append(candidates, region)
			}
		}
	}

	return candidates
}

// TemplateMatchPyramid 使用图像金字塔执行多尺度模板匹配
// 对于大图像，这比全分辨率匹配更快
func TemplateMatchPyramid(source, template image.Image, threshold float64, scales []float64) (*MatchResult, error) {
	if len(scales) == 0 {
		scales = []float64{0.25, 0.5, 1.0} // 默认尺度
	}

	bestMatch := &MatchResult{
		Similarity: 0.0,
		Found:      false,
	}

	srcBounds := source.Bounds()
	tmplBounds := template.Bounds()

	for i, scale := range scales {
		// 根据尺度调整图像大小
		scaledSrcW := int(float64(srcBounds.Dx()) * scale)
		scaledSrcH := int(float64(srcBounds.Dy()) * scale)
		scaledTmplW := int(float64(tmplBounds.Dx()) * scale)
		scaledTmplH := int(float64(tmplBounds.Dy()) * scale)

		if scaledSrcW < scaledTmplW || scaledSrcH < scaledTmplH {
			continue
		}

		scaledSrc := ResizeImage(source, scaledSrcW, scaledSrcH)
		scaledTmpl := ResizeImage(template, scaledTmplW, scaledTmplH)

		// 对于粗略尺度，使用较低的阈值
		adjustedThreshold := threshold
		if i < len(scales)-1 {
			adjustedThreshold = threshold * 0.85 // 粗略搜索的较低阈值
		}

		// 在此尺度执行模板匹配
		result, err := TemplateMatch(scaledSrc, scaledTmpl, adjustedThreshold)
		if err != nil {
			continue
		}

		if result.Found {
			// 将位置缩放回原始坐标
			result.Location.X = int(float64(result.Location.X) / scale)
			result.Location.Y = int(float64(result.Location.Y) / scale)

			if i == len(scales)-1 {
				// 最后一个尺度（全分辨率），直接返回
				return result, nil
			}

			// 对于粗略尺度，通过聚焦找到的区域在下一次迭代中细化搜索
			if result.Similarity > bestMatch.Similarity {
				bestMatch = result
			}
		}
	}

	return bestMatch, nil
}

// min 返回两个整数中的最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max 返回两个整数中的最大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
// EdgeDetect 对灰度图像应用 Sobel 边缘检测
// 这有助于消除背景噪声并专注于文本结构
func EdgeDetect(img *image.Gray) *image.Gray {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	edges := image.NewGray(bounds)

	// Sobel 核
	gx := [3][3]int{
		{-1, 0, 1},
		{-2, 0, 2},
		{-1, 0, 1},
	}

	gy := [3][3]int{
		{-1, -2, -1},
		{0, 0, 0},
		{1, 2, 1},
	}

	// 应用 Sobel 算子
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			var sumX, sumY int

			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					pixel := int(img.GrayAt(bounds.Min.X+x+kx, bounds.Min.Y+y+ky).Y)
					sumX += pixel * gx[ky+1][kx+1]
					sumY += pixel * gy[ky+1][kx+1]
				}
			}

			// 计算梯度幅度
			magnitude := math.Sqrt(float64(sumX*sumX + sumY*sumY))
			if magnitude > 255 {
				magnitude = 255
			}

			edges.SetGray(bounds.Min.X+x, bounds.Min.Y+y, color.Gray{Y: uint8(magnitude)})
		}
	}

	return edges
}

// ThresholdImage 应用二值阈值化以突出显示强边缘
func ThresholdImage(img *image.Gray, threshold uint8) *image.Gray {
	bounds := img.Bounds()
	result := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			v := img.GrayAt(x, y).Y
			if v >= threshold {
				result.SetGray(x, y, color.Gray{Y: 255})
			} else {
				result.SetGray(x, y, color.Gray{Y: 0})
			}
		}
	}

	return result
}

// CountVerticalSegments 通过计数垂直段来分析罗马数字
// 这有效是因为：I=1段，II=2段，III=3段
// 返回找到的垂直段数，如果检测失败则返回 -1
func CountVerticalSegments(img image.Image) int {
	// 转换为灰度图
	gray := RGB2Gray(img)

	// 应用阈值化得到二值图像（黑色背景上的白色文本）
	// 使用适度的阈值来捕获白色文本
	binary := ThresholdImage(gray, 160)

	bounds := binary.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width == 0 || height == 0 {
		return -1
	}

	// 计算垂直投影（每列中白色像素的总和）
	projection := make([]int, width)
	maxProjection := 0
	for x := 0; x < width; x++ {
		sum := 0
		for y := 0; y < height; y++ {
			if binary.GrayAt(bounds.Min.X+x, bounds.Min.Y+y).Y > 128 {
				sum++
			}
		}
		projection[x] = sum
		if sum > maxProjection {
			maxProjection = sum
		}
	}

	// 在投影中查找峰值（每个峰值 = 一条垂直线/段）
	// 使用基于最大投影的自适应阈值
	// 这可以处理不同的图像大小和对比度水平
	threshold := maxProjection / 3 // 至少是最大投影的 1/3

	if threshold < height/6 {
		threshold = height / 6 // 备用：至少是高度的 1/6
	}

	// 峰值检测：在投影中查找局部最大值
	// 每个罗马数字垂直条在投影中创建一个峰值
	peaks := 0
	// 平衡灵敏度和抗噪声：45% 是经过测试的折衷值
	// 35% 会产生假峰值（Day2_test2 检测到 5 个），50% 会漏检
	peakThreshold := int(float64(maxProjection) * 0.45)

	for x := 1; x < width-1; x++ {
		// 检查这是否是局部最大值
		if projection[x] > projection[x-1] && projection[x] > projection[x+1] && projection[x] >= peakThreshold {
			peaks++
			// 跳过附近的点以避免多次计数同一个峰值
			x += 2
		}
	}

	// 如果没有找到峰值，回退到简单的段计数
	if peaks == 0 {
		inSegment := false
		for x := 0; x < width; x++ {
			if projection[x] >= threshold {
				if !inSegment {
					peaks++
					inSegment = true
				}
			} else if projection[x] < threshold/2 {
				// 需要显著下降才能结束段
				inSegment = false
			}
		}
	}

	return peaks
}

// ExtractRomanNumeralRegion 提取可能包含罗马数字的区域
// 它查找"DAY X"文本的最右边部分，其中 X 是罗马数字
func ExtractRomanNumeralRegion(img image.Image, fullWidth int) image.Image {
	bounds := img.Bounds()

	// 罗马数字通常位于"DAY X"模板的右侧 40%
	// 并垂直居中
	numeralX := int(float64(bounds.Dx()) * 0.6) // 从图像的 60% 开始
	numeralWidth := int(float64(bounds.Dx()) * 0.35)  // 宽度约为 35%
	numeralY := int(float64(bounds.Dy()) * 0.2) // 从顶部 20% 开始
	numeralHeight := int(float64(bounds.Dy()) * 0.6) // 高度约为 60%

	// 确保在边界内
	if numeralX < 0 {
		numeralX = 0
	}
	if numeralX+numeralWidth > bounds.Dx() {
		numeralWidth = bounds.Dx() - numeralX
	}
	if numeralY < 0 {
		numeralY = 0
	}
	if numeralY+numeralHeight > bounds.Dy() {
		numeralHeight = bounds.Dy() - numeralY
	}

	// 提取区域
	region := NewRect(numeralX, numeralY, numeralWidth, numeralHeight)
	return CropImage(img, region)
}
