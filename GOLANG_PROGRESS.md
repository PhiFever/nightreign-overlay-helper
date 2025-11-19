# Golang 重构进度报告

## 概述

本文档记录 NightReign Overlay Helper 项目的 Golang 重构进度。

**当前版本**: v0.9.0
**重构启动时间**: 2025-11-17
**当前阶段**: Phase 2 - 检测器层 ⏳ (进行中)

---

## Phase 1: 核心基础设施 (已完成 ✅)

### 1.1 项目结构

```
nightreign-overlay-helper/
├── cmd/
│   └── app/
│       └── main.go              ✅ 主程序入口
├── internal/
│   ├── config/
│   │   ├── config.go            ✅ 配置结构定义
│   │   └── loader.go            ✅ 配置加载器
│   ├── logger/
│   │   └── logger.go            ✅ 日志系统
│   ├── detector/
│   │   ├── base.go              ✅ 检测器基础接口
│   │   ├── utils.go             ✅ 图像处理工具
│   │   └── day_detector.go      ✅ Day检测器(最小可用)
│   ├── updater/
│   │   └── updater.go           ✅ 更新调度器
│   ├── ui/                      ⏳ 待实现
│   └── input/                   ⏳ 待实现
├── pkg/
│   ├── version/
│   │   └── version.go           ✅ 版本信息
│   └── utils/
│       ├── path.go              ✅ 路径工具
│       ├── time.go              ✅ 时间工具
│       └── yaml.go              ✅ YAML工具
├── go.mod                       ✅
├── go.sum                       ✅
└── config.yaml                  ✅ (已存在)
```

### 1.2 已实现功能

#### 版本管理 (pkg/version)
- ✅ 应用名称和版本常量
- ✅ 作者信息
- ✅ 游戏窗口标题常量

#### 工具函数 (pkg/utils)
- ✅ `GetAssetPath()` - 获取资源文件路径
- ✅ `GetDataPath()` - 获取数据文件路径
- ✅ `GetAppDataPath()` - 获取应用数据路径
- ✅ `GetDesktopPath()` - 获取桌面路径
- ✅ `GetReadableTimeDelta()` - 时间格式化
- ✅ `LoadYAML()` - 加载YAML配置
- ✅ `SaveYAML()` - 保存YAML配置(原子写入)

#### 配置管理 (internal/config)
- ✅ 完整的配置结构体定义 (Config)
- ✅ 配置加载和保存功能
- ✅ 自动检测配置文件修改并重新加载
- ✅ 线程安全的全局配置访问

支持的配置项包括：
- 缩圈相关配置
- 检测器配置 (日期/血量/地图/绝招/雨天)
- UI样式配置
- 更新间隔配置

#### 日志系统 (internal/logger)
- ✅ 多级别日志 (DEBUG, INFO, WARNING, ERROR, CRITICAL)
- ✅ 同时输出到控制台和文件
- ✅ 自动创建日志目录
- ✅ 按日期命名日志文件
- ✅ 错误级别自动包含堆栈跟踪
- ✅ 线程安全

#### 主程序 (cmd/app/main.go)
- ✅ 基础程序入口
- ✅ 日志系统初始化
- ✅ 配置加载
- ✅ 程序信息输出
- ✅ 检测器注册和初始化
- ✅ Updater启动和管理
- ✅ 优雅关闭机制

### 1.3 编译和运行

```bash
# 下载依赖
go mod tidy

# 编译
go build -o nightreign-overlay-helper ./cmd/app

# 运行
./nightreign-overlay-helper
```

**运行输出示例**:
```
Starting 黑夜君临悬浮助手v0.9.0...
2025-11-17 15:54:56 [INFO] Logger initialized
2025-11-17 15:54:56 [INFO] Application: 黑夜君临悬浮助手v0.9.0
2025-11-17 15:54:56 [INFO] Version: 0.9.0
2025-11-17 15:54:56 [INFO] Author: NeuraXmy
2025-11-17 15:54:56 [INFO] Configuration loaded successfully
```

### 1.4 依赖项

