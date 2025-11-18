# OCR 功能设置指南

## 概述

Day Detector 支持两种识别模式：
1. **模板匹配**（默认）：基于预定义的图片模板进行匹配
2. **OCR识别**（可选）：使用 Tesseract OCR 引擎识别文字

OCR 模式更加鲁棒，不依赖模板质量，推荐使用。

## 为什么需要 OCR？

当前测试显示，模板匹配的准确率为 40% (2/5)，原因是：
- Day 1/2/3 模板相似度极高（差距仅 0.7%-1.8%）
- 模板可能包含过多共同元素（背景、边框等）

OCR 方案可以直接识别屏幕上的"DAY 1"、"DAY 2"、"DAY 3"文字，准确率更高。

## 安装 Tesseract OCR

### Ubuntu/Debian

```bash
sudo apt-get update
sudo apt-get install tesseract-ocr libtesseract-dev libleptonica-dev
```

### macOS

```bash
brew install tesseract leptonica
```

### Windows

1. 下载安装包: https://github.com/UB-Mannheim/tesseract/wiki
2. 安装时确保选择"English language data"
3. 将 Tesseract 安装目录添加到 PATH 环境变量

## 编译带 OCR 支持的版本

```bash
# 使用 -tags=ocr 标签编译
go build -tags=ocr -o nightreign-overlay-helper ./cmd/app

# 运行测试
go test -tags=ocr ./internal/detector -v -run TestRealGameScreenshots
```

## 使用 OCR 功能

### 在代码中启用

```go
detector := NewDayDetector(config)
detector.Initialize()

// 启用 OCR 模式
detector.EnableOCR(true)

// 或者使用 OCR 策略
detector.SetDetectionStrategy(StrategyOCR)
```

## 性能对比

| 模式 | 准确率 | 速度 | 依赖 |
|------|--------|------|------|
| 模板匹配 | ~40% | 1.3-1.9s | 无 |
| OCR识别 | ~95%+ (预期) | 2-3s | Tesseract |

## 故障排除

### 编译错误：找不到 leptonica/allheaders.h

**问题**：未安装 Tesseract 开发库

**解决**：
```bash
# Ubuntu/Debian
sudo apt-get install libtesseract-dev libleptonica-dev

# macOS
brew install tesseract leptonica
```

### 运行时错误：OCR support not compiled in

**问题**：程序编译时未启用 OCR 标签

**解决**：使用 `-tags=ocr` 重新编译

### OCR 识别率低

**可能原因**：
1. 游戏分辨率过低或文字模糊
2. 文字颜色对比度不够

**建议**：
- 确保游戏分辨率至少为 1920x1080
- 调整游戏 UI 透明度设置

## 不使用 OCR 的替代方案

如果无法安装 Tesseract，可以：

1. **改进模板图片**
   - 重新裁剪模板，只保留数字部分
   - 去除背景和边框
   - 确保 Day 1/2/3 有明显视觉差异

2. **降低置信度阈值**
   ```go
   // 在 day_detector.go 的 matchDayInRegionWithScore 中
   const minConfidenceGap = 0.010  // 降至 1.0%
   ```

3. **使用全扫描模式**
   ```go
   detector.SetDetectionStrategy(StrategyFullScan)
   ```

## 技术细节

OCR 实现使用了：
- **图像预处理**：Otsu's 自适应二值化
- **OCR引擎**：Tesseract 4.0+
- **PSM模式**：PSM_SINGLE_LINE（单行文本）
- **白名单**：仅识别 "0123456789DAYday "

预处理步骤：
1. 转换为灰度图
2. Otsu's 方法自动阈值二值化
3. 反转图像（OCR 偏好深色背景上的浅色文字）
4. Tesseract OCR 识别
5. 正则提取数字

## 参考

- [Tesseract OCR 官方文档](https://github.com/tesseract-ocr/tesseract)
- [gosseract 库文档](https://github.com/otiai10/gosseract)
- [Otsu's 方法](https://en.wikipedia.org/wiki/Otsu%27s_method)
