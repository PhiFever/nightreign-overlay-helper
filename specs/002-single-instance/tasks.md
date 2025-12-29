# Tasks: 单实例运行功能

**Input**: Design documents from `/specs/002-single-instance/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md

**Tests**: Unit tests are included per the plan.md Phase 2 requirements. Integration tests from quickstart.md will be performed manually.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: `src/`, `tests/` at repository root
- Python 3.13 with PyQt6 framework
- Desktop application structure

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Verify environment and dependencies for single-instance implementation

- [x] T001 Verify Python 3.13 environment with uv run python --version
- [x] T002 Verify PyQt6 dependencies are available (QSharedMemory, QLocalSocket, QLocalServer)
- [x] T003 [P] Verify platformdirs is available or add as dependency in pyproject.toml

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core single-instance infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Create src/single_instance.py with SingleInstanceGuard class skeleton
- [x] T005 Implement user-specific app ID generation using getpass.getuser()
- [x] T006 Implement QSharedMemory-based single instance detection
- [x] T007 Implement stale lock detection with attach/detach verification
- [x] T008 Implement stale lock recovery mechanism for Linux /dev/shm cleanup
- [x] T009 Implement QLocalServer for IPC server in primary instance
- [x] T010 Implement QLocalSocket for IPC client in secondary instances
- [x] T011 Add instance_started pyqtSignal to SingleInstanceGuard class
- [x] T012 Add logging for all single-instance events (create, detect, cleanup)
- [x] T013 Add cleanup() method for proper resource release

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - 防止重复启动程序实例 (Priority: P1) 🎯 MVP

**Goal**: Prevent multiple instances from running. When user tries to start a second instance, it should be blocked and the existing instance should receive focus.

**Independent Test**: 启动程序 → 再次双击程序图标 → 验证只有一个程序实例在运行，只有一个托盘图标显示，现有窗口获得焦点。

### Tests for User Story 1 (Unit Tests)

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T014 [P] [US1] Create tests/test_single_instance.py with test framework setup
- [x] T015 [P] [US1] Write test_first_instance_acquires_lock in tests/test_single_instance.py
- [x] T016 [P] [US1] Write test_second_instance_fails_to_acquire_lock in tests/test_single_instance.py
- [x] T017 [P] [US1] Write test_cleanup_releases_lock in tests/test_single_instance.py

### Implementation for User Story 1

- [x] T018 [US1] Integrate SingleInstanceGuard into src/app.py before QApplication creation
- [x] T019 [US1] Add single instance check in src/app.py main() that exits if secondary instance detected
- [x] T020 [US1] Add notify_existing_instance() call before sys.exit(0) in src/app.py
- [x] T021 [US1] Verify unit tests pass with uv run pytest tests/test_single_instance.py -v
- [x] T022 [US1] Run manual test from quickstart.md Test 1 (basic single-instance)
- [x] T023 [US1] Run manual test from quickstart.md Test 2 (fast multiple clicks)

**Checkpoint**: At this point, User Story 1 should be fully functional - only one instance can run, second instance exits cleanly

---

## Phase 4: User Story 2 - 用户友好的重复启动提示 (Priority: P2)

**Goal**: Provide clear visual feedback when user tries to start a second instance (window activation, tray notification)

**Independent Test**: 启动程序 → 再次点击程序图标 → 验证是否有明显的视觉反馈（窗口置顶、闪烁或通知）告知用户程序已在运行。

### Implementation for User Story 2

- [x] T024 [US2] Setup IPC server after QApplication creation in src/app.py
- [x] T025 [US2] Create activate_window() callback function in src/app.py
- [x] T026 [US2] Implement window visibility check in activate_window()
- [x] T027 [US2] Implement window restoration from minimized state with showNormal()
- [x] T028 [US2] Implement window raise and focus with activateWindow() and raise_()
- [x] T029 [US2] Implement tray notification fallback when no window is visible
- [x] T030 [US2] Connect instance_started signal to activate_window callback in src/app.py
- [x] T031 [US2] Run manual test from quickstart.md Test 3 (window minimized restoration)
- [x] T032 [US2] Run manual test from quickstart.md Test 4 (tray-only notification)

**Checkpoint**: At this point, User Stories 1 AND 2 should both work - second instance triggers visual feedback

---

## Phase 5: User Story 3 - 异常情况下的单实例恢复 (Priority: P3)

**Goal**: Ensure single-instance mechanism can recover from crashes. After abnormal termination, user should be able to restart the program.

**Independent Test**: 启动程序 → 强制终止进程（kill -9 或任务管理器结束进程）→ 再次启动程序 → 验证程序可以正常启动。

### Implementation for User Story 3

- [x] T033 [US3] Add cleanup() call in app.aboutToQuit handler in src/app.py
- [x] T034 [US3] Add try-finally block around app.exec() to ensure cleanup on exceptions
- [x] T035 [US3] Verify cleanup() releases QSharedMemory in src/single_instance.py
- [x] T036 [US3] Test stale lock recovery on Linux by checking /dev/shm cleanup
- [x] T037 [US3] Run manual test from quickstart.md Test 5 (crash recovery with kill -9)
- [x] T038 [US3] Verify normal exit cleanup works correctly

**Checkpoint**: All user stories should now be independently functional - crash recovery works

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [x] T039 [P] Add comprehensive logging for debugging in src/single_instance.py
- [x] T040 [P] Verify per-user isolation with quickstart.md Test 6 (multi-user)
- [x] T041 [P] Run performance benchmark from quickstart.md Test 7 (IPC <50ms)
- [x] T042 Run full test suite with uv run pytest tests/ -v
- [x] T043 Verify all acceptance criteria from spec.md are met
- [x] T044 Update README.md with single-instance behavior documentation
- [x] T045 Run all quickstart.md validation tests on Linux
- [x] T046 Prepare for Windows testing (document process for future validation)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed sequentially in priority order (P1 → P2 → P3)
  - Each story builds on the previous (US2 adds to US1, US3 adds cleanup to US1+US2)
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
  - Creates core single-instance blocking mechanism
  - Tests verify only one instance can run

- **User Story 2 (P2)**: Depends on User Story 1 completion
  - Adds visual feedback to existing blocking mechanism
  - Requires IPC communication setup from US1
  - Tests verify window activation and tray notifications work

- **User Story 3 (P3)**: Depends on User Stories 1 and 2
  - Adds robust cleanup and recovery
  - Tests verify crash recovery and stale lock cleanup

### Within Each User Story

- Tests (unit tests) MUST be written and FAIL before implementation
- Core implementation before integration into src/app.py
- Unit tests before manual tests
- Story complete before moving to next priority

### Parallel Opportunities

- **Phase 1 (Setup)**: All tasks marked [P] can run in parallel (T002, T003)
- **Phase 2 (Foundational)**: Tasks are sequential due to dependencies in SingleInstanceGuard implementation
- **Phase 3 (US1 Tests)**: All test writing tasks T014-T017 marked [P] can run in parallel
- **Phase 6 (Polish)**: Tasks T039-T041 marked [P] can run in parallel

---

## Parallel Example: User Story 1 Tests

```bash
# Launch all unit tests creation for User Story 1 together:
Task: "Create tests/test_single_instance.py with test framework setup"
Task: "Write test_first_instance_acquires_lock in tests/test_single_instance.py"
Task: "Write test_second_instance_fails_to_acquire_lock in tests/test_single_instance.py"
Task: "Write test_cleanup_releases_lock in tests/test_single_instance.py"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - implements SingleInstanceGuard)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
   - Run unit tests: `uv run pytest tests/test_single_instance.py -v`
   - Run manual tests: quickstart.md Test 1 and Test 2
   - Verify only one instance can run, second instance exits cleanly
