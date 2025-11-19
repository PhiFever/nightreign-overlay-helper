package detector

import (
	"fmt"
	"image"
	_ "image/png" // 导入 PNG 解码器
	"os"
	"path/filepath"
	"time"

	"github.com/PhiFever/nightreign-overlay-helper/internal/config"
	"github.com/PhiFever/nightreign-overlay-helper/internal/logger"
)

// DayResult 表示天数检测的结果
type DayResult struct {
	Day          int           // 当前天数（从0开始）
	Phase        int           // 当前天内的阶段
	ElapsedTime  time.Duration // 从当天开始经过的时间
	ShrinkTime   time.Duration // 到下一次缩圈的时间
	NextPhaseIn  time.Duration // 到下一阶段的时间
	IsDetected   bool          // 是否成功检测到天数
	LastUpdateAt time.Time     // 上次更新此结果的时间
}

// String 返回结果的字符串表示
func (r *DayResult) String() string {
	if !r.IsDetected {
		return "Day: Not Detected"
	}
	return fmt.Sprintf("Day %d Phase %d | Elapsed: %v | Shrink in: %v | Next phase in: %v",
		r.Day, r.Phase, r.ElapsedTime, r.ShrinkTime, r.NextPhaseIn)
}

// DayTemplate 表示特定语言的模板
type DayTemplate struct {
	Language string
	Day1     image.Image
	Day2     image.Image
	Day3     image.Image
}

// DetectionStrategy 表示要使用的检测策略
type DetectionStrategy int

const (
	// StrategyAuto 自动选择最佳策略
	StrategyAuto DetectionStrategy = iota
	// StrategyHotspotCache 使用之前检测的缓存热点
	StrategyHotspotCache
	// StrategyColorFilter 使用基于颜色的过滤来查找候选区域
	StrategyColorFilter
	// StrategyPyramid 使用图像金字塔进行多尺度搜索
	StrategyPyramid
	// StrategyPredefined 在预定义的常见位置搜索
	StrategyPredefined
	// StrategyFullScan 执行全屏扫描（最慢，最彻底）
	StrategyFullScan
)

// DetectionStats 跟踪检测性能指标
type DetectionStats struct {
	LastStrategy      DetectionStrategy
	LastDetectionTime time.Duration
	CacheHitCount     int
	ColorFilterCount  int
	PyramidCount      int
	PredefinedCount   int
	FullScanCount     int
	TotalDetections   int
}

// DayDetector 检测游戏中的当前天数和阶段
type DayDetector struct {
	*BaseDetector
	config     *config.Config
	lastResult *DayResult

	// 检测区域（旧版，用于向后兼容）
	dayRegion Rect

	// 模板缓存
	templates map[string]*DayTemplate

	// 当前语言
	currentLang string

	// 配置
	updateInterval      time.Duration
	lastUpdateTime      time.Time
	matchThreshold      float64
	enableTemplateMatch bool

	// 智能检测
	lastMatchLocation *Point            // 上次成功匹配的缓存位置
	searchRadius      int               // 围绕缓存位置进行本地搜索的半径
	strategy          DetectionStrategy // 当前检测策略
	stats             DetectionStats    // 性能统计

	// 性能调优
	colorFilterThreshold float64 // 亮像素比率的阈值（0.0-1.0）
	pyramidScales        []float64
	candidateStepSize    int // 候选区域扫描的步长
}

// NewDayDetector 创建一个新的天数检测器
func NewDayDetector(cfg *config.Config) *DayDetector {
	return &DayDetector{
		BaseDetector:        NewBaseDetector("DayDetector"),
		config:              cfg,
		updateInterval:      time.Duration(cfg.UpdateInterval * float64(time.Second)),
		templates:           make(map[string]*DayTemplate),
		currentLang:         "chs", // 默认为简体中文
		matchThreshold:      0.8,   // 默认阈值
		enableTemplateMatch: false, // 默认禁用（使用模拟模式）
		lastResult: &DayResult{
			IsDetected: false,
		},
		// 智能检测设置
		searchRadius:         100,              // 在上次匹配的100px半径内搜索
		strategy:             StrategyAuto,     // 自动选择策略
		colorFilterThreshold: 0.1,              // 10%的亮像素表示潜在的文本
		pyramidScales:        []float64{0.125}, // 优化：激进的降采样以提高速度（8倍缩小）
		candidateStepSize:    80,               // 优化：更大的步长以加快扫描速度
		stats:                DetectionStats{},
	}
}

