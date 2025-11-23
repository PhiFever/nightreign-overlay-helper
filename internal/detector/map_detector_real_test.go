package detector

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PhiFever/nightreign-overlay-helper/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// 实际游戏截图测试套件
// 测试文件应放置在 data/test/map_detector/ 目录下

// 地形名称到 ID 的映射
var earthShiftingNameMap = map[string]int{
	"普通": 0,
	"雪山": 1,
	"火山": 2,
	"腐败": 3,
	"隐城": 5,
}

// 从文件名提取地形类型
func extractEarthShiftingFromFilename(filename string) (int, bool) {
	for name, id := range earthShiftingNameMap {
		if strings.Contains(filename, name) {
			return id, true
		}
	}
	return -1, false
}

// 判断文件名是否表示全屏截图
func isFullScreenImage(filename string) bool {
	return strings.Contains(filename, "全屏") ||
		strings.Contains(filename, "fullscreen") ||
		strings.Contains(filename, "full")
}

// TestMapDetectorWithRealScreenshots 使用实际游戏截图测试完整的地图检测流程
func TestMapDetectorWithRealScreenshots(t *testing.T) {
	// 查找测试图片
	testDir := utils.GetDataPath("test/map_detector")
	files, err := os.ReadDir(testDir)
	if err != nil {
		t.Skipf("测试目录不存在或无法访问: %s", testDir)
		return
	}

	// 过滤出图片文件
	imageFiles := []string{}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := filepath.Ext(file.Name())
		if ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
			imageFiles = append(imageFiles, file.Name())
		}
	}

	if len(imageFiles) == 0 {
		t.Skip("没有找到测试图片，请将游戏截图放到 data/test/map_detector/ 目录")
		return
	}

	t.Logf("找到 %d 个测试图片", len(imageFiles))

	// 创建地图检测器
	detector, err := NewMapDetector()
	if err != nil {
		t.Fatalf("创建地图检测器失败: %v", err)
	}

	// 对每个图片进行测试
	for _, filename := range imageFiles {
		t.Run(filename, func(t *testing.T) {
			testMapDetectionWithScreenshot(t, detector, testDir, filename)
		})
	}
}

// testMapDetectionWithScreenshot 测试单个截图的地图检测
func testMapDetectionWithScreenshot(t *testing.T, detector *MapDetector, testDir, filename string) {
	filePath := filepath.Join(testDir, filename)

	// 加载图片
	img, err := LoadImageFromFile(filePath)
	if err != nil {
		t.Fatalf("加载图片失败 %s: %v", filename, err)
	}

	bounds := img.Bounds()
	t.Logf("图片尺寸: %dx%d", bounds.Dx(), bounds.Dy())

	// 尝试从文件名提取期望的地形类型
	expectedEarth, hasExpected := extractEarthShiftingFromFilename(filename)
	if hasExpected {
		t.Logf("期望地形: %d (从文件名识别)", expectedEarth)
	}

	// 执行地图检测
	result, err := detector.Detect(img)
	if err != nil {
		t.Errorf("地图检测失败: %v", err)
		return
	}

	// 验证结果
	if result == nil {
		t.Error("检测结果为空")
		return
	}

	mapResult, ok := result.(*MapDetectResult)
	if !ok {
		t.Error("结果类型不正确")
		return
	}

	// 输出检测结果
	t.Logf("检测地形: %d (分数: %.4f)", mapResult.EarthShifting, mapResult.EarthShiftingScore)

	// 如果有期望值，进行验证
	if hasExpected {
		if mapResult.EarthShifting == expectedEarth {
			t.Logf("✓ 地形检测正确！")
		} else {
			t.Errorf("✗ 地形检测错误: 期望 %d, 实际 %d", expectedEarth, mapResult.EarthShifting)
		}
	}

	if mapResult.Pattern != nil {
		t.Logf("匹配的地图模式: #%d", mapResult.Pattern.ID)
		t.Logf("  - 地形: %d", mapResult.Pattern.EarthShifting)
		t.Logf("  - Day1 BOSS: %d (位置: %v)", mapResult.Pattern.Day1Boss, mapResult.Pattern.Day1Pos)
		t.Logf("  - Day2 BOSS: %d (位置: %v)", mapResult.Pattern.Day2Boss, mapResult.Pattern.Day2Pos)
		t.Logf("  - 宝藏: %d", mapResult.Pattern.Treasure)
	}

	// 基本验证
	assert.GreaterOrEqual(t, mapResult.EarthShifting, 0, "地形ID应该有效")
	assert.LessOrEqual(t, mapResult.EarthShifting, 5, "地形ID应该在有效范围")
}

