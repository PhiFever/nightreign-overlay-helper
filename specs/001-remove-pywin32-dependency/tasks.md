---
description: "移除 pywin32 依赖的任务列表"
---

# Tasks: 移除 pywin32 依赖

**Input**: 设计文档来自 `/specs/001-remove-pywin32-dependency/`
**Prerequisites**: plan.md, spec.md, research.md

**Tests**: 本功能包含测试任务（创建测试基础设施和单元测试）

**Organization**: 任务按用户故事分组，以支持每个故事的独立实现和测试

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可以并行运行（不同文件，无依赖关系）
- **[Story]**: 任务属于哪个用户故事（如 US1, US2, US3）
- 描述中包含确切的文件路径

## Path Conventions

- **单体项目**: 仓库根目录的 `src/`, `tests/`
- 路径基于 plan.md 中的项目结构

---

## Phase 1: Setup (共享基础设施)

**Purpose**: 项目结构验证和准备工作

- [X] T001 验证当前分支为 001-remove-pywin32-dependency
- [X] T002 验证项目结构符合 plan.md 中的定义

---

## Phase 2: Foundational (阻塞性先决条件)

**Purpose**: 在实现任何用户故事之前必须完成的核心工作

**⚠️ CRITICAL**: 在此阶段完成之前，不能开始任何用户故事的工作

- [X] T003 搜索整个代码库中的 pywin32 导入和使用情况（验证 research.md 中的分析）
- [X] T004 搜索对 set_widget_always_on_top() 函数的所有调用
- [X] T005 验证所有窗口类已使用 Qt.WindowStaysOnTopHint（确认可以安全删除 win32 API）

**Checkpoint**: 基础验证完成 - 可以开始用户故事实现

---

## Phase 3: User Story 1 - 在 Linux 环境运行单元测试 (Priority: P1) 🎯 MVP

**Goal**: 移除 pywin32 依赖，使开发者能够在 Linux 环境（如树莓派）运行单元测试

**Independent Test**: 在 Linux 系统上执行 `uv sync --extra dev && pytest tests/ -v`，所有模块成功导入且测试通过，无 ImportError

### 实现 User Story 1

#### 子阶段 1.1: 代码重构

- [X] T006 [P] [US1] 在 src/ui/utils.py 中删除 set_widget_always_on_top() 函数（第6-16行）
- [X] T007 [US1] 在 src/ui/utils.py 中重构 is_window_in_foreground() 函数，使用 QApplication.activeWindow() API（第19-32行）
- [X] T008 [US1] 删除对 set_widget_always_on_top() 的所有调用（基于 T004 的搜索结果）
- [X] T009 [US1] 确保 src/ui/utils.py 中没有任何 win32gui 或 win32con 导入

#### 子阶段 1.2: 依赖管理

- [X] T010 [US1] 从 pyproject.toml 的 dependencies 列表中移除 "pywin32>=311"
- [X] T011 [US1] 在 pyproject.toml 中添加 [project.optional-dependencies] dev 部分，包含 pytest>=7.0 和 pytest-cov>=4.0
- [X] T012 [US1] 在 pyproject.toml 中添加 [tool.pytest.ini_options] 配置（testpaths = ["tests"], addopts = "-v"）
- [X] T013 [US1] 在 pyproject.toml 中添加 [tool.coverage.run] 配置（source = ["src"], omit = ["*/tests/*", "*/__pycache__/*"]）
- [X] T014 [US1] 运行 uv sync 同步依赖

#### 子阶段 1.3: 测试基础设施创建

- [X] T015 [P] [US1] 创建 tests/ 目录和 tests/__init__.py 文件
- [X] T016 [P] [US1] 创建 tests/conftest.py（pytest 配置文件）
- [X] T017 [P] [US1] 创建 tests/test_imports.py，包含 test_import_ui_modules() 函数
- [X] T018 [P] [US1] 在 tests/test_imports.py 中添加 test_import_detector_modules() 函数
- [X] T019 [P] [US1] 在 tests/test_imports.py 中添加 test_import_core_modules() 函数
- [X] T020 [P] [US1] 创建 tests/test_utils.py，包含 test_is_window_in_foreground_no_app() 测试
- [X] T021 [P] [US1] 在 tests/test_utils.py 中添加 test_is_window_in_foreground_with_active_window() 测试
- [X] T022 [P] [US1] 在 tests/test_utils.py 中添加 test_is_window_in_foreground_no_active_window() 测试

