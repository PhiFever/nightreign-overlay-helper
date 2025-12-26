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
