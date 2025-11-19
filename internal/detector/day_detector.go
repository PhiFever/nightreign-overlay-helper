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

// detectDay 使用固定比例从图像中检测当前天数
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

	// 使用固定比例提取方法 (不再需要智能策略)
	bounds := img.Bounds()
	fullRegion := NewRect(bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy())
	day, _ := d.matchDayInRegion(img, template, fullRegion)

	// 更新统计信息
	d.stats.LastDetectionTime = time.Since(startTime)
	d.stats.TotalDetections++
	d.stats.PredefinedCount++ // 统计为预定义区域方法

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
	// 新方法：直接按比例提取中央区域，无需模板匹配定位
	// 优势：更快、更稳定、避免模板匹配的子串问题

	// 裁剪到区域
	regionImg := CropImage(img, region)
	regionBounds := regionImg.Bounds()

	// 直接提取屏幕中央 1/3 区域 (横向33%-67%, 纵向33%-67%)
	// 确保这个区域正好包含 "DAY" 文本
	centerStartX := int(float64(regionBounds.Dx()) / 3.0)
	centerStartY := int(float64(regionBounds.Dy()) / 3.0)
	centerWidth := int(float64(regionBounds.Dx()) / 3.0)
	centerHeight := int(float64(regionBounds.Dy()) / 3.0)

	// 确保区域有效
	if centerWidth <= 0 || centerHeight <= 0 {
		logger.Warningf("[%s] Invalid center region dimensions", d.Name())
		return -1, nil
	}

	centerRegion := NewRect(centerStartX, centerStartY, centerWidth, centerHeight)
	centerImg := CropImage(regionImg, centerRegion)

	logger.Debugf("[%s] Extracted center region: x=%d, y=%d, w=%d, h=%d (from region %dx%d)",
		d.Name(), centerStartX, centerStartY, centerWidth, centerHeight,
		regionBounds.Dx(), regionBounds.Dy())

	// 在中央区域中提取罗马数字区域
	// 使用固定比例：罗马数字在中央区域的右侧部分
	centerBounds := centerImg.Bounds()

	// 使用固定比例提取罗马数字区域
	// "DAY" 文本结构: "DAY" 占约 70%, 罗马数字占约 30% 在最右侧
	// 参考旧实现的 75%-100% 提取策略
	numeralStartX := int(float64(centerBounds.Dx()) * 0.70)
	numeralWidth := int(float64(centerBounds.Dx()) * 0.30)  // 最右侧 30%
	numeralStartY := int(float64(centerBounds.Dy()) * 0.30) // 垂直居中
	numeralHeight := int(float64(centerBounds.Dy()) * 0.40)

	// 确保区域有效
	if numeralStartX < 0 {
		numeralStartX = 0
	}
	if numeralStartX+numeralWidth > centerBounds.Dx() {
		numeralWidth = centerBounds.Dx() - numeralStartX
	}
	if numeralStartY < 0 {
		numeralStartY = 0
	}
	if numeralStartY+numeralHeight > centerBounds.Dy() {
		numeralHeight = centerBounds.Dy() - numeralStartY
	}

	if numeralWidth <= 10 || numeralHeight <= 10 {
		logger.Warningf("[%s] Numeral region too small (w=%d, h=%d), skipping detection",
			d.Name(), numeralWidth, numeralHeight)
		return -1, nil
	}

	numeralRegion := NewRect(numeralStartX, numeralStartY, numeralWidth, numeralHeight)

	logger.Debugf("[%s] Numeral region (relative to center): x=%d, y=%d, w=%d, h=%d",
		d.Name(), numeralRegion.X, numeralRegion.Y, numeralRegion.Width, numeralRegion.Height)

	// 从中央区域提取罗马数字图像
	numeralImg := CropImage(centerImg, numeralRegion)

	// 优先尝试 OCR（最快最准确）
	logger.Infof("[%s] 🔍 OCR support compiled: %v", d.Name(), OCRAvailable)
	if OCRAvailable {
		logger.Infof("[%s] 🚀 Trying OCR detection on numeral region (%dx%d)...",
			d.Name(), numeralRegion.Width, numeralRegion.Height)
		dayNum, err := OCRExtractDayNumber(numeralImg)
		if err == nil && dayNum >= 1 && dayNum <= 3 {
			logger.Infof("[%s] ✅ OCR detection succeeded: Day %d", d.Name(), dayNum)
			location := &Point{
				X: region.X + centerStartX,
				Y: region.Y + centerStartY,
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

	// 计算绝对位置 (相对于原始图像)
	// region起点 + 中央区域起点
	location := &Point{
		X: region.X + centerStartX,
		Y: region.Y + centerStartY,
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
