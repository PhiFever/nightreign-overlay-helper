# 研究报告：应用程序单例化

**分支**: `001-app-singleton` | **日期**: 2026-02-08

## 技术方案评估

### 方案对比

| 方案 | 崩溃自动释放 | 竞态安全 | PyInstaller 兼容 | 激活已有实例 | 代码复杂度 |
|------|:---:|:---:|:---:|:---:|:---:|
| Windows Named Mutex (win32event) | ✅ | ✅ | ✅ | ⚠️ 需额外机制 | 低 |
| QLocalServer + QSharedMemory | ✅ (Windows) | ✅ | ✅ | ✅ 内建 IPC | 中 |
| Lock File | ❌ 需手动清理 | ⚠️ TOCTOU 漏洞 | ✅ | ❌ | 低 |
| Named Pipe | ✅ | ✅ | ✅ | ✅ | 高 |

### 决策：Windows Named Mutex + win32gui 窗口激活

**理由**：
1. **崩溃自动恢复**：Windows 内核在进程终止（包括异常终止）时自动关闭所有句柄，Mutex 随之销毁。完美满足 FR-005
2. **原子操作**：`CreateMutex` 是操作系统级原子操作，天然避免竞态条件（FR-006）
3. **已有依赖**：项目已依赖 `pywin32`，无需引入新依赖
4. **代码量少**：约 30-40 行代码即可完成全部功能
5. **窗口激活**：通过 `win32gui.FindWindow` + `SetForegroundWindow` 实现，项目已有 `GAME_WINDOW_TITLE` 的窗口查找模式

**未选方案**：
- **QLocalServer**：功能更强但代码量是 Mutex 的两倍，对于本项目（仅需单例检测+激活窗口）过度设计
- **Lock File**：崩溃后残留锁文件，违反 FR-005
- **Named Pipe**：代码复杂度最高，无额外收益

### 实现要点

1. **Mutex 命名**：使用应用名称作为全局唯一标识 `nightreign-overlay-helper-singleton`
2. **检测流程**：在 `QApplication` 创建之前调用 `CreateMutex`，检查 `ERROR_ALREADY_EXISTS`
3. **激活已有实例**：通过设置窗口标题查找已有实例的设置窗口，发送激活信号
4. **退出处理**：Mutex 句柄随进程退出自动释放，无需显式清理代码（但可选择性添加 `CloseHandle` 保持代码整洁）
