# Feature Specification: 移除 pywin32 依赖以支持跨平台测试

**Feature Branch**: `001-remove-pywin32-dependency`
**Created**: 2025-12-26
**Status**: Draft
**Input**: User description: "移除项目中的pywin32依赖，这样我就可以在linux设备上运行单元测试。记得不要影响原有功能"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 在 Linux 环境运行单元测试 (Priority: P1)

作为开发者，我希望能够在 Linux 开发环境（如树莓派）上运行项目的单元测试，而不必依赖 Windows 专有库，这样我可以在多种平台上进行开发和测试，提高开发效率。

**Why this priority**: 这是核心需求，直接解决了用户无法在 Linux 上运行测试的问题，是实现跨平台开发的基础。

**Independent Test**: 可以通过在 Linux 系统上执行 `pytest` 或其他测试命令来验证，如果测试框架能够成功导入所有模块而不报 `ImportError: No module named 'win32gui'` 等错误，则说明功能实现成功。

**Acceptance Scenarios**:

1. **Given** 开发者在 Linux 系统上，**When** 执行单元测试命令，**Then** 测试框架能够成功导入所有模块，不会因 pywin32 缺失而失败
2. **Given** 开发者在 Linux 系统上，**When** 运行包含检测器逻辑的测试，**Then** 测试能够正常执行并返回结果
3. **Given** 项目代码中有条件导入 Windows 专有功能，**When** 在 Linux 环境执行，**Then** 程序优雅降级，不会崩溃

---

### User Story 2 - Windows 环境功能完全保留 (Priority: P1)

作为最终用户，我在 Windows 系统上运行游戏辅助程序时，所有依赖 pywin32 的功能（窗口置顶、前台检测）应该继续正常工作，不能因为移除依赖而导致功能缺失。

**Why this priority**: 保持 Windows 用户体验不受影响是必须的，项目的主要运行环境仍是 Windows。

**Independent Test**: 可以通过在 Windows 系统上启动程序并测试窗口置顶和游戏前台检测功能来验证。具体测试步骤：启动程序 → 将其他窗口置于顶层 → 验证悬浮窗是否仍保持在最上方；切换游戏到前台/后台 → 验证程序是否正确检测到状态变化。

**Acceptance Scenarios**:

1. **Given** 用户在 Windows 系统上启动程序，**When** 悬浮窗显示，**Then** 窗口自动置顶且始终保持在其他窗口之上
2. **Given** 用户在 Windows 系统上运行程序，**When** 游戏窗口切换到前台/后台，**Then** 程序能够正确检测并响应（例如暂停/恢复检测）
3. **Given** 用户在 Windows 系统上，**When** pywin32 库可用，**Then** 所有依赖 Windows API 的功能正常启用

---

### User Story 3 - 非 Windows 环境优雅降级 (Priority: P2)

作为在非 Windows 系统上运行程序的用户（如测试或演示场景），当 Windows 专有功能不可用时，程序应该优雅降级，核心功能（检测器逻辑、数据处理）仍然可用，只是缺少窗口管理等平台特定功能。

**Why this priority**: 虽然生产环境是 Windows，但允许程序在其他平台上以降级模式运行有助于测试、CI/CD 和跨平台演示。

**Independent Test**: 可以通过在 macOS 或 Linux 上启动程序来验证。程序应该能够启动并显示主界面，检测器模块可以被导入和测试，只是窗口置顶等功能不生效（可以在日志中记录警告信息）。

**Acceptance Scenarios**:

1. **Given** 用户在非 Windows 系统上启动程序，**When** 程序初始化，**Then** 核心模块成功加载，只有平台特定功能显示警告信息
2. **Given** 用户在非 Windows 系统上运行测试，**When** 调用窗口置顶函数，**Then** 函数安全返回而不抛出异常，记录警告日志
3. **Given** 用户在非 Windows 系统上，**When** 调用前台窗口检测函数，**Then** 函数返回默认值（如始终返回 `False`）而不崩溃

---

### Edge Cases

- **当 Windows 系统上 pywin32 未安装时**: 程序应该优雅降级，记录警告日志，窗口置顶功能失效但其他功能正常
- **当条件导入失败时**: 程序不应崩溃，应该捕获 ImportError 并提供降级实现
- **当在 CI/CD 环境运行测试时**: 测试应该能够在 Linux 容器中成功执行，不依赖 Windows 库
- **当用户在 Wine 或 WSL 环境运行时**: 程序应该能够检测平台特性并选择合适的实现

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统必须将 pywin32 的导入改为条件导入，仅在 Windows 平台且库可用时导入
- **FR-002**: 系统必须在所有使用 pywin32 API 的地方提供平台检测逻辑（如 `platform.system() == 'Windows'`）
- **FR-003**: 系统必须为依赖 pywin32 的函数提供降级实现，在非 Windows 平台或库不可用时使用
- **FR-004**: 窗口置顶功能（`set_widget_always_on_top`）必须在 pywin32 不可用时优雅降级，使用 PyQt6 的跨平台 API（`setWindowFlags`）或记录警告
- **FR-005**: 前台窗口检测功能（`is_window_in_foreground`）必须在 pywin32 不可用时返回安全的默认值（如 `False` 或根据配置决定）
- **FR-006**: 所有单元测试必须能够在 Linux 环境运行而不抛出 ImportError
- **FR-007**: 系统必须在日志中记录平台特定功能的可用性状态（如 "pywin32 available: True/False"）
- **FR-008**: 系统必须保持现有的 Windows 用户体验，当 pywin32 可用时所有功能正常工作

### Assumptions

- 项目的主要运行环境仍然是 Windows，只是需要支持在 Linux 上运行测试
- PyQt6 提供的跨平台窗口置顶 API（`Qt.WindowStaysOnTopHint`）可以作为降级方案，虽然效果可能不如 Windows API 强制
- 前台窗口检测功能在非 Windows 平台可以被禁用或始终返回 False，不会严重影响测试逻辑
- 用户在 Linux 上主要运行单元测试，不需要完整的 GUI 功能

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 开发者能够在 Linux 系统上成功执行所有单元测试，测试通过率不低于 Windows 环境
- **SC-002**: 程序在 Windows 系统上的所有功能保持不变，窗口置顶和前台检测功能正常工作
- **SC-003**: 程序在非 Windows 系统上能够启动并加载核心模块，不因缺少 pywin32 而崩溃
- **SC-004**: CI/CD 流水线能够在 Linux 容器中成功运行测试，构建时间不超过原来的 120%
- **SC-005**: 代码中的条件导入和平台检测逻辑不增加超过 10% 的代码复杂度（通过 cyclomatic complexity 指标衡量）