// SetLanguage 设置模板匹配的当前语言
func (d *DayDetector) SetLanguage(lang string) {
	d.currentLang = lang
}

// EnableTemplateMatching 启用或禁用模板匹配
func (d *DayDetector) EnableTemplateMatching(enable bool) {
	d.enableTemplateMatch = enable
}

// SetMatchThreshold 设置模板匹配的相似度阈值
func (d *DayDetector) SetMatchThreshold(threshold float64) {
	d.matchThreshold = threshold
}

// SetDetectionStrategy 设置检测策略
func (d *DayDetector) SetDetectionStrategy(strategy DetectionStrategy) {
	d.strategy = strategy
}

// GetDetectionStats 返回当前的检测统计信息
func (d *DayDetector) GetDetectionStats() DetectionStats {
	return d.stats
}

// SetSearchRadius 设置热点缓存的搜索半径
func (d *DayDetector) SetSearchRadius(radius int) {
	d.searchRadius = radius
}

// ResetCache 清除缓存的热点位置
func (d *DayDetector) ResetCache() {
	d.lastMatchLocation = nil
	logger.Debugf("[%s] Hotspot cache reset", d.Name())
}

// Initialize 初始化天数检测器
func (d *DayDetector) Initialize() error {
	logger.Infof("[%s] Initializing...", d.Name())

	// 从数据目录加载模板
	if err := d.loadTemplates(); err != nil {
		logger.Warningf("[%s] Failed to load templates: %v (using mock mode)", d.Name(), err)
		// 不返回错误 - 我们仍然可以在模拟模式下运行
	} else {
		logger.Infof("[%s] Templates loaded successfully", d.Name())
	}

	// 设置默认检测区域（应针对实际游戏进行校准）
	d.dayRegion = NewRect(100, 50, 200, 50)

	logger.Infof("[%s] Initialized successfully", d.Name())
	return nil
}

// loadTemplates 从数据目录加载天数数字模板
func (d *DayDetector) loadTemplates() error {
	// 获取数据目录路径，尝试多个可能的位置
	possiblePaths := []string{
		"data/day_template",       // 从项目根目录运行时
		"../../data/day_template", // 运行测试时
		"../data/day_template",    // 备用位置
	}

	var dataDir string
	var found bool
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			dataDir = path
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("template directory not found in any of: %v", possiblePaths)
	}

	// 要加载的语言
	languages := []string{"chs", "cht", "eng", "jp"}

	for _, lang := range languages {
		template := &DayTemplate{
			Language: lang,
		}

		// 加载第1、2、3天的模板
		for day := 1; day <= 3; day++ {
			filename := filepath.Join(dataDir, fmt.Sprintf("%s_%d.png", lang, day))

			img, err := loadImageFromFile(filename)
			if err != nil {
				return fmt.Errorf("failed to load template %s: %w", filename, err)
			}

			// 存储模板
			switch day {
			case 1:
				template.Day1 = img
			case 2:
				template.Day2 = img
			case 3:
				template.Day3 = img
			}
		}

		d.templates[lang] = template
		logger.Debugf("[%s] Loaded templates for language: %s", d.Name(), lang)
	}

	return nil
}

// loadImageFromFile 从文件加载图像
func loadImageFromFile(filename string) (image.Image, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return img, nil
}

// Detect 对给定的图像执行天数检测
func (d *DayDetector) Detect(img image.Image) (interface{}, error) {
	if !d.IsEnabled() {
		return d.lastResult, nil
	}

	// 检查是否应该更新（速率限制）
	now := time.Now()
	if now.Sub(d.lastUpdateTime) < d.updateInterval {
		return d.lastResult, nil
	}
	d.lastUpdateTime = now

	// 从图像中检测天数和阶段
	day := d.detectDay(img)
	phase := d.detectPhase(img)

	// 如果检测失败，返回上次的结果
	if day < 0 || phase < 0 {
		result := &DayResult{
			IsDetected:   false,
			LastUpdateAt: now,
		}
		d.lastResult = result
		return result, nil
	}

	// 计算时间信息
	elapsed, shrink, nextPhase := d.calculateTimes(day, phase)

	result := &DayResult{
		Day:          day,
		Phase:        phase,
		ElapsedTime:  elapsed,
		ShrinkTime:   shrink,
		NextPhaseIn:  nextPhase,
		IsDetected:   true,
		LastUpdateAt: now,
	}

	d.lastResult = result
	return result, nil
}