当前依赖 (`go.mod`):
```go
module github.com/PhiFever/nightreign-overlay-helper

go 1.21

require gopkg.in/yaml.v3 v3.0.1
```

---

## Phase 2: 检测器层 (进行中 ⏳)

### 2.1 已实现功能

#### 检测器基础框架 (internal/detector) ✅
- ✅ **Detector 接口定义** (`base.go`)
  - `Detector` 核心接口：Name(), Detect(), Initialize(), Cleanup(), IsEnabled(), SetEnabled()
  - `BaseDetector` 基础实现：提供通用功能
  - `DetectorRegistry` 检测器注册表：管理所有检测器的生命周期
- ✅ **图像处理工具** (`utils.go`)
  - 几何类型：Point, Rect
  - 图像操作：CropImage(), ResizeImage()
  - 色彩空间转换：RGB2Gray(), RGB2HSV(), RGB2HLS()
  - 图像分析：CreateMask(), CountNonZero(), InRange(), CalculateSimilarity()
  - ✅ **模板匹配功能** (新增 2025-11-18)
    - `MatchResult`: 模板匹配结果结构
    - `TemplateMatch()`: 单区域模板匹配
    - `TemplateMatchMultiple()`: 多区域模板匹配
    - `extractROI()`: 提取感兴趣区域

#### Updater 系统 (internal/updater) ✅
- ✅ 检测循环调度器
- ✅ 多检测器并发执行
- ✅ 结果处理和缓存（避免重复日志）
- ✅ 优雅关闭机制
- ✅ 屏幕截图抽象接口

#### 屏幕截图系统 (pkg/screenshot) ✅ (新增 2025-11-18)
- ✅ **截图接口** (`screenshot.go`)
  - `Capturer` 接口：定义截图功能
  - `DefaultCapturer`: 基于 kbinani/screenshot 的实现
  - `CaptureScreen()`: 捕获整个屏幕
  - `CaptureRegion()`: 捕获指定区域
  - `GetDisplayCount()`: 获取显示器数量
  - `GetDisplayBounds()`: 获取显示器边界
- ✅ **完整的测试套件** (`screenshot_test.go`)
  - 截图功能测试 (支持 headless 环境)
  - 性能基准测试

#### Day Detector (日期检测) ✅ (增强 2025-11-18)
- ✅ **完整版本实现**
  - ✅ **模板加载系统**
    - 支持4种语言: 简体中文(chs)、繁体中文(cht)、英文(eng)、日文(jp)
    - 自动加载 Day 1/2/3 模板图片
    - `DayTemplate` 结构：管理每种语言的模板
    - `loadTemplates()`: 从 data/day_template 加载模板
    - `loadImageFromFile()`: PNG 图片加载
  - ✅ **真实图像识别**
    - 基于模板匹配的 Day 检测 (替换模拟数据)
    - `detectDay()`: 使用模板匹配识别天数
    - `detectDayMock()`: Mock 模式用于测试
    - 可配置的匹配阈值
  - ✅ **配置接口**
    - `SetLanguage()`: 设置检测语言
    - `EnableTemplateMatching()`: 启用/禁用模板匹配
    - `SetMatchThreshold()`: 设置相似度阈值
  - ✅ **时间计算**
    - Elapsed Time (已流逝时间)
    - Shrink Time (缩圈倒计时)
    - Next Phase Time (下阶段倒计时)
  - ✅ **其他功能**
    - Phase (阶段) 检测 - 当前使用模拟数据
    - 结果格式化输出
    - 速率限制（rate limiting）
    - 优雅降级 (模板加载失败时使用 mock 模式)

**运行输出示例**:
```
2025-11-18 01:57:55 [INFO] [Updater] Detection loop started (interval: 100ms)
2025-11-18 01:57:55 [INFO] [Updater] DayDetector: Day 3 Phase 3 | Elapsed: 11m55s | Shrink in: 2m5s | Next phase in: 2m5s
2025-11-18 01:58:00 [INFO] [Updater] DayDetector: Day 1 Phase 0 | Elapsed: 1m0s | Shrink in: 3m30s | Next phase in: 3m29s
2025-11-18 01:58:05 [INFO] [Updater] DayDetector: Day 1 Phase 1 | Elapsed: 5m35s | Shrink in: 1m55s | Next phase in: 1m55s
```