// TestMapRegionDetectionWithRealScreenshots 测试地图区域定位功能
func TestMapRegionDetectionWithRealScreenshots(t *testing.T) {
	testDir := utils.GetDataPath("test/map_detector")
	files, err := os.ReadDir(testDir)
	if err != nil {
		t.Skipf("测试目录不存在: %s", testDir)
		return
	}

	// 过滤全屏截图
	fullScreenImages := []string{}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		ext := filepath.Ext(name)
		// 查找包含 "全屏"、"fullscreen" 或 "full" 的图片
		if (ext == ".png" || ext == ".jpg" || ext == ".jpeg") && isFullScreenImage(name) {
			fullScreenImages = append(fullScreenImages, name)
		}
	}

	if len(fullScreenImages) == 0 {
		t.Skip("没有找到全屏截图（文件名应包含'全屏'或'fullscreen'）")
		return
	}

	t.Logf("找到 %d 个全屏截图", len(fullScreenImages))

	regionDetector := NewMapRegionDetector()

	for _, filename := range fullScreenImages {
		t.Run(filename, func(t *testing.T) {
			testMapRegionDetection(t, regionDetector, testDir, filename)
		})
	}
}

// testMapRegionDetection 测试单个截图的地图区域定位
func testMapRegionDetection(t *testing.T, detector *MapRegionDetector, testDir, filename string) {
	filePath := filepath.Join(testDir, filename)

	// 加载图片
	img, err := LoadImageFromFile(filePath)
	if err != nil {
		t.Fatalf("加载图片失败: %v", err)
	}

	bounds := img.Bounds()
	screenWidth, screenHeight := bounds.Dx(), bounds.Dy()
	t.Logf("屏幕尺寸: %dx%d", screenWidth, screenHeight)

	// 步骤1: 尝试检测小地图圆形
	t.Log("步骤1: 检测小地图...")
	minimap, err := FindMiniMapCircle(img)
	if err != nil {
		t.Errorf("小地图检测出错: %v", err)
	}

	if minimap != nil {
		t.Logf("✓ 检测到小地图:")
		t.Logf("  - 位置: (%d, %d)", minimap.X, minimap.Y)
		t.Logf("  - 半径: %d", minimap.Radius)
		t.Logf("  - 置信度: %.4f", minimap.Score)

		// 验证位置合理性（应该在左下角）
		assert.Less(t, minimap.X, screenWidth/2, "小地图应该在屏幕左侧")
		assert.Greater(t, minimap.Y, screenHeight/2, "小地图应该在屏幕下方")
	} else {
		t.Log("✗ 未检测到小地图，将使用fallback")
	}

	// 步骤2: 计算地图区域
	t.Log("步骤2: 计算地图区域...")
	mapRegion := CalculateMapRegionFromMiniMap(screenWidth, screenHeight, minimap)
	t.Logf("计算出的地图区域:")
	t.Logf("  - X: %d, Y: %d", mapRegion.X, mapRegion.Y)
	t.Logf("  - 宽度: %d, 高度: %d", mapRegion.Width, mapRegion.Height)
	t.Logf("  - 占屏幕比例: %.1f%% x %.1f%%",
		float64(mapRegion.Width)/float64(screenWidth)*100,
		float64(mapRegion.Height)/float64(screenHeight)*100)

	// 验证区域合理性
	assert.Greater(t, mapRegion.Width, 0, "地图宽度应该大于0")
	assert.Greater(t, mapRegion.Height, 0, "地图高度应该大于0")
	assert.LessOrEqual(t, mapRegion.X+mapRegion.Width, screenWidth, "地图区域不应超出屏幕")
	assert.LessOrEqual(t, mapRegion.Y+mapRegion.Height, screenHeight, "地图区域不应超出屏幕")

	// 步骤3: 验证地图区域
	t.Log("步骤3: 验证地图内容...")
	isValid := VerifyMapRegion(img, mapRegion)
	t.Logf("地图区域验证结果: %v", isValid)

	// 步骤4: 提取地图区域
	t.Log("步骤4: 提取地图区域...")
	mapImg, success := detector.ExtractMapRegion(img)
	assert.NotNil(t, mapImg, "提取的地图图像不应为空")

	extractedBounds := mapImg.Bounds()
	t.Logf("提取的地图尺寸: %dx%d (成功: %v)",
		extractedBounds.Dx(), extractedBounds.Dy(), success)
}