// Cleanup 释放检测器使用的资源
func (d *DayDetector) Cleanup() error {
	logger.Infof("[%s] Cleaning up...", d.Name())

	// 清除模板
	d.templates = nil
	d.lastResult = nil

	logger.Infof("[%s] Cleaned up successfully", d.Name())
	return nil
}

// GetLastResult 返回上次的检测结果
func (d *DayDetector) GetLastResult() *DayResult {
	return d.lastResult
}

// detectDay 使用智能多层搜索从图像中检测当前天数
// 如果未检测到则返回 -1
func (d *DayDetector) detectDay(img image.Image) int {
	// 如果禁用了模板匹配，则使用模拟模式
	if !d.enableTemplateMatch {
		return d.detectDayMock()
	}

	// 获取当前语言的模板
	template, ok := d.templates[d.currentLang]
	if !ok {
		logger.Warningf("[%s] No template found for language: %s", d.Name(), d.currentLang)
		return d.detectDayMock()
	}

	startTime := time.Now()

	// 根据策略使用智能检测
	var day int
	var location *Point

	switch d.strategy {
	case StrategyHotspotCache:
		day, location = d.detectWithHotspotCache(img, template)
	case StrategyColorFilter:
		day, location = d.detectWithColorFilter(img, template)
	case StrategyPyramid:
		day, location = d.detectWithPyramid(img, template)
	case StrategyPredefined:
		day, location = d.detectWithPredefined(img, template)
	case StrategyFullScan:
		day, location = d.detectWithFullScan(img, template)
	default: // StrategyAuto
		day, location = d.detectDayIntelligent(img, template)
	}

	// 更新统计信息
	d.stats.LastDetectionTime = time.Since(startTime)
	d.stats.TotalDetections++

	// 如果找到，则更新缓存位置
	if day > 0 && location != nil {
		d.lastMatchLocation = location
	}

	return day
}

// detectDayIntelligent 使用多层智能检测（自动策略）
func (d *DayDetector) detectDayIntelligent(img image.Image, template *DayTemplate) (int, *Point) {
	// 第1层：热点缓存（最快，通常命中）
	if d.lastMatchLocation != nil {
		day, loc := d.detectWithHotspotCache(img, template)
		if day > 0 {
			d.stats.LastStrategy = StrategyHotspotCache
			d.stats.CacheHitCount++
			return day, loc
		}
	}

	// 第2层：预定义热点（优化：屏幕中心优先 - 对典型的DAY显示来说很快）
	day, loc := d.detectWithPredefined(img, template)
	if day > 0 {
		d.stats.LastStrategy = StrategyPredefined
		d.stats.PredefinedCount++
		return day, loc
	}

	// 第3层：基于颜色的过滤（快速，缩小搜索范围）
	day, loc = d.detectWithColorFilter(img, template)
	if day > 0 {
		d.stats.LastStrategy = StrategyColorFilter
		d.stats.ColorFilterCount++
		return day, loc
	}

	// 第4层：图像金字塔（中等速度，覆盖良好）
	day, loc = d.detectWithPyramid(img, template)
	if day > 0 {
		d.stats.LastStrategy = StrategyPyramid
		d.stats.PyramidCount++
		return day, loc
	}

	// 第5层：全屏扫描（最慢，最彻底 - 最后手段）
	logger.Debugf("[%s] Falling back to full scan", d.Name())
	day, loc = d.detectWithFullScan(img, template)
	if day > 0 {
		d.stats.LastStrategy = StrategyFullScan
		d.stats.FullScanCount++
	}

	return day, loc
}

// detectWithHotspotCache 在最后已知位置附近搜索
func (d *DayDetector) detectWithHotspotCache(img image.Image, template *DayTemplate) (int, *Point) {
	if d.lastMatchLocation == nil {
		return -1, nil
	}

	bounds := img.Bounds()
	x := max(0, d.lastMatchLocation.X-d.searchRadius)
	y := max(0, d.lastMatchLocation.Y-d.searchRadius)
	w := min(d.searchRadius*2, bounds.Dx()-x)
	h := min(d.searchRadius*2, bounds.Dy()-y)

	region := NewRect(x, y, w, h)
	day, loc := d.matchDayInRegion(img, template, region)

	if day > 0 && loc != nil {
		logger.Debugf("[%s] Cache hit! Found Day %d near cached location", d.Name(), day)
		return day, loc
	}

	return -1, nil
}