### 2.1.1 测试覆盖率

#### Day Detector 测试 (`day_detector_test.go`) ✅
- ✅ `TestDayTemplateLoading`: 模板文件加载测试
- ✅ `TestDayDetectorInitialization`: 初始化测试
- ✅ `TestDayDetectorDetect`: 检测功能测试
- ✅ `TestDayDetectorCalculateTimes`: 时间计算测试
- ✅ `TestDayDetectorEnableDisable`: 启用/禁用测试
- ✅ `TestDayDetectorTemplateLoading`: 模板加载功能测试
- ✅ `TestDayDetectorLanguageSwitch`: 语言切换测试
- ✅ `TestDayDetectorTemplateMatching`: 模板匹配模式测试
- ✅ `TestTemplateMatchFunction`: 模板匹配算法测试
- ✅ `TestImageProcessingUtils`: 图像处理工具测试
- ✅ `BenchmarkDayDetectorDetect`: 检测性能基准
- ✅ `BenchmarkTemplateMatch`: 模板匹配性能基准

**测试结果**:
```
✓ 所有测试通过
✓ 模板加载成功 (4种语言 x 3个数字 = 12个模板)
✓ 模板匹配精度: 100% (相似度 1.0000)
✓ 图像处理工具正常工作
```

#### Screenshot 测试 (`screenshot_test.go`) ✅
- ✅ `TestNewCapturer`: 创建截图器测试
- ✅ `TestGetDisplayCount`: 获取显示器数量测试
- ✅ `TestGetDisplayBounds`: 获取显示器边界测试
- ✅ `TestCaptureScreen`: 全屏截图测试
- ✅ `TestCaptureRegion`: 区域截图测试
- ✅ `BenchmarkCaptureScreen`: 全屏截图性能基准
- ✅ `BenchmarkCaptureRegion`: 区域截图性能基准

**测试结果**:
```
✓ 所有测试通过
✓ Headless 环境自动跳过显示相关测试
✓ 接口设计验证成功
```

### 2.2 待实现功能

1. **Day Detector 增强** ⏳
   - [x] ~~集成模板匹配（替换模拟数据）~~ ✅ 已完成
   - [x] ~~模板图片加载~~ ✅ 已完成
   - [x] ~~多语言支持 (简中/繁中/英文/日文)~~ ✅ 已完成
   - [ ] Phase 检测的真实实现 (当前仍使用 mock)
   - [ ] 真实屏幕截图集成到主程序
   - [ ] 优化模板匹配性能 (考虑多尺度匹配)

2. **HP Detector (血量检测)** ⏳
   - [ ] 血条区域识别
   - [ ] HLS色彩空间转换
   - [ ] 血量百分比计算

3. **Rain Detector (雨天检测)** ⏳
   - [ ] 血条颜色分析
   - [ ] 雨天状态判断

4. **Map Detector (地图检测)** ⏳
   - [ ] 霍夫圆检测
   - [ ] 地图模式识别
   - [ ] 特殊地形检测

5. **Art Detector (绝招检测)** ⏳
   - [ ] 技能图标模板匹配
   - [ ] 多角色支持
   - [ ] 技能时间计算

### 2.3 技术选型与依赖

#### 当前技术方案
- ✅ **图像处理**: 纯 Go 实现 (无需 OpenCV)
  - 优点: 无外部依赖，部署简单，跨平台
  - 缺点: 性能略低于 OpenCV
  - 状态: 满足当前需求
- ✅ **屏幕截图**: `github.com/kbinani/screenshot`
  - 跨平台支持 (Windows/Linux/macOS)
  - 简单易用的 API
  - 性能良好

#### 已集成依赖

```go
require (
    github.com/kbinani/screenshot v0.0.0-20250624051815-089614a94018 ✅
    github.com/stretchr/testify v1.11.1  // 测试框架 ✅
    gopkg.in/yaml.v3 v3.0.1              // YAML 配置 ✅
)
```

