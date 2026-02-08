# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

Nightreign Overlay Helper 是一个基于 PyQt6 的 Windows 游戏叠加层工具，用于《ELDEN RING NIGHTREIGN》。提供倒计时、地图识别、血条标记等实时信息显示功能。

- Python 3.13，Windows 专用
- 界面和注释均为中文
- 许可证：Apache 2.0

## 常用命令

```bash
# 安装依赖
uv sync

# 从源码运行
uv run python src/app.py

# 构建 Windows 可执行文件（输出到 dist/nightreign-overlay-helper/）
.\build.bat
```

项目无测试框架和 lint 工具配置。

## 架构

### 线程模型

- **主线程**：Qt 事件循环 + UI 渲染
- **线程 1**：`InputWorker`（pynput 键盘/手柄监听）
- **线程 2**：`Updater`（检测循环 + 状态管理）

所有线程间通信通过 Qt signals 实现线程安全。

### 核心流程

`src/app.py` 入口 → 创建叠加层、输入监听、Updater、设置窗口 → 启动系统托盘 → 后台线程开始运行。

`src/updater.py` 是核心状态管理器，驱动所有检测器（通过 `DetectorManager`），管理游戏状态（日相、雨状态、计时器），通过 Qt signals 更新 UI。

### 检测系统（src/detector/）

所有检测器遵循统一接口：
```python
def detect(self, sct: MSS, param: DetectParam) -> DetectResult:
    # 截取屏幕区域 → OpenCV/PIL 图像处理 → 返回结构化结果
```

- **DayDetector**：模板匹配识别 "DAY X" 文字
- **RainDetector**：HLS 色彩空间分析血条区域
- **MapDetector**：POI 匹配的地图识别（最复杂，1000+ 行）
- **HpDetector**：边缘检测 + 长度追踪计算血量百分比
- **ArtDetector**：模板匹配技能图标

### UI 层（src/ui/）

叠加层模式：
```python
class SomeOverlay(QWidget):
    def update_ui_state(self, state: SomeUIState)  # 更新状态
    def timerEvent(self, event)                      # 50ms 周期刷新
```

- `OverlayWidget`：主计时器显示
- `MapOverlayWidget`：地图信息叠加层
- `HpOverlayWidget`：血条百分比标记
- `SettingsWindow`：设置界面

### 配置系统

- **运行时配置** `config.yaml`：游戏时序参数、检测阈值、UI 样式、颜色校准
- **用户设置** 保存至 `%APPDATA%/nightreign-overlay-helper/settings.yaml`：叠加层位置/大小/透明度、快捷键、检测区域
- **预设** 保存至 `%APPDATA%/preset_settings/`

`src/common.py` 提供路径和 YAML 工具函数，`src/config.py` 为 dataclass 配置结构。

### HDR 支持

各检测器独立处理 HDR：Day 检测做 HDR→SDR 转换，Map 识别用 CLAHE 归一化，HP/Rain 检测使用原始图像。

## CI/CD

- Tag push `v*` 触发 `.github/workflows/build.yml`：UV 安装 → 版本校验 → PyInstaller 构建 → 打包 ZIP → 创建 GitHub Release
- 手动触发 `.github/workflows/release.yml`：更新 pyproject.toml 版本 → 提交 → 创建 tag → 触发构建
- `scripts/ci_version.py` 处理 tag 到 PEP 440 版本号的转换
- Release 工作流需要 `RELEASE_PAT` secret

## 关键依赖

PyQt6（GUI）、mss（屏幕截取）、opencv-python（图像处理）、Pillow（图像操作）、numpy（数组运算）、pynput（输入监听）、pywin32（Windows API）、PyYAML（配置解析）

PyPI 源配置为腾讯云镜像。
