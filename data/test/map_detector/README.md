# Map Detector 测试数据说明

## 目录结构

将测试截图放在这个目录下：
```
data/test/map_detector/
├── fullscreen_1.png          # 全屏截图（用于测试地图区域定位）
├── fullscreen_2.png
├── map_earth0_1.png          # 地形0的地图截图
├── map_earth1_1.png          # 地形1的地图截图
└── ...
```

## 测试图片要求

### 1. 全屏截图
- **用途**: 测试地图区域自动定位功能
- **命名**: 包含 "fullscreen" 或 "full"
- **要求**:
  - 必须是完整的游戏窗口截图
  - 包含可见的小地图圆形（左下角）
  - 建议分辨率: 1920x1080 或 2560x1440

### 2. 地图区域截图
- **用途**: 测试地形检测和模式匹配
- **命名**: `map_earth{X}_{N}.png`
  - `X`: 地形ID (0/1/2/3/5)
  - `N`: 序号
- **要求**:
  - 已裁剪到地图区域
  - 包含完整的地图内容
  - 清晰可见

## 运行测试

### 基本测试
```bash
# 运行所有实际截图测试
go test -v ./internal/detector -run "TestMapDetectorWithRealScreenshots"

# 测试地图区域定位
go test -v ./internal/detector -run "TestMapRegionDetectionWithRealScreenshots"

# 测试地形检测准确率（需要配置已知地形）
go test -v ./internal/detector -run "TestEarthShiftingDetectionAccuracy"
```

### 可视化测试
```bash
# 生成调试图片（标记检测到的圆形等）
go test -v ./internal/detector -run "TestCircleDetectionVisualization"
```

### 性能测试
```bash
# 使用真实图片的性能基准
go test -bench=BenchmarkMapDetectionWithRealImage ./internal/detector
```

## 配置已知地形测试

编辑 `map_detector_real_test.go` 中的 `TestEarthShiftingDetectionAccuracy` 函数：

```go
knownEarthShifting := map[string]int{
    "map_earth0_1.png": 0,  // 地形0
    "map_earth1_1.png": 1,  // 地形1
    "map_earth2_1.png": 2,  // 地形2
    "map_earth3_1.png": 3,  // 地形3
    "map_earth5_1.png": 5,  // 地形5
}
```

## 预期输出

成功的测试应该显示：
```
=== RUN   TestMapDetectorWithRealScreenshots
=== RUN   TestMapDetectorWithRealScreenshots/fullscreen_1.png
    图片尺寸: 1920x1080
    地形类型: 2 (分数: 12.5432)
    匹配的地图模式: #156
      - 地形: 2
      - Day1 BOSS: 4929 (位置: (100, 200))
      - Day2 BOSS: 4860 (位置: (300, 400))
      - 宝藏: 8005
--- PASS: TestMapDetectorWithRealScreenshots (2.34s)
```

## 调试技巧

1. **圆形检测失败**：
   - 检查小地图是否清晰可见
   - 确认左下角没有UI遮挡
   - 可以调整 `circle_detect.go` 中的阈值

2. **地形检测错误**：
   - 查看分数差异（应该有明显差距）
   - 检查图片是否包含完整地图
   - 确认地形背景图 `data/maps/{0-5}.jpg` 存在

3. **区域定位偏移**：
   - 使用可视化测试查看检测位置
   - 调整 `map_region.go` 中的位置计算参数

## 注意事项

- 测试图片不会被提交到git（已在.gitignore中排除）
- 建议准备至少每种地形2-3张截图
- 截图应该来自不同的游戏场景以提高测试覆盖率