#### 未来可选优化
如果性能成为瓶颈，可以考虑：
```go
// 可选：高性能图像处理
gocv.io/x/gocv v0.35.0  // OpenCV 绑定
// 注意: 需要系统安装 OpenCV >= 4.6.0
```

---

## Phase 3-5: 后续计划

### Phase 3: UI层 (待规划)
- [ ] 选择GUI框架 (Fyne 或 Wails)
- [ ] 实现覆盖层窗口
- [ ] 实现设置界面
- [ ] 系统托盘集成

### Phase 4: 整合与优化 (待规划)
- [ ] 模块整合
- [ ] 性能优化
- [ ] 错误处理完善

### Phase 5: 测试与发布 (待规划)
- [ ] 单元测试
- [ ] 集成测试
- [ ] 跨平台编译
- [ ] 版本发布

---

## 技术对比

### Python vs Golang (已实现部分)

| 模块 | Python | Golang | 状态 |
|------|--------|--------|------|
| 配置管理 | PyYAML + dataclass | yaml.v3 | ✅ |
| 日志系统 | logging + 自定义 | 自定义logger | ✅ |
| 版本管理 | common.py | pkg/version | ✅ |
| 工具函数 | common.py | pkg/utils | ✅ |
| 检测器框架 | detector/base.py | detector/base.go | ✅ |
| 图像工具 | OpenCV (cv2) | 纯Go实现 | ✅ |
| 更新调度器 | updater.py | updater/updater.go | ✅ |
| Day检测器 | day_detector.py | day_detector.go | ⏳ (最小可用) |

### 代码量对比

| 模块 | Python (行) | Golang (行) | 减少/增加 |
|------|------------|------------|----------|
| 配置管理 | ~80 | ~120 | +50% (类型安全) |
| 日志系统 | ~60 | ~200 | +233% (功能增强) |
| 工具函数 | ~75 | ~100 | +33% |
| 检测器基础 | ~100 | ~130 | +30% |
| 图像工具 | ~150 | ~270 | +80% (无OpenCV依赖) |
| Updater | ~120 | ~245 | +104% (并发优化) |
| Day检测器 | ~200 | ~185 | -8% (简化版) |
| **总计** | ~785 | ~1,250 | +59% |

> 注: Golang代码行数较多主要是因为：
> 1. 显式类型定义和更完善的错误处理
> 2. 自实现图像处理工具（暂未使用OpenCV）
> 3. 更完善的并发控制和资源管理

---

## 性能指标 (预期)

| 指标 | Python | Golang (目标) | 提升 |
|------|--------|--------------|------|
| 启动时间 | 3-5s | <0.5s | 6-10x |
| 内存占用 | 150-200MB | 20-30MB | 5-7x |
| 配置加载 | ~50ms | <5ms | 10x |

---

## 下一步行动

### 当前优先级

1. **完善 Day Detector** 🎯
   - [x] ~~实现真实的模板匹配（替换模拟数据）~~ ✅ 2025-11-18
   - [x] ~~加载Day数字模板图片~~ ✅ 2025-11-18
   - [x] ~~实现多语言支持~~ ✅ 2025-11-18
   - [ ] 实现 Phase 检测的真实识别
   - [ ] 集成屏幕截图到 Day Detector
   - [ ] 性能优化和调优

2. **屏幕截图系统** ✅ 已完成 (2025-11-18)
   - [x] ~~选择截图库~~ ✅ 使用 kbinani/screenshot
   - [x] ~~实现截图接口~~ ✅
   - [x] ~~编写测试用例~~ ✅
   - [ ] 实现窗口检测（查找游戏窗口）
   - [ ] 集成到 Updater 系统

3. **实现 HP Detector**
   - [ ] 血条区域识别
   - [ ] 血量百分比计算
   - [ ] HLS色彩空间转换优化

### 技术选型讨论

**图像处理方案**:
- **方案A**: 使用 gocv (OpenCV绑定)
  - 优点: 功能完整，性能好
  - 缺点: 需要安装OpenCV依赖，部署复杂

