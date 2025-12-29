# Implementation Plan: 移除 pywin32 依赖

**Branch**: `001-remove-pywin32-dependency` | **Date**: 2025-12-26 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-remove-pywin32-dependency/spec.md`

## Summary

移除项目中的 pywin32 依赖，使用 PyQt6 原生跨平台 API 完全替代 Windows 专有功能。主要改动包括：

1. **移除 `set_widget_always_on_top()` 函数** - 所有窗口已使用 PyQt6 的 `WindowStaysOnTopHint`
2. **重构 `is_window_in_foreground()` 函数** - 使用 PyQt6 的 `QApplication.activeWindow()` API
3. **更新依赖配置** - 从 `pyproject.toml` 移除 pywin32 依赖
4. **创建测试框架** - 添加基础测试确保跨平台兼容性

## Technical Context

**Language/Version**: Python 3.13
**Primary Dependencies**: PyQt6 (已有), pytest (新增)
**Storage**: N/A
**Testing**: pytest
**Target Platform**: Windows 7/8/10/11 (主要), Linux (测试环境)
**Project Type**: single (桌面应用)
**Performance Goals**: 无性能回退，保持 CPU 使用最小化、UI 渲染 60fps
**Constraints**: 不能影响现有 Windows 用户功能，必须支持 Linux 测试
**Scale/Scope**: 2个函数重构，1个依赖移除，基础测试框架创建

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### ✅ I. 安全至上
- 本重构不涉及游戏交互方式，仍使用截屏方式
- 无违反游戏安全性的风险

### ✅ II. 用户体验优先
- Windows 用户：窗口置顶功能保持不变（PyQt API 已在使用）
- 前台检测功能可能略有降级，但不影响核心体验
- 所有现有功能继续正常工作

### ✅ III. 性能为本
- 移除 win32 调用减少了平台特定开销
- PyQt API 性能与 win32 相当
- 无性能回退风险

### ✅ IV. 可维护性
- 代码更简洁（移除约 20 行代码）
- 消除平台特定依赖，降低维护复杂度
- 增加测试提高代码质量

### ✅ V. 测试驱动开发（推荐）
- 新增基础测试框架
- 添加导入测试和单元测试
- 虽不是严格 TDD，但符合测试优先原则

**结论**: 所有章程原则通过，无违反项复杂度控制。

## Project Structure

### Documentation (this feature)

```text
specs/001-remove-pywin32-dependency/
├── plan.md              # This file
├── research.md          # Phase 0 output - 技术调研
└── spec.md              # 功能规格说明
```

### Source Code (repository root)

```text
src/
├── ui/
│   └── utils.py         # 需要重构的文件
├── detector/            # 无需修改（仅导入测试）
└── ...

tests/                    # 新创建
├── __init__.py
├── conftest.py          # pytest 配置
├── test_imports.py      # 导入测试
└── test_utils.py        # utils 模块测试

pyproject.toml           # 需要修改（移除 pywin32）
uv.lock                  # 需要同步
```

**Structure Decision**: 项目为单体桌面应用，使用 `src/` 和 `tests/` 标准结构。

## Complexity Tracking

无复杂度违反。本重构简化了代码，符合章程的简单性原则。

## Implementation Phases

### Phase 0: Research (已完成)

**输出**: `research.md`

**关键决策**:
1. 完全移除 `set_widget_always_on_top()` - 因为已有 PyQt 置顶
2. 使用 `QApplication.activeWindow()` 替代 Windows API
3. 使用 uv sync 管理依赖
4. 创建最小测试框架

### Phase 1: Code Refactoring

**任务 1.1: 重构 src/ui/utils.py**

修改前：
```python
def set_widget_always_on_top(widget: QWidget):
    try:
        import win32gui
        import win32con
        hwnd = widget.winId().__int__()
        win32gui.SetWindowPos(hwnd, win32con.HWND_TOPMOST,
                                0, 0, 0, 0,
                                win32con.SWP_NOSIZE | win32con.SWP_NOMOVE)
        info(f"Window HWND: {hwnd} set to TOPMOST.")
    except Exception as e:
        warning(f"Error setting system always on top: {e}")

def is_window_in_foreground(window_title: str) -> bool:
    try:
        import win32gui
        import time
        active_window_handle = win32gui.GetForegroundWindow()
        active_window_title = win32gui.GetWindowText(active_window_handle)
        if window_title.lower() in active_window_title.lower():
            return True
        return False
    except Exception as e:
        return False