// detectWithColorFilter 使用基于颜色的过滤来查找候选区域
func (d *DayDetector) detectWithColorFilter(img image.Image, template *DayTemplate) (int, *Point) {
	// 根据模板估计搜索窗口大小
	templateBounds := template.Day1.Bounds()
	windowW := templateBounds.Dx() * 3
	windowH := templateBounds.Dy() * 3

	// 查找具有亮像素的候选区域
	candidates := FindCandidateRegions(img, windowW, windowH, d.candidateStepSize, d.colorFilterThreshold)

	logger.Debugf("[%s] Color filter found %d candidate regions", d.Name(), len(candidates))

	// 在候选区域中搜索
	for _, region := range candidates {
		day, loc := d.matchDayInRegion(img, template, region)
		if day > 0 {
			return day, loc
		}
	}

	return -1, nil
}

// detectWithPyramid 使用图像金字塔进行多尺度搜索
func (d *DayDetector) detectWithPyramid(img image.Image, template *DayTemplate) (int, *Point) {
	// 使用金字塔搜索尝试每个天数模板
	templates := []image.Image{template.Day1, template.Day2, template.Day3}

	for dayNum, tmpl := range templates {
		result, err := TemplateMatchPyramid(img, tmpl, d.matchThreshold, d.pyramidScales)
		if err == nil && result.Found {
			day := dayNum + 1
			logger.Debugf("[%s] Pyramid found Day %d at (%d, %d) with similarity %.2f",
				d.Name(), day, result.Location.X, result.Location.Y, result.Similarity)
			return day, &result.Location
		}
	}

	return -1, nil
}

// detectWithPredefined 在预定义的常见UI位置搜索
func (d *DayDetector) detectWithPredefined(img image.Image, template *DayTemplate) (int, *Point) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// 基于典型游戏布局的常见UI位置
	// 优化：屏幕中心优先（DAY文本通常出现的位置）
	predefinedRegions := []Rect{
		// 中心区域（DAY显示的最高优先级）
		NewRect(int(float64(w)*0.35), int(float64(h)*0.35), int(float64(w)*0.30), int(float64(h)*0.30)),
		// 更宽的中心区域（如果文本稍微偏离中心的备选方案）
		NewRect(int(float64(w)*0.25), int(float64(h)*0.25), int(float64(w)*0.50), int(float64(h)*0.50)),
		// 顶部中心
		NewRect(int(float64(w)*0.40), int(float64(h)*0.02), int(float64(w)*0.20), int(float64(h)*0.15)),
		// 左上角
		NewRect(int(float64(w)*0.02), int(float64(h)*0.02), int(float64(w)*0.20), int(float64(h)*0.15)),
	}

	for _, region := range predefinedRegions {
		day, loc := d.matchDayInRegion(img, template, region)
		if day > 0 {
			logger.Debugf("[%s] Found in predefined region at (%d, %d)", d.Name(), loc.X, loc.Y)
			return day, loc
		}
	}

	return -1, nil
}

// detectWithFullScan 执行全屏模板匹配（最慢）
func (d *DayDetector) detectWithFullScan(img image.Image, template *DayTemplate) (int, *Point) {
	bounds := img.Bounds()
	fullRegion := NewRect(bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy())

	return d.matchDayInRegion(img, template, fullRegion)
}