- **方案B**: 使用纯Go库 (当前)
  - 优点: 无外部依赖，部署简单
  - 缺点: 功能有限，需要自己实现算法

**当前选择**: 方案B（纯Go实现基础功能）+ 后续评估是否需要方案A

---

## 参考资料

- [Go官方文档](https://go.dev/doc/)
- [gocv文档](https://gocv.io/)
- [原Python代码](./src/)
- [重构方案](./README.md)

---

**最后更新**: 2025-11-18
**负责人**: Claude Code
**状态**: Phase 2 进行中 ⏳

**本次更新内容** (2025-11-18 下午):
1. ✅ **Day Detector 性能优化**
   - 添加 stride-based 模板匹配 (3-4倍加速)
   - 优化图像缩放策略 (scale 0.25→0.5 保留更多细节)
   - 实现置信度检查机制 (防止模糊匹配)
   - 测试时间从 46秒降至 9秒 (**5倍提速**)

2. ✅ **智能检测策略优化**
   - 新增 `matchDayInRegionWithScore()` 返回相似度分数
   - 实现跨区域最佳匹配选择 (选择所有区域中相似度最高的)
   - 添加置信度差距检查 (要求最佳匹配比次佳高1.5%)
   - 预定义区域优先搜索 (屏幕中心优先)

3. ✅ **调试和诊断**
   - 添加详细的调试日志输出
   - 显示每个Day模板的相似度分数
   - 记录匹配策略和置信度差距

4. ⚠️ **发现的问题**
   - 当前准确率: 40% (2/5 测试通过)
   - Day 2/3 测试图片被误识别为 Day 1
   - 根本原因: Day 1/2/3 模板相似度极为接近 (差距仅 1.5-3%)
   - 可能是模板图片包含过多共同元素 (如背景、边框等)

**性能对比**:
| 指标 | 优化前 | 优化后 | 提升 |
|------|--------|--------|------|
| 测试总时间 | 46.4s | 9.0s | **5.2x** |
| 单图检测 | ~15-25s | ~1.3-1.9s | **10-15x** |
| 使用策略 | Pyramid (慢) | Predefined (快) | 策略优化 |

**关键成果**:
- 截图功能: 完整实现并测试
- 图像识别: 模板匹配精度 100% (相同模板测试)
- 模板加载: 支持 4 种语言共 12 个模板
- 测试覆盖: Day Detector (12个测试) + Screenshot (5个测试)
- 性能优化: 检测速度提升 5-15倍

**已确认问题** (2025-11-19 分析完成):

1. **根本原因**：eng 模板图片包含游戏场景背景噪声
   - eng_1.png: "DAY I" + 蓝色天空背景
   - eng_2.png: "DAY II" + 树枝/地面背景
   - eng_3.png: "DAY III" + 树木/石头背景

2. **子串匹配问题**："DAY I" 是 "DAY II" 和 "DAY III" 的子串
   - Day 1 模板在所有图片中都得分最高 (~0.91)
   - Day 2/3 模板得分较低 (~0.86-0.88)
   - 无法通过简单的阈值策略区分

3. **尝试的解决方案均失败**：
   - ✗ 边缘检测：太慢 (134秒) 且相似度太低 (<0.7)
   - ✗ 模板宽度优先：Day 3 总是被选中
   - ✗ 启发式匹配：无法找到有效的阈值 (2-3% 都无效)

**推荐解决方案**:

方案 A：**重新制作模板** (最佳)
- 使用纯色背景 (黑色或白色)
- 或使用透明背景 + 只保留文字
- 或只截取罗马数字部分 ("I", "II", "III")

方案 B：**改进 OCR 支持罗马数字**
- 修改 OCR whitelist 包含罗马数字 "IVX"
- 添加罗马数字识别逻辑

方案 C：**使用降采样比例区分** (临时方案)
- 利用模板宽度差异 (342 vs 513 vs 519)
- 在不同缩放比例下匹配，观察分数变化

**当前状态**: 测试失败率 60-80%，需要用户提供更好的模板或启用 OCR