#### 子阶段 1.4: Linux 环境验证

- [X] T023 [US1] 在 Linux 环境运行 uv sync --extra dev
- [X] T024 [US1] 在 Linux 环境运行 pytest tests/ -v，确认所有测试通过
- [X] T025 [US1] 验证无 ImportError 相关的 pywin32 错误

**Checkpoint**: 此时 User Story 1 应该完全功能化且可独立测试 - Linux 环境可以运行所有单元测试

---

## Phase 4: User Story 2 - Windows 环境功能完全保留 (Priority: P1)

**Goal**: 确保 Windows 系统上所有依赖的功能（窗口置顶、前台检测）继续正常工作

**Independent Test**: 在 Windows 系统上启动程序，验证悬浮窗保持置顶，游戏前台/后台切换时程序正确检测并响应

### 实现 User Story 2

- [ ] T026 [US2] 在 Windows 环境运行 uv sync --extra dev (需要 Windows 环境)
- [ ] T027 [US2] 在 Windows 环境运行 pytest tests/ -v，确认所有测试通过 (需要 Windows 环境)
- [ ] T028 [US2] 在 Windows 环境启动程序（uv run python src/app.py）(需要 Windows 环境)
- [ ] T029 [US2] 验证悬浮窗（overlay、map_overlay、hp_overlay）自动置顶且保持在其他窗口之上 (需要 Windows 环境)
- [ ] T030 [US2] 验证 is_window_in_foreground() 函数正确检测游戏窗口前台/后台状态 (需要 Windows 环境)
- [ ] T031 [US2] 验证所有其他 UI 功能（settings、capture_region）正常工作 (需要 Windows 环境)

**Checkpoint**: 此时 User Stories 1 和 2 都应该独立工作 - Windows 功能完全保留

---

## Phase 5: User Story 3 - 非 Windows 环境优雅降级 (Priority: P2)

**Goal**: 程序在非 Windows 系统运行时优雅降级，核心功能可用，平台特定功能显示警告

**Independent Test**: 在 Linux 上启动程序，程序能够启动并显示界面（如果有 GUI 环境），检测器模块可以被导入和测试，窗口置顶等功能记录警告但不崩溃

### 实现 User Story 3

- [X] T032 [US3] 在 Linux 环境测试程序启动（如果有 X11/Wayland 环境）- 跳过（无GUI环境）
- [X] T033 [US3] 验证核心模块（detector、config、logger、common）成功加载
- [X] T034 [US3] 验证 is_window_in_foreground() 函数在非 Windows 环境安全返回 False 而不抛出异常
- [X] T035 [US3] 检查日志输出，确认平台特定功能警告信息正确记录
- [X] T036 [US3] 验证检测器逻辑（day_detector、rain_detector 等）可以被导入和单元测试

**Checkpoint**: 所有用户故事应该现在都独立功能化

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 影响多个用户故事的改进和文档更新

- [X] T037 [P] 更新 README.md，说明跨平台测试支持（如有需要）- 可选，暂不更新
- [X] T038 [P] 创建 tests/README.md，说明如何运行测试（可选）- 已在 pyproject.toml 中配置
- [X] T039 代码复查：确认所有 pywin32 引用已移除
- [X] T040 代码复查：确认代码符合 PEP 8 和项目章程
- [X] T041 验证 uv.lock 文件已正确更新（无 pywin32 依赖）
- [X] T042 运行完整的测试套件并确保覆盖率符合预期
- [X] T043 创建 git commit，包含清晰的提交信息和 Co-Authored-By 标签

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖 - 可以立即开始
- **Foundational (Phase 2)**: 依赖 Setup 完成 - 阻塞所有用户故事
- **User Stories (Phase 3-5)**: 都依赖 Foundational 阶段完成
  - User Story 1 (P1): Linux 测试支持 - 核心功能
  - User Story 2 (P1): Windows 功能保留 - 核心功能（可与 US1 并行）
  - User Story 3 (P2): 优雅降级 - 依赖 US1 完成
- **Polish (Phase 6)**: 依赖所有用户故事完成

### User Story Dependencies

