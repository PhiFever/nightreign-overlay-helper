# Implementation Plan: 单实例运行功能

**Branch**: `002-single-instance` | **Date**: 2025-12-26 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-single-instance/spec.md`

## Summary

实现单实例运行功能，防止用户短时间内多次点击导致多个程序实例和托盘图标。主要技术挑战包括：
1. 跨平台单实例检测（Windows 和 Linux）
2. 进程间通信（通知现有实例）
3. 失效锁的自动检测和清理
4. 每用户独立的单实例控制

## Technical Context

**Language/Version**: Python 3.13
**Primary Dependencies**: PyQt6 (已有), QSharedMemory/QLockFile (PyQt6 内置)
**Storage**: 临时锁文件在用户临时目录 (platformdirs 管理)
**Testing**: pytest (已有测试框架)
**Target Platform**: Windows 7/8/10/11 (主要), Linux (测试环境)
**Project Type**: single (桌面应用)
**Performance Goals**: 单实例检测 <10ms，进程间通信 <50ms，不影响启动时间
**Constraints**: 必须在 QApplication 创建之前执行检测，异常退出后锁自动失效
**Scale/Scope**: 单文件模块 (~200 行)，集成到现有 app.py 启动流程

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### ✅ I. 安全至上
- 本功能不涉及游戏交互
- 仅涉及本地进程管理和锁文件
- 无违反游戏安全性的风险

### ✅ II. 用户体验优先
- 解决用户报告的核心问题（多个托盘图标）
- 提供明确的视觉反馈（窗口置顶/通知）
- 异常退出后不阻止用户重启程序
- 符合用户体验优先原则

### ✅ III. 性能为本
- 单实例检测在启动早期执行，延迟 <10ms
- 不影响主程序启动性能
- 内存占用增加 <1MB（锁文件和 IPC 机制）
- 符合性能为本原则

### ✅ IV. 可维护性
- 单独模块实现（src/single_instance.py）
- 清晰的接口和错误处理
- 平台特定代码隔离
- 符合可维护性原则

### ✅ V. 测试驱动开发（推荐）
- 创建单实例测试（tests/test_single_instance.py）
- 测试正常启动、重复启动、异常退出等场景
- 符合测试驱动开发原则

**结论**: 所有章程原则通过，无违反项。

## Project Structure

### Documentation (this feature)

```text
specs/002-single-instance/
├── plan.md              # This file
├── research.md          # Phase 0 output - 技术调研
├── quickstart.md        # Phase 1 output - 快速测试指南
└── spec.md              # 功能规格说明
```

### Source Code (repository root)

```text
src/
├── single_instance.py   # 新增：单实例管理器
├── app.py               # 修改：集成单实例检测
├── ui/
│   ├── overlay.py       # 修改：添加窗口激活方法
│   ├── settings.py      # 修改：添加窗口激活方法
│   └── ...
└── ...

tests/
├── test_single_instance.py  # 新增：单实例测试
├── test_imports.py          # 已有
└── ...

pyproject.toml           # 无需修改（依赖已满足）
```

**Structure Decision**: 项目为单体桌面应用，使用 `src/` 和 `tests/` 标准结构。单实例功能作为独立模块 `single_instance.py` 实现，在 `app.py` 的 main() 函数最早期集成。

## Complexity Tracking

无复杂度违反。本功能是标准的单实例实现，符合章程的简单性原则。

## Implementation Phases

### Phase 0: Research (进行中)

**输出**: `research.md`

**调研内容**:
1. PyQt6 单实例实现的最佳实践（QSharedMemory vs QLockFile vs 文件锁）
2. 跨平台兼容性方案（Windows 和 Linux）
3. 进程间通信机制（QLocalSocket/QLocalServer）
4. 失效锁检测方法（进程 PID 检查）
5. 每用户单实例的实现方式

### Phase 1: Design

**任务 1.1: 核心架构设计**

创建 `SingleInstanceManager` 类：

```python
class SingleInstanceManager:
    """单实例管理器"""

    def __init__(self, app_id: str):
        """
        初始化单实例管理器

        Args:
            app_id: 应用唯一标识符（如 "nightreign-overlay-helper"）
        """
        self.app_id = app_id
        self.lock = None  # 锁对象（QLockFile 或 QSharedMemory）
        self.server = None  # IPC 服务器（QLocalServer）

    def try_acquire_lock(self) -> bool:
        """
        尝试获取单实例锁

        Returns:
            True 如果成功获取锁（首次启动），False 如果已有实例运行
        """
        pass

    def notify_existing_instance(self) -> bool:
        """
        通知现有实例（发送激活信号）

        Returns:
            True 如果成功通知现有实例
        """
        pass

    def setup_ipc_server(self, callback):
        """
        设置 IPC 服务器，监听来自新实例的通知

        Args:
            callback: 收到通知时的回调函数（激活窗口）
        """
        pass

    def release_lock(self):
        """释放单实例锁（程序退出时）"""
        pass