// TestEarthShiftingDetectionAccuracy 测试地形检测准确率
func TestEarthShiftingDetectionAccuracy(t *testing.T) {
	testDir := utils.GetDataPath("test/map_detector")
	files, err := os.ReadDir(testDir)
	if err != nil {
		t.Skipf("测试目录不存在: %s", testDir)
		return
	}

	// 自动从文件名构建已知地形的测试用例
	knownEarthShifting := make(map[string]int)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filename := file.Name()
		ext := filepath.Ext(filename)
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			continue
		}

		// 从文件名提取地形类型
		if earthID, ok := extractEarthShiftingFromFilename(filename); ok {
			knownEarthShifting[filename] = earthID
		}
	}

	if len(knownEarthShifting) == 0 {
		t.Skip("没有找到包含地形名称的测试图片（支持：普通、雪山、火山、腐败、隐城）")
		return
	}

	t.Logf("找到 %d 个包含已知地形标签的测试图片", len(knownEarthShifting))

	detector, err := NewMapDetector()
	if err != nil {
		t.Fatalf("创建检测器失败: %v", err)
	}

	correctCount := 0
	totalCount := 0

	// 按地形分组统计
	earthStats := make(map[int]struct{ correct, total int })

	for filename, expectedEarth := range knownEarthShifting {
		t.Run(filename, func(t *testing.T) {
			filePath := filepath.Join(testDir, filename)

			// 加载图片
			img, err := LoadImageFromFile(filePath)
			if err != nil {
				t.Fatalf("加载图片失败: %v", err)
			}

			// 执行检测
			result, err := detector.Detect(img)
			if err != nil {
				t.Errorf("检测失败: %v", err)
				return
			}

			mapResult := result.(*MapDetectResult)
			detected := mapResult.EarthShifting

			totalCount++
			stats := earthStats[expectedEarth]
			stats.total++

			if detected == expectedEarth {
				correctCount++
				stats.correct++
				t.Logf("✓ 正确: 期望 %d, 检测到 %d (分数: %.4f)",
					expectedEarth, detected, mapResult.EarthShiftingScore)
			} else {
				t.Errorf("✗ 错误: 期望 %d, 但检测到 %d (分数: %.4f)",
					expectedEarth, detected, mapResult.EarthShiftingScore)
			}

			earthStats[expectedEarth] = stats
		})
	}

	// 输出总体准确率
	if totalCount > 0 {
		accuracy := float64(correctCount) / float64(totalCount) * 100
		t.Logf("\n=== 地形检测准确率统计 ===")
		t.Logf("总体准确率: %.1f%% (%d/%d)", accuracy, correctCount, totalCount)

		// 输出各地形的准确率
		earthNames := map[int]string{0: "普通", 1: "雪山", 2: "火山", 3: "腐败", 5: "隐城"}
		for earthID := 0; earthID <= 5; earthID++ {
			if stats, ok := earthStats[earthID]; ok && stats.total > 0 {
				acc := float64(stats.correct) / float64(stats.total) * 100
				t.Logf("  %s(ID=%d): %.1f%% (%d/%d)",
					earthNames[earthID], earthID, acc, stats.correct, stats.total)
			}
		}
	}
}

