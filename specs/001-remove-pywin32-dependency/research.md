# Research: 移除 pywin32 依赖

**Date**: 2025-12-26
**Feature**: 001-remove-pywin32-dependency
**Context**: 完全移除 pywin32 依赖，使用 PyQt6 原生跨平台 API 替代，实现在 Linux 环境运行单元测试。

## 当前状态分析

### pywin32 使用情况

**位置**: `src/ui/utils.py`

1. **set_widget_always_on_top()** (第6-16行)
   - 当前：使用 `win32gui.SetWindowPos()` 强制窗口置顶
   - 替代：**移除此函数**，因为所有窗口已使用 `Qt.WindowStaysOnTopHint`

2. **is_window_in_foreground()** (第19-32行)
   - 当前：使用 `win32gui.GetForegroundWindow()` 检测前台窗口
   - 替代：**使用 PyQt6 的 `QApplication.activeWindow()`**

### 关键发现：项目已使用 PyQt 置顶

所有窗口类**已经使用了 PyQt6 的 `WindowStaysOnTopHint`**：

```python
# src/ui/overlay.py, map_overlay.py, hp_overlay.py, settings.py, capture_region.py
self.setWindowFlags(
    Qt.WindowType.FramelessWindowHint |
    Qt.WindowType.WindowStaysOnTopHint |  # ✅ 已有跨平台置顶
    Qt.WindowType.Tool
)
```

**结论**: `set_widget_always_on_top()` 是**冗余**的，可以直接移除。

## 决策 1: 移除 set_widget_always_on_top()

### 选择: 完全移除此函数

**理由**:
1. 所有窗口在 `__init__` 中已设置 `WindowStaysOnTopHint`
2. PyQt 的置顶在 Windows、Linux、macOS 都有效
3. 简化代码，移除平台特定逻辑

**影响分析**:
需要搜索并删除对此函数的所有调用。

## 决策 2: 重构 is_window_in_foreground()

### 选择: 使用 PyQt6 原生 API

**决策**:
```python
def is_window_in_foreground(window_title: str) -> bool:
    """
    检查应用是否有激活的窗口（跨平台）。

    Args:
        window_title: 窗口标题（保留参数以兼容现有代码）

    Returns:
        bool: 应用是否有激活窗口
    """
    try:
        app = QApplication.instance()
        return app is not None and app.activeWindow() is not None
    except Exception as e:
        warning(f"Error checking window state: {e}")
        return False
```

**PyQt6 替代方案**:

| 方法 | 跨平台 | 准确性 | 说明 |
|------|--------|--------|------|
| `app.activeWindow()` | ✅ | 高 | 检查当前应用是否有激活窗口 |
| `widget.isActiveWindow()` | ✅ | 高 | 检查特定窗口是否激活 |
| `app.applicationState()` | ✅ | 中 | 检查应用状态（Qt 6.5+） |

**理由**:
1. **跨平台**: PyQt API 在所有平台一致
2. **足够用**: 对于游戏前台检测，简单判断应用是否激活即可
3. **无依赖**: 不需要任何平台特定库

## 决策 3: 依赖管理

### 选择: 从 pyproject.toml 移除 pywin32，使用 uv sync

**修改 `pyproject.toml`**:
```toml
[project]
name = "nightreign-overlay-helper"
version = "0.10.2"
requires-python = ">=3.13"
dependencies = [
    "mss>=10",
    "numpy>=2",
    "opencv-python>=4",
    "pillow>=12",
    "platformdirs>=4",
    "pygame>=2",
    "pynput>=1",
    "pyqt6>=6",
    # 移除: "pywin32>=311",  ← 完全删除
    "pyyaml>=6",
]

[project.optional-dependencies]
# 开发和测试依赖
dev = [
    "pytest>=7.0",
    "pytest-cov>=4.0",
]

[[tool.uv.index]]
url = "https://mirrors.cloud.tencent.com/pypi/simple/"
default = true
```

**依赖同步**:
```bash
# 修改 pyproject.toml 后，同步依赖
uv sync

# 安装开发依赖
uv sync --extra dev

# 锁定依赖（更新 uv.lock）
uv lock
```

**说明**:
- `uv sync`: 自动读取 `pyproject.toml`，安装依赖到虚拟环境
- `uv.lock`: uv 会自动更新锁文件
- 所有平台使用相同的依赖配置

## 决策 4: 测试策略

### 创建基础测试框架

**测试目录结构**:
```
tests/
├── __init__.py
├── conftest.py              # pytest 配置
├── test_imports.py          # 测试模块导入
└── test_utils.py            # 测试 utils 模块
```

**test_imports.py**:
```python
"""测试所有模块可以导入（跨平台）"""
def test_import_ui_modules():
    from src.ui import utils
    from src.ui import overlay
    from src.ui import map_overlay
    from src.ui import hp_overlay

def test_import_detector_modules():
    from src.detector import day_detector
    from src.detector import rain_detector
    from src.detector import map_detector
```