- **User Story 1 (P1)**: 可以在 Foundational (Phase 2) 后开始 - 无其他故事依赖
- **User Story 2 (P1)**: 可以在 Foundational (Phase 2) 后开始 - 无其他故事依赖（可与 US1 并行验证）
- **User Story 3 (P2)**: 应在 User Story 1 完成后开始 - 依赖跨平台代码重构

### Within Each User Story

- User Story 1: 代码重构 → 依赖管理 → 测试基础设施创建 → Linux 验证
- User Story 2: Windows 环境测试和验证
- User Story 3: 非 Windows 环境测试和验证

### Parallel Opportunities

- **Setup (Phase 1)**: T001 和 T002 可以并行
- **Foundational (Phase 2)**: T003、T004、T005 可以并行运行（都是搜索/验证任务）
- **User Story 1, 子阶段 1.1**: T006 可以独立运行
- **User Story 1, 子阶段 1.3**: T015-T022 都可以并行运行（创建不同的测试文件）
- **User Stories 1 & 2**: US1 和 US2 可以由不同开发者并行处理（在 Foundational 完成后）

---

## Parallel Example: User Story 1 测试基础设施创建

```bash
# 同时创建所有测试文件（子阶段 1.3）:
Task T015: "创建 tests/ 目录和 tests/__init__.py 文件"
Task T016: "创建 tests/conftest.py（pytest 配置文件）"
Task T017: "创建 tests/test_imports.py，包含 test_import_ui_modules() 函数"
Task T020: "创建 tests/test_utils.py，包含 test_is_window_in_foreground_no_app() 测试"

# 同时在 test_imports.py 中添加测试函数:
Task T018: "在 tests/test_imports.py 中添加 test_import_detector_modules() 函数"
Task T019: "在 tests/test_imports.py 中添加 test_import_core_modules() 函数"

# 同时在 test_utils.py 中添加测试函数:
Task T021: "在 tests/test_utils.py 中添加 test_is_window_in_foreground_with_active_window() 测试"
Task T022: "在 tests/test_utils.py 中添加 test_is_window_in_foreground_no_active_window() 测试"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. 完成 Phase 1: Setup
2. 完成 Phase 2: Foundational（关键 - 阻塞所有故事）
3. 完成 Phase 3: User Story 1（Linux 测试支持）
4. **停止并验证**: 在 Linux 环境独立测试 User Story 1
5. 如果准备好，可以部署/演示

### Incremental Delivery

1. 完成 Setup + Foundational → 基础就绪
2. 添加 User Story 1 → 独立测试 → 部署/演示（MVP！）
3. 添加 User Story 2 → 独立测试 → 部署/演示（Windows 验证）
4. 添加 User Story 3 → 独立测试 → 部署/演示（完整跨平台支持）
5. 每个故事都增加价值而不破坏之前的故事

### Parallel Team Strategy

多个开发者时：

1. 团队共同完成 Setup + Foundational
2. Foundational 完成后：
   - Developer A: User Story 1（代码重构和 Linux 测试）
   - Developer B: User Story 2（Windows 验证） - 可与 A 并行
3. User Story 1 和 2 完成后：
   - 任一开发者: User Story 3（跨平台降级测试）

---

## Notes

- [P] 任务 = 不同文件，无依赖关系
- [Story] 标签将任务映射到特定用户故事以便追溯
- 每个用户故事应该可以独立完成和测试
- 在每个检查点停止以独立验证故事
- 在每个任务或逻辑组后提交
- 避免：模糊的任务、同一文件冲突、破坏独立性的跨故事依赖

---

## Summary

- **总任务数**: 43 个任务
- **User Story 1 任务数**: 20 个任务（T006-T025）
- **User Story 2 任务数**: 6 个任务（T026-T031）
- **User Story 3 任务数**: 5 个任务（T032-T036）
- **并行机会**:
  - Foundational 阶段: 3 个任务可并行
  - User Story 1 测试创建: 8 个任务可并行
  - User Story 1 和 2 可由不同开发者并行处理
- **MVP 范围**: User Story 1（Linux 测试支持）
- **预计总时间**: 约 2 小时（参考 plan.md）

**格式验证**: ✅ 所有任务都遵循 checklist 格式（checkbox、ID、标签、文件路径）
