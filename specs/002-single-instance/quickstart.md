# Quickstart: 单实例运行功能测试指南

**Feature**: 002-single-instance
**Created**: 2025-12-26
**Purpose**: 快速验证单实例功能是否正常工作

## 前置条件

- 已完成代码实现（src/single_instance.py 和 app.py 集成）
- Python 3.13 环境
- 已安装项目依赖（uv sync）

## 测试场景

### 测试 1: 基本单实例功能

**目标**: 验证第二个实例无法启动

**步骤**:
1. 启动程序：
   ```bash
   uv run python src/app.py
   ```

2. 等待 2-3 秒，确保程序完全启动

3. 再次双击程序图标或运行：
   ```bash
   uv run python src/app.py
   ```

**预期结果**:
- ✅ 第二个实例立即退出（无新窗口创建）
- ✅ 托盘区域只有 1 个程序图标
- ✅ 第一个实例的窗口获得焦点（如果窗口可见）
- ✅ 或托盘图标显示气泡通知"程序已在运行"

**日志验证**:
```
[INFO] Primary instance created with ID: NightReignOverlayHelper-...
[INFO] Local server started: NightReignOverlayHelper-...
[INFO] Received message from another instance: ACTIVATE
[INFO] Activating window due to new instance detection
```

---

### 测试 2: 快速多次点击

**目标**: 验证短时间内多次启动只创建一个实例

**步骤**:
1. 确保程序未运行

2. 快速双击程序图标 5-10 次（在 1 秒内）

**预期结果**:
- ✅ 只有 1 个进程启动
- ✅ 托盘区域只有 1 个图标
- ✅ 其他启动请求被忽略

**验证方法**:
```bash
# Linux/Mac
ps aux | grep "python.*app.py" | grep -v grep | wc -l
# 应该只返回 1

# Windows (PowerShell)
(Get-Process python | Where-Object {$_.CommandLine -like "*app.py*"}).Count
# 应该只返回 1
```

---

### 测试 3: 窗口最小化后的激活

**目标**: 验证窗口最小化后再次启动能够恢复显示

**步骤**:
1. 启动程序

2. 打开设置窗口

3. 最小化设置窗口

4. 再次双击程序图标

**预期结果**:
- ✅ 设置窗口从最小化状态恢复
- ✅ 设置窗口置顶显示
- ✅ 设置窗口获得焦点

---

### 测试 4: 仅托盘图标时的通知

**目标**: 验证没有窗口显示时的视觉反馈

**步骤**:
1. 启动程序

2. 关闭所有窗口（仅保留托盘图标）

3. 再次双击程序图标

**预期结果**:
- ✅ 托盘图标显示气泡通知
- ✅ 通知内容为"程序已在运行"
- ✅ 通知持续约 2 秒

---

### 测试 5: 异常退出后的恢复

**目标**: 验证程序崩溃后可以正常重启

**步骤**:
1. 启动程序

2. 强制终止进程：
   ```bash
   # Linux/Mac
   kill -9 <PID>

   # Windows (任务管理器)
   右键进程 → 结束任务
   ```

3. 立即再次启动程序：
   ```bash
   uv run python src/app.py
   ```

**预期结果**:
- ✅ 程序能够正常启动
- ✅ 不会因为残留锁而被阻止
- ✅ 日志可能显示清理残留锁的信息

**日志验证**:
```
[WARNING] Found stale shared memory, attempting cleanup...
[INFO] Successfully recovered from stale shared memory
[INFO] Primary instance created with ID: ...
```

---

### 测试 6: 多用户隔离（Linux）

**目标**: 验证不同用户可以同时运行程序

**步骤**:
1. 用户 A 启动程序

2. 切换到用户 B（或使用 su）

3. 用户 B 启动程序

**预期结果**:
- ✅ 两个用户都能成功启动程序
- ✅ 互不干扰

**注意**: Windows 测试需要多个用户账户

---

### 测试 7: 进程间通信性能

**目标**: 验证 IPC 延迟符合要求（<50ms）

**步骤**:
1. 启动程序并记录时间戳 T1

2. 再次启动程序并记录时间戳 T2

3. 观察窗口激活时间戳 T3

**预期结果**:
- ✅ T3 - T2 < 1 秒（用户可感知的响应时间）
- ✅ 日志中显示的 IPC 通信时间 < 50ms

---

## 单元测试

运行自动化测试：

```bash
# 运行单实例测试
uv run pytest tests/test_single_instance.py -v

# 运行所有测试
uv run pytest tests/ -v
```

**预期输出**:
```
tests/test_single_instance.py::test_first_instance_is_primary PASSED
tests/test_single_instance.py::test_second_instance_detection PASSED
tests/test_single_instance.py::test_cleanup_releases_lock PASSED
========================= 3 passed in 1.23s =========================
```

---

## 常见问题排查

### 问题 1: 第二个实例没有退出

**可能原因**:
- 共享内存创建失败
- IPC 服务器未启动

**排查步骤**:
1. 检查日志输出
2. 确认日志中有 "Primary instance created" 或 "Local server started"
3. 如果都没有，说明单实例检测未正确集成

### 问题 2: 窗口没有激活

**可能原因**:
- IPC 通信失败
- 激活回调未正确连接

**排查步骤**:
1. 检查日志中是否有 "Received message from another instance"
2. 检查是否有 "Activating window due to new instance detection"
3. 如果有前者无后者，检查信号连接代码

### 问题 3: Linux 上共享内存残留

**现象**: 程序异常退出后无法重启

**解决方法**:
```bash
# 查看共享内存段
ls -la /dev/shm/ | grep nightreign

# 手动清理（如果自动清理失败）
rm /dev/shm/qipc_sharedmemory_nightreignoverlayhelper-*
```

### 问题 4: Windows 上命名管道问题

**现象**: IPC 通信失败

**排查步骤**:
```powershell
# 查看命名管道
Get-ChildItem \\.\pipe\ | Where-Object {$_.Name -like "*nightreign*"}
```

---

## 性能基准测试

### 启动时间影响

**测试方法**:
```bash
# 测量启动时间（首次）
time uv run python src/app.py &
PID=$!
sleep 2
kill $PID

# 测量启动时间（第二实例）
uv run python src/app.py &  # 首次启动
sleep 2
time uv run python src/app.py  # 第二实例（应该立即退出）
```

**预期结果**:
- 首次启动时间增加 < 10ms（几乎无影响）
- 第二实例启动时间 < 100ms（快速检测并退出）

---

## 验收标准检查表

基于 spec.md 的成功标准：

- [ ] **SC-001**: 5 秒内多次点击只启动一个进程 ✓
- [ ] **SC-002**: 托盘区域只显示一个图标 ✓
- [ ] **SC-003**: 窗口在 1 秒内获得焦点 ✓
- [ ] **SC-004**: 异常退出后 5 秒内可重启 ✓
- [ ] **SC-005**: Windows 和 Linux 都能正常工作 ✓
- [ ] **SC-006**: 多实例问题工单减少 100% ✓

---

## 下一步

测试通过后：
1. 在 Windows 环境验证（如果当前在 Linux）
2. 创建 Pull Request
3. 更新 CHANGELOG.md
4. 准备发布说明

**文档**: 本快速测试指南应该涵盖所有关键场景，确保单实例功能健壮可靠。