```

**任务 1.2: 集成到 app.py**

修改 `src/app.py` 的 main() 函数：

```python
def main():
    # Step 1: 在创建 QApplication 之前进行单实例检测
    from src.single_instance import SingleInstanceManager

    app_id = "nightreign-overlay-helper"  # 唯一标识符
    instance_manager = SingleInstanceManager(app_id)

    if not instance_manager.try_acquire_lock():
        # 已有实例运行，通知现有实例并退出
        info("Another instance is already running. Activating existing instance...")
        instance_manager.notify_existing_instance()
        sys.exit(0)

    # Step 2: 创建 QApplication（仅在首次启动时）
    app = QApplication(sys.argv)

    # Step 3: 设置 IPC 服务器，监听来自新实例的激活请求
    def on_activation_requested():
        """激活主窗口的回调函数"""
        # 显示主窗口或托盘通知
        if settings_window and settings_window.isVisible():
            settings_window.activateWindow()
            settings_window.raise_()
        else:
            # 显示托盘通知
            tray_icon.showMessage(
                APP_FULLNAME,
                "程序已在运行",
                QSystemTrayIcon.MessageIcon.Information,
                2000
            )

    instance_manager.setup_ipc_server(on_activation_requested)

    # Step 4: 继续正常的启动流程
    # ... (现有代码)

    # Step 5: 程序退出时释放锁
    try:
        exit_code = app.exec()
    finally:
        instance_manager.release_lock()

    sys.exit(exit_code)
```

**任务 1.3: 窗口激活增强**

为 `SettingsWindow` 添加激活方法：

```python
# src/ui/settings.py

def activateWindow(self):
    """激活窗口（从最小化恢复、置顶、获得焦点）"""
    if self.isMinimized():
        self.showNormal()
    self.raise_()
    self.activateWindow()
    # Windows 平台窗口闪烁效果（可选）
    if sys.platform == 'win32':
        try:
            from PyQt6.QtWinExtras import QtWin
            QtWin.flashWindow(self)
        except ImportError:
            pass  # PyQt6 无 QtWinExtras，跳过闪烁效果
```

### Phase 2: Testing Strategy

**测试 2.1: 单元测试 (tests/test_single_instance.py)**

```python
def test_first_instance_acquires_lock():
    """测试首次启动时成功获取锁"""
    manager = SingleInstanceManager("test-app")
    assert manager.try_acquire_lock() is True
    manager.release_lock()

def test_second_instance_fails_to_acquire_lock():
    """测试第二个实例无法获取锁"""
    manager1 = SingleInstanceManager("test-app")
    manager1.try_acquire_lock()

    manager2 = SingleInstanceManager("test-app")
    assert manager2.try_acquire_lock() is False

    manager1.release_lock()

def test_lock_released_after_crash():
    """测试异常退出后锁自动失效"""
    # 模拟：创建锁 → 强制终止进程 → 新实例应该能获取锁
    pass
```

**测试 2.2: 集成测试**

手动测试场景：
1. 启动程序 → 再次双击 → 验证只有一个托盘图标，窗口获得焦点
2. 启动程序 → 最小化窗口 → 再次双击 → 验证窗口恢复显示
3. 启动程序 → 强制结束进程 → 再次启动 → 验证程序正常启动

### Phase 3: Platform-Specific Considerations

**Windows 平台**:
- 使用 QLockFile 或 QSharedMemory（推荐 QLockFile，更健壮）
- 锁文件路径：`%TEMP%\nightreign-overlay-helper.lock`
- IPC 通道：`\\.\pipe\nightreign-overlay-helper`

**Linux 平台**:
- 使用 QLockFile（与 Windows 一致）
- 锁文件路径：`/tmp/nightreign-overlay-helper-<uid>.lock`（包含用户 ID）
- IPC 通道：`/tmp/nightreign-overlay-helper-<uid>.socket`

**跨平台兼容性**:
```python
import os
import platformdirs

def get_lock_file_path(app_id: str) -> str:
    """获取锁文件路径（跨平台）"""
    runtime_dir = platformdirs.user_runtime_dir(appname=app_id, ensure_exists=True)
    return os.path.join(runtime_dir, f"{app_id}.lock")

def get_ipc_name(app_id: str) -> str:
    """获取 IPC 通道名称（跨平台）"""
    import getpass
    username = getpass.getuser()
    return f"{app_id}-{username}"
```

## Post-Implementation Checklist

**Constitution Re-check**: ✅ 通过（见上文）

**Acceptance Criteria**:
- ✅ 用户多次点击只启动一个进程
- ✅ 托盘区域只显示一个图标
- ✅ 现有实例的窗口获得焦点
- ✅ 异常退出后可以重新启动
- ✅ Windows 和 Linux 都能正常工作

**Documentation Updates**:
- 更新 README.md 说明单实例功能（可选）
- 添加 quickstart.md 说明如何测试单实例功能

## Risks and Mitigation

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|---------|---------|
| 锁文件路径权限不足 | 中 | 低 | 使用 platformdirs 获取用户临时目录，确保权限 |
| IPC 通信失败 | 低 | 低 | 降级：仅阻止启动，不激活窗口 |
| 锁文件损坏 | 中 | 低 | 检测锁文件有效性，必要时重建 |
| 跨平台兼容性问题 | 高 | 中 | 充分测试 Windows 和 Linux，使用 PyQt 跨平台 API |

## Timeline Estimate

- Phase 0 (Research): 30 分钟
- Phase 1 (Design & Implementation): 2 小时
- Phase 2 (Testing): 1 小时
- Phase 3 (Integration & Verification): 30 分钟

**总计**: 约 4 小时

## Next Steps

此计划完成后，运行 `/speckit.tasks` 生成详细的任务列表。