// matchDayInRegion 尝试在特定区域匹配天数模板
func (d *DayDetector) matchDayInRegion(img image.Image, template *DayTemplate, region Rect) (int, *Point) {
	// 新方法：使用垂直段计数来识别罗马数字
	// 这对背景噪声和模板匹配问题免疫

	// 裁剪到区域
	regionImg := CropImage(img, region)

	// 首先，使用第1天模板定位"DAY"文本（所有模板都有相同的"DAY"部分）
	// 我们使用较低的阈值进行初始检测
	regionBounds := regionImg.Bounds()
	scale := 0.5 // 更好的细节保留
	scaledWidth := int(float64(regionBounds.Dx()) * scale)
	scaledHeight := int(float64(regionBounds.Dy()) * scale)

	if scaledWidth < 100 || scaledHeight < 100 {
		scale = 1.0
		scaledWidth = regionBounds.Dx()
		scaledHeight = regionBounds.Dy()
	}

	scaledRegion := ResizeImage(regionImg, scaledWidth, scaledHeight)

	// 使用第1天模板定位"DAY"文本
	day1Template := template.Day1
	tmplBounds := day1Template.Bounds()
	scaledTmplWidth := int(float64(tmplBounds.Dx()) * scale)
	scaledTmplHeight := int(float64(tmplBounds.Dy()) * scale)
	scaledTmpl := ResizeImage(day1Template, scaledTmplWidth, scaledTmplHeight)

	// 较低的阈值用于初始"DAY"检测
	result, err := TemplateMatch(scaledRegion, scaledTmpl, 0.7)
	if err != nil || !result.Found {
		logger.Debugf("[%s] No DAY text found in region", d.Name())
		return -1, nil
	}

	logger.Infof("[%s] Found DAY text at (%d, %d) with similarity=%.4f",
		d.Name(), result.Location.X, result.Location.Y, result.Similarity)

	// 提取罗马数字区域（"DAY"之后的最右边部分）
	// 将坐标缩放回原始区域大小
	dayX := int(float64(result.Location.X) / scale)
	dayY := int(float64(result.Location.Y) / scale)

	// Phase 2 优化：动态提取罗马数字区域
	// 使用二值化 + 垂直投影分析，而非固定比例
	// 确保裁剪区域不越界
	cropWidth := tmplBounds.Dx()
	cropHeight := tmplBounds.Dy()
	if dayX+cropWidth > regionBounds.Dx() {
		cropWidth = regionBounds.Dx() - dayX
	}
	if dayY+cropHeight > regionBounds.Dy() {
		cropHeight = regionBounds.Dy() - dayY
	}

	if cropWidth <= 0 || cropHeight <= 0 {
		logger.Warningf("[%s] Invalid crop region, skipping detection", d.Name())
		return -1, nil
	}

	matchedTemplateRegion := CropImage(regionImg, NewRect(dayX, dayY, cropWidth, cropHeight))

	relativeNumeralRegion := ExtractRomanNumeralRegionDynamic(matchedTemplateRegion, tmplBounds.Dx(), tmplBounds.Dy())

	// 检查动态提取是否成功
	if relativeNumeralRegion.Width == 0 {
		logger.Warningf("[%s] Failed to extract numeral region dynamically, skipping detection", d.Name())
		return -1, nil
	}

	// 转换为绝对坐标（相对于 regionImg）
	numeralRegion := NewRect(
		dayX+relativeNumeralRegion.X,
		dayY+relativeNumeralRegion.Y,
		relativeNumeralRegion.Width,
		relativeNumeralRegion.Height,
	)

	logger.Debugf("[%s] Dynamic numeral region: x=%d, y=%d, w=%d, h=%d (template w=%d, h=%d)",
		d.Name(), numeralRegion.X, numeralRegion.Y, numeralRegion.Width, numeralRegion.Height,
		tmplBounds.Dx(), tmplBounds.Dy())

	// 自动修正区域边界以确保在图像范围内
	clippedRegion := numeralRegion

	// 修正左边界
	if clippedRegion.X < 0 {
		clippedRegion.Width += clippedRegion.X
		clippedRegion.X = 0
	}

	// 修正上边界
	if clippedRegion.Y < 0 {
		clippedRegion.Height += clippedRegion.Y
		clippedRegion.Y = 0
	}

	// 修正右边界
	if clippedRegion.X+clippedRegion.Width > regionBounds.Dx() {
		clippedRegion.Width = regionBounds.Dx() - clippedRegion.X
	}

	// 修正下边界
	if clippedRegion.Y+clippedRegion.Height > regionBounds.Dy() {
		clippedRegion.Height = regionBounds.Dy() - clippedRegion.Y
	}

	// 检查修正后的区域是否仍然有效
	if clippedRegion.Width <= 10 || clippedRegion.Height <= 10 {
		logger.Warningf("[%s] Numeral region too small after clipping (w=%d, h=%d), skipping detection",
			d.Name(), clippedRegion.Width, clippedRegion.Height)
		return -1, nil
	}

	// 使用修正后的区域
	numeralRegion = clippedRegion

	numeralImg := CropImage(regionImg, numeralRegion)

	// 优先尝试 OCR（最快最准确）
	logger.Infof("[%s] 🔍 OCR support compiled: %v", d.Name(), OCRAvailable)
	if OCRAvailable {
		logger.Infof("[%s] 🚀 Trying OCR detection on numeral region (%dx%d)...",
			d.Name(), numeralRegion.Width, numeralRegion.Height)
		dayNum, err := OCRExtractDayNumber(numeralImg)
		if err == nil && dayNum >= 1 && dayNum <= 3 {
			logger.Infof("[%s] ✅ OCR detection succeeded: Day %d", d.Name(), dayNum)
			location := &Point{
				X: region.X + dayX,
				Y: region.Y + dayY,
			}
			return dayNum, location
		}
		logger.Warningf("[%s] ❌ OCR failed (%v), falling back to segment counting", d.Name(), err)
	} else {
		logger.Warningf("[%s] ⚠️  OCR not available, using segment counting", d.Name())
	}

	// OCR 失败或不可用，使用垂直段计数
	segments := CountVerticalSegments(numeralImg)
	logger.Infof("[%s] Detected %d vertical segments (Roman numeral)", d.Name(), segments)

	// 将段数映射到天数
	var day int
	switch segments {
	case 1:
		day = 1 // I
	case 2:
		day = 2 // II
	case 3:
		day = 3 // III
	default:
		logger.Warningf("[%s] Invalid segment count: %d, skipping detection", d.Name(), segments)
		return -1, nil
	}

	// 计算绝对位置
	location := &Point{
		X: region.X + dayX,
		Y: region.Y + dayY,
	}

	logger.Infof("[%s] Segment-based detection: Day %d", d.Name(), day)
	return day, location
}

