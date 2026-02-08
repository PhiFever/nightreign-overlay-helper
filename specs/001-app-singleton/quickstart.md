# 快速开始：应用程序单例化

**分支**: `001-app-singleton` | **日期**: 2026-02-08

## 前置条件

- Python 3.13
- 已安装项目依赖（`uv sync`）
- Windows 操作系统

## 实现概览

在 `src/app.py` 的 `if __name__ == "__main__":` 入口处，于 `QApplication` 创建之前添加单例检测：

```python
import win32event
import win32api
import win32gui
import winerror

# 尝试创建命名 Mutex
mutex = win32event.CreateMutex(None, False, "nightreign-overlay-helper-singleton")
if win32api.GetLastError() == winerror.ERROR_ALREADY_EXISTS:
    # 已有实例运行，尝试激活已有实例窗口
    # ... 查找并激活窗口 ...
    sys.exit(0)

# 正常启动流程继续...
```

## 修改文件

| 文件 | 修改内容 |
|------|----------|
| `src/app.py` | 在入口函数开头添加 Mutex 创建和检测逻辑 |

## 验证步骤

1. **正常启动**：运行 `uv run python src/app.py`，确认程序正常启动
2. **重复启动拦截**：保持第一个实例运行，再次运行 `uv run python src/app.py`，确认第二个实例被拦截
3. **已有实例激活**：重复启动时，确认已有实例的设置窗口被激活
4. **崩溃恢复**：通过任务管理器终止进程，确认可以立即重新启动
5. **PyInstaller 构建**：运行 `.\build.bat`，验证打包后的可执行文件同样具备单例行为