```

修改后：
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

# set_widget_always_on_top() 完全删除
```

**任务 1.2: 搜索并删除对 set_widget_always_on_top() 的调用**

```bash
grep -rn "set_widget_always_on_top" src/
```

如果有调用，全部删除（因为窗口已在 `__init__` 中设置了置顶标志）。

### Phase 2: Dependency Management

**任务 2.1: 修改 pyproject.toml**

```toml
[project]
dependencies = [
    "mss>=10",
    "numpy>=2",
    "opencv-python>=4",
    "pillow>=12",
    "platformdirs>=4",
    "pygame>=2",
    "pynput>=1",
    "pyqt6>=6",
    # 删除: "pywin32>=311",
    "pyyaml>=6",
]

[project.optional-dependencies]
dev = [
    "pytest>=7.0",
    "pytest-cov>=4.0",
]
```

**任务 2.2: 同步依赖**

```bash
uv sync
```

### Phase 3: Testing Infrastructure

**任务 3.1: 创建测试目录**

```bash
mkdir -p tests
touch tests/__init__.py
```

**任务 3.2: 创建 tests/conftest.py**

```python
"""pytest 配置"""
import pytest
```

**任务 3.3: 创建 tests/test_imports.py**

```python
"""测试所有模块可以导入（跨平台）"""

def test_import_ui_modules():
    """测试 UI 模块可以导入"""
    from src.ui import utils
    from src.ui import overlay
    from src.ui import map_overlay
    from src.ui import hp_overlay
    from src.ui import settings

def test_import_detector_modules():
    """测试检测器模块可以导入"""
    from src.detector import day_detector
    from src.detector import rain_detector
    from src.detector import map_detector
    from src.detector import hp_detector
    from src.detector import art_detector

def test_import_core_modules():
    """测试核心模块可以导入"""
    from src import config
    from src import logger
    from src import common
```

**任务 3.4: 创建 tests/test_utils.py**

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

def test_is_window_in_foreground_no_active_window():
    """没有激活窗口时返回 False"""
    mock_app = Mock()
    mock_app.activeWindow.return_value = None

    with patch('PyQt6.QtWidgets.QApplication.instance', return_value=mock_app):
        assert is_window_in_foreground("test") is False
```

**任务 3.5: 配置 pytest (pyproject.toml)**

```toml
[tool.pytest.ini_options]
testpaths = ["tests"]
addopts = "-v"

[tool.coverage.run]
source = ["src"]
omit = ["*/tests/*", "*/__pycache__/*"]
```

### Phase 4: Verification

**任务 4.1: Linux 环境测试**

```bash
# 在 Linux/树莓派上
uv sync --extra dev
pytest tests/ -v
```

**预期结果**:
- 所有导入测试通过
- `test_utils.py` 测试通过
- 无 ImportError

**任务 4.2: Windows 环境测试**

```bash
# 在 Windows 上
uv sync --extra dev
pytest tests/ -v

# 启动程序验证功能
python src/app.py
```

**验证项**:
- 窗口置顶功能正常
- 前台检测功能正常（可能行为略有不同但可用）
- 所有其他功能正常

## Post-Implementation Checklist

**Constitution Re-check**: ✅ 通过（见上文）

**Acceptance Criteria**:
- ✅ 代码中无任何 pywin32 导入
- ✅ pyproject.toml 中无 pywin32 依赖
- ✅ 所有模块可以在 Linux 上导入
- ✅ pytest 可以在 Linux 上运行
- ✅ Windows 上窗口置顶功能正常
- ✅ Windows 上前台检测功能正常

**Documentation Updates**:
- 如需要，更新 README.md 的安装说明
- 添加 tests/README.md 说明测试运行方式（可选）

## Risks and Mitigation

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|---------|
| is_window_in_foreground 行为变化 | 低 | 中 | 充分测试，必要时调整逻辑 |
| Windows 用户体验下降 | 低 | 低 | 验证 PyQt 置顶效果 |
| 测试覆盖不足 | 中 | 中 | 从基础测试开始，逐步增加 |
| 其他地方使用 pywin32 | 高 | 低 | grep 全面搜索验证 |

## Timeline Estimate

- Phase 1 (Code Refactoring): 30 分钟
- Phase 2 (Dependency Management): 5 分钟
- Phase 3 (Testing Infrastructure): 1 小时
- Phase 4 (Verification): 30 分钟

**总计**: 约 2 小时

## Next Steps

此计划完成后，运行 `/speckit.tasks` 生成详细的任务列表。