5. Deploy/demo if ready (core functionality complete)

### Incremental Delivery

1. Complete Setup + Foundational → SingleInstanceGuard class ready
2. Add User Story 1 → Test independently → Core blocking works (MVP!)
3. Add User Story 2 → Test independently → Visual feedback works
4. Add User Story 3 → Test independently → Crash recovery works
5. Add Polish → Final validation → Ready for production

Each story adds value without breaking previous stories.

### Testing Strategy

- **Unit tests** (pytest): Automated verification of SingleInstanceGuard behavior
  - Tests run isolated without full application
  - Focus on lock acquisition, detection, cleanup

- **Manual tests** (quickstart.md): Real-world scenarios
  - Test 1-2: Verify US1 (blocking)
  - Test 3-4: Verify US2 (visual feedback)
  - Test 5-6: Verify US3 (crash recovery)
  - Test 7: Performance validation

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story builds incrementally (US2 adds to US1, US3 adds to US1+US2)
- Verify unit tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Use `uv run` for all Python execution (per project convention)
- Single-instance detection must happen BEFORE QApplication creation
- Performance target: <10ms startup overhead, <50ms IPC latency

---

## Task Count Summary

- **Total Tasks**: 46
- **Phase 1 (Setup)**: 3 tasks
- **Phase 2 (Foundational)**: 10 tasks (CRITICAL - blocks all user stories)
- **Phase 3 (US1 - P1)**: 10 tasks (4 tests + 6 implementation)
- **Phase 4 (US2 - P2)**: 9 tasks
- **Phase 5 (US3 - P3)**: 6 tasks
- **Phase 6 (Polish)**: 8 tasks

**Parallel Opportunities Identified**: 13 tasks marked [P] can run in parallel within their phases

**Independent Test Criteria**:
- **US1**: Only one process runs, second instance exits cleanly
- **US2**: Window activates or tray notification appears on second launch
- **US3**: Program can restart after crash (kill -9)

**Suggested MVP Scope**: Phase 1 + Phase 2 + Phase 3 (User Story 1 only) = 23 tasks