// matchDayInRegionOld 是旧的基于模板匹配的方法（备用）
// 简化版本：只选择相似度最高的模板
func (d *DayDetector) matchDayInRegionOld(img image.Image, template *DayTemplate, region Rect) (int, *Point) {
	// 禁用模板匹配 fallback：由于子串匹配问题，总是倾向选择 Day 1 或 Day 3
	// 更诚实的做法是承认检测失败，而不是给出误导性的结果
	logger.Warningf("[%s] Segment-based detection failed, no reliable fallback available", d.Name())
	return -1, nil
}

// detectDayMock 提供用于测试的模拟天数检测
func (d *DayDetector) detectDayMock() int {
	// 基于时间模拟检测第1-3天
	seconds := time.Now().Unix() % 30
	if seconds < 10 {
		return 1
	} else if seconds < 20 {
		return 2
	} else {
		return 3
	}
}

// detectPhase 从图像中检测当前阶段
// 如果未检测到则返回 -1
func (d *DayDetector) detectPhase(img image.Image) int {
	// TODO: 实现阶段标记的模板匹配
	// 目前，模拟循环0-3的阶段检测

	// 基于时间模拟检测0-3阶段
	seconds := time.Now().Unix() % 20
	return int(seconds / 5) // 返回 0, 1, 2, 或 3
}

// calculateTimes 计算经过时间、缩圈时间和下一阶段时间
func (d *DayDetector) calculateTimes(day, phase int) (elapsed, shrink, nextPhase time.Duration) {
	// 基于游戏配置计算
	if day < 0 || phase < 0 || phase >= len(d.config.DayPeriodSeconds) {
		return 0, 0, 0
	}

	// 计算从当天开始经过的时间
	elapsedSeconds := 0
	for i := 0; i < phase; i++ {
		if i < len(d.config.DayPeriodSeconds) {
			elapsedSeconds += d.config.DayPeriodSeconds[i]
		}
	}

	// 添加当前阶段的经过时间（模拟 - 在实际版本中这将被检测）
	currentPhaseElapsed := int(time.Now().Unix() % int64(d.config.DayPeriodSeconds[phase]))
	elapsedSeconds += currentPhaseElapsed

	elapsed = time.Duration(elapsedSeconds) * time.Second

	// 计算到下一次缩圈的时间
	if phase < len(d.config.DayPeriodSeconds) {
		shrinkSeconds := d.config.DayPeriodSeconds[phase] - currentPhaseElapsed
		shrink = time.Duration(shrinkSeconds) * time.Second
	}

	// 下一阶段的时间目前与缩圈时间相同
	nextPhase = shrink

	return elapsed, shrink, nextPhase
}