**test_utils.py**:
```python
"""测试 utils 模块"""
import pytest
from unittest.mock import Mock, patch
from src.ui.utils import is_window_in_foreground

def test_is_window_in_foreground_no_app():
    """没有 QApplication 时返回 False"""
    with patch('PyQt6.QtWidgets.QApplication.instance', return_value=None):
        assert is_window_in_foreground("test") is False

def test_is_window_in_foreground_with_active_window():
    """有激活窗口时返回 True"""
    mock_app = Mock()
    mock_app.activeWindow.return_value = Mock()

    with patch('PyQt6.QtWidgets.QApplication.instance', return_value=mock_app):
        assert is_window_in_foreground("test") is True
```

**pyproject.toml 测试配置**:
```toml
[tool.pytest.ini_options]
testpaths = ["tests"]
addopts = "-v"

[tool.coverage.run]
source = ["src"]
omit = ["*/tests/*"]
```

## 实施路线图

### Phase 1: 重构 src/ui/utils.py（30 分钟）
1. ✅ 移除 win32gui/win32con 导入
2. ✅ 删除 `set_widget_always_on_top()` 函数
3. ✅ 重写 `is_window_in_foreground()` 使用 PyQt API
4. ✅ 删除对 `set_widget_always_on_top()` 的所有调用

### Phase 2: 更新依赖（5 分钟）
1. ✅ 从 `pyproject.toml` 移除 pywin32
2. ✅ 运行 `uv sync` 同步依赖

### Phase 3: 创建测试（1 小时）
1. ✅ 创建 `tests/` 目录结构
2. ✅ 实现基础导入测试
3. ✅ 实现 utils 测试

### Phase 4: 验证（30 分钟）
1. ✅ 在 Linux 上运行 `pytest`
2. ✅ 在 Windows 上验证功能

**总估计时间**: 2 小时

## 重构后的代码

### src/ui/utils.py（简化版）

```python
from PyQt6.QtWidgets import QWidget, QApplication
from PyQt6.QtCore import Qt
from src.logger import info, warning, error


def is_window_in_foreground(window_title: str) -> bool:
    """
    检查应用是否有激活的窗口（跨平台）。

    Args:
        window_title: 窗口标题（保留参数以兼容现有代码）

    Returns:
        bool: 应用是否有激活窗口
    """
    try:
        app = QApplication.instance()
        return app is not None and app.activeWindow() is not None
    except Exception as e:
        warning(f"Error checking window state: {e}")
        return False


def get_qt_screen_by_mss_region(region: tuple[int]) -> QWidget:
    """根据 mss 的 region 获取对应的 QScreen 对象。"""
    x, y, w, h = region
    app: QApplication = QApplication.instance()
    screens = app.screens()
    for screen in screens:
        sx = screen.geometry().x()
        sy = screen.geometry().y()
        sw = screen.geometry().width()
        sh = screen.geometry().height()
        ratio = screen.devicePixelRatio()
        mss_sw = int(sw * ratio)
        mss_sh = int(sh * ratio)
        if sx <= x <= sx + mss_sw and sy <= y <= sy + mss_sh:
            return screen
    raise ValueError(f"Region {region} is out of all screen bounds")


def mss_region_to_qt_region(region: tuple[int]):
    """将 mss region 转换为 Qt region"""
    screen = get_qt_screen_by_mss_region(region)
    x, y, w, h = region
    sx = screen.geometry().x()
    sy = screen.geometry().y()
    ratio = screen.devicePixelRatio()
    qx = sx + int((x - sx) / ratio)
    qy = sy + int((y - sy) / ratio)
    qw = int(w / ratio)
    qh = int(h / ratio)
    return (qx, qy, qw, qh)


def process_region_to_adapt_scale(region: tuple[int], scale: float) -> tuple[int]:
    """处理一个 region 的大小，使其能够适配指定的缩放比例。"""
    x, y, w, h = region
    new_w = int(int(w / scale) * scale)
    new_h = int(int(h / scale) * scale)
    return [x, y, new_w, new_h]
```

## 风险分析

### 风险 1: is_window_in_foreground 行为变化
- **影响**: 低（用于游戏前台检测，降级影响可接受）
- **概率**: 中（PyQt API 可能不如 Windows API 准确）
- **缓解**: 保持函数签名兼容，添加日志记录

### 风险 2: Windows 用户体验下降
- **影响**: 低（Qt WindowStaysOnTopHint 在 Windows 上仍然有效）
- **概率**: 低（项目已使用 Qt 置顶作为主要方案）
- **缓解**: 充分测试 Windows 环境

## 成功标准

1. ✅ 代码中无任何 pywin32 导入
2. ✅ 所有模块可以在 Linux 上导入
3. ✅ 测试可以在 Linux 上运行
4. ✅ Windows 上窗口置顶功能正常
5. ✅ Windows 上前台检测功能正常（可能降级但可用）

## 参考资料

- [PyQt6 QApplication 文档](https://doc.qt.io/qtforpython-6/PySide6/QtWidgets/QApplication.html)
- [Qt WindowFlags 文档](https://doc.qt.io/qt-6/qt.html#WindowType-enum)
- [uv 依赖管理文档](https://github.com/astral-sh/uv)
- [pytest 文档](https://docs.pytest.org/)