// TestCircleDetectionVisualization 圆形检测可视化测试
// 这个测试会保存带有检测结果标记的图片，方便调试
func TestCircleDetectionVisualization(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过可视化测试（使用 -short 标志）")
	}

	testDir := utils.GetDataPath("test/map_detector")
	files, err := os.ReadDir(testDir)
	if err != nil {
		t.Skipf("测试目录不存在: %s", testDir)
		return
	}

	// 只测试第一个图片（避免生成太多调试图片）
	var testFile string
	for _, file := range files {
		if !file.IsDir() {
			ext := filepath.Ext(file.Name())
			if ext == ".png" || ext == ".jpg" {
				testFile = file.Name()
				break
			}
		}
	}

	if testFile == "" {
		t.Skip("没有找到测试图片")
		return
	}

	filePath := filepath.Join(testDir, testFile)
	img, err := LoadImageFromFile(filePath)
	if err != nil {
		t.Fatalf("加载图片失败: %v", err)
	}

	bounds := img.Bounds()
	t.Logf("测试图片: %s (%dx%d)", testFile, bounds.Dx(), bounds.Dy())

	// 检测圆形
	minimap, err := FindMiniMapCircle(img)
	if err != nil {
		t.Errorf("圆形检测失败: %v", err)
		return
	}

	if minimap == nil {
		t.Log("未检测到圆形")
		return
	}

	t.Logf("检测到圆形:")
	t.Logf("  - 中心: (%d, %d)", minimap.X, minimap.Y)
	t.Logf("  - 半径: %d", minimap.Radius)
	t.Logf("  - 置信度: %.4f", minimap.Score)

	// TODO: 可以在这里添加保存标记图片的代码
	// 例如在图片上画出检测到的圆形，保存到 data/test/map_detector/debug_圆形.png
}

// BenchmarkMapDetectionWithRealImage 使用真实图片的性能基准测试
func BenchmarkMapDetectionWithRealImage(b *testing.B) {
	testDir := utils.GetDataPath("test/map_detector")
	files, err := os.ReadDir(testDir)
	if err != nil {
		b.Skip("测试目录不存在")
		return
	}

	// 找到第一个测试图片
	var testFile string
	for _, file := range files {
		if !file.IsDir() {
			ext := filepath.Ext(file.Name())
			if ext == ".png" || ext == ".jpg" {
				testFile = file.Name()
				break
			}
		}
	}

	if testFile == "" {
		b.Skip("没有找到测试图片")
		return
	}

	// 加载图片
	filePath := filepath.Join(testDir, testFile)
	img, err := LoadImageFromFile(filePath)
	if err != nil {
		b.Fatalf("加载图片失败: %v", err)
	}

	// 创建检测器
	detector, err := NewMapDetector()
	if err != nil {
		b.Fatalf("创建检测器失败: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	// 运行基准测试
	for i := 0; i < b.N; i++ {
		_, err := detector.Detect(img)
		if err != nil {
			b.Fatalf("检测失败: %v", err)
		}
	}
}

// 辅助函数：加载图片的便捷方法
func loadTestImage(t *testing.T, filename string) image.Image {
	testDir := utils.GetDataPath("test/map_detector")
	filePath := filepath.Join(testDir, filename)

	img, err := LoadImageFromFile(filePath)
	if err != nil {
		t.Fatalf("加载图片失败 %s: %v", filename, err)
	}

	return img
}

// 辅助函数：打印检测结果的详细信息
func printDetectionResult(t *testing.T, result *MapDetectResult, detector *MapDetector) {
	t.Log("=" + "=" + "=" + " 检测结果详情 " + "=" + "=" + "=")
	t.Logf("地形: %d (置信度: %.4f)", result.EarthShifting, result.EarthShiftingScore)

	if result.Pattern != nil {
		p := result.Pattern
		t.Logf("\n地图模式 #%d:", p.ID)
		t.Logf("  夜君: %s", detector.info.GetName(p.NightLord+100000))
		t.Logf("  地形: %d", p.EarthShifting)
		t.Logf("  起始位置: %v", p.StartPos)
		t.Logf("  Day1 BOSS: %s @ %v",
			detector.info.GetName(p.Day1Boss), p.Day1Pos)
		if p.Day1ExtraBoss != -1 {
			t.Logf("    额外BOSS: %s", detector.info.GetName(p.Day1ExtraBoss))
		}
		t.Logf("  Day2 BOSS: %s @ %v",
			detector.info.GetName(p.Day2Boss), p.Day2Pos)
		if p.Day2ExtraBoss != -1 {
			t.Logf("    额外BOSS: %s", detector.info.GetName(p.Day2ExtraBoss))
		}
		t.Logf("  宝藏: %d", p.Treasure)
		if p.RotRew != 0 {
			t.Logf("  腐败庇佑: %d", p.RotRew)
		}
		if p.EventValue != 0 {
			t.Logf("  特殊事件: flag=%d, value=%d", p.EventFlag, p.EventValue)
		}
		t.Logf("  建筑物数量: %d", len(p.PosConstructs))
	} else {
		t.Log("未匹配到地图模式")
	}
	t.Log("=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=")
}
