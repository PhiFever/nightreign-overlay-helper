# 实现计划：应用程序单例化

**分支**: `001-app-singleton` | **日期**: 2026-02-08 | **规格**: [spec.md](spec.md)
**输入**: 来自 `/specs/001-app-singleton/spec.md` 的功能规格

## 概要

通过 Windows Named Mutex 实现应用程序单例化，确保同一时间只有一个实例运行。当检测到已有实例时，激活已有实例的设置窗口并退出新实例。Mutex 由操作系统管理，进程异常终止时自动释放，无需手动清理。

## 技术上下文

**语言/版本**: Python 3.13
**主要依赖**: PyQt6（GUI）、pywin32（Win32 API：CreateMutex、FindWindow、SetForegroundWindow）
**存储**: 不适用
**测试**: 无测试框架（手动测试）
**目标平台**: Windows 10/11
**项目类型**: 单体桌面应用
**性能目标**: 单例检测对启动时间影响 < 500ms
**约束**: 仅 Windows 平台、需兼容 PyInstaller 打包
**规模/范围**: 修改 1 个文件（`src/app.py`），新增约 30-40 行代码

## Constitution Check

*门控：宪法文件为空白模板，未配置项目特定原则。跳过门控检查。*

## 项目结构

### 文档（本功能）

```text
specs/001-app-singleton/
├── plan.md              # 本文件
├── research.md          # 阶段 0 研究输出
├── quickstart.md        # 快速开始指南
└── tasks.md             # 阶段 2 输出（由 /speckit.tasks 创建）
```

### 源码（仓库根目录）

```text
src/
├── app.py               # 修改：在 main 入口添加单例检测逻辑
└── common.py            # 引用：APP_NAME 常量
```

**结构决策**: 本功能改动极小，仅修改 `src/app.py` 入口文件。在 `QApplication` 创建之前添加 Mutex 检测逻辑，在检测到已有实例时通过 win32gui 激活已有窗口后退出。不需要新建文件或模块。

## 设计详情

### 核心流程

```
程序启动
  ↓
调用 CreateMutex("nightreign-overlay-helper-singleton")
  ↓
检查 GetLastError() == ERROR_ALREADY_EXISTS?
  ├── 否 → 正常启动（持有 Mutex 句柄直到进程退出）
  └── 是 → 查找已有实例窗口 → 激活窗口 → 退出
```

### 关键设计决策

1. **Mutex 检测位置**：在 `QApplication` 创建之前执行，避免重复实例创建 Qt 资源
2. **窗口激活方式**：通过设置窗口标题（SettingsWindow）用 `FindWindow` 查找，用 `SetForegroundWindow` 激活
3. **Mutex 生命周期**：句柄存储在模块级变量中，随进程退出自动释放。不需要显式 `CloseHandle`（`os._exit()` 会关闭所有句柄）
4. **无新依赖**：仅使用已有的 `pywin32` 包中的 `win32event`、`win32api`、`win32gui` 模块

## 复杂度追踪

> 无宪法违规，无需追踪。
