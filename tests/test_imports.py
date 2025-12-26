"""测试所有模块可以导入（跨平台）"""
import pytest


def test_import_ui_modules():
    """测试 UI 模块可以导入"""
    from src.ui import utils
    from src.ui import overlay
    from src.ui import map_overlay
    from src.ui import hp_overlay
    # Settings module requires X server (for pynput), skip in headless environment
    try:
        from src.ui import settings
    except ImportError as e:
        if "this platform is not supported" in str(e) or "X connection" in str(e):
            pytest.skip("Settings module requires X server (pynput dependency)")
        else:
            raise


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
