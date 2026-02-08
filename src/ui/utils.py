from PyQt6.QtWidgets import QWidget, QApplication
from PyQt6.QtCore import Qt
from src.logger import info, warning, error


def set_widget_always_on_top(widget: QWidget):
    try:
        import win32gui
        import win32con
        hwnd = widget.winId().__int__()
        win32gui.SetWindowPos(hwnd, win32con.HWND_TOPMOST,
                                0, 0, 0, 0,
                                win32con.SWP_NOSIZE | win32con.SWP_NOMOVE)
        info(f"Window HWND: {hwnd} set to TOPMOST.")
    except Exception as e:
        warning(f"Error setting system always on top: {e}")


def ensure_visible_on_top(widget: QWidget):
    """确保窗口在 OS 层面可见并置顶。
    每次 timerEvent 中调用，当窗口已可见时为近似空操作。
    当 Win+D 隐藏 Tool 窗口后，ShowWindow + SetWindowPos 会将其恢复。
    """
    try:
        import win32gui
        import win32con
        hwnd = widget.winId().__int__()
        win32gui.ShowWindow(hwnd, win32con.SW_SHOWNOACTIVATE)
        win32gui.SetWindowPos(hwnd, win32con.HWND_TOPMOST,
                                0, 0, 0, 0,
                                win32con.SWP_NOSIZE | win32con.SWP_NOMOVE | win32con.SWP_NOACTIVATE)
    except Exception:
        pass


def is_window_in_foreground(window_title: str) -> bool:
    """
    检查包含特定标题的窗口是否在 Windows 的最前面。
    """
    try:
        import win32gui
        import time
        active_window_handle = win32gui.GetForegroundWindow()
        active_window_title = win32gui.GetWindowText(active_window_handle)
        if window_title.lower() in active_window_title.lower():
            return True
        return False
    except Exception as e:
        return False


def get_qt_screen_by_mss_region(region: tuple[int]) -> QWidget:
    """
    根据mss的region获取对应的QScreen对象。
    """
    x, y, w, h = region
    app: QApplication = QApplication.instance()
    screens = app.screens()
    for screen in screens:
        sx = screen.geometry().x()
        sy = screen.geometry().y()
        sw = screen.geometry().width()
        sh = screen.geometry().height()
        ratio = screen.devicePixelRatio()
        mss_sw = int(sw * ratio)
        mss_sh = int(sh * ratio)
        if sx <= x <= sx + mss_sw and sy <= y <= sy + mss_sh:
            return screen
    raise ValueError(f"Region {region} is out of all screen bounds")


def mss_region_to_qt_region(region: tuple[int]):
    screen = get_qt_screen_by_mss_region(region)
    x, y, w, h = region
    sx = screen.geometry().x()
    sy = screen.geometry().y()
    ratio = screen.devicePixelRatio()
    qx = sx + int((x - sx) / ratio)
    qy = sy + int((y - sy) / ratio)
    qw = int(w / ratio)
    qh = int(h / ratio)
    return (qx, qy, qw, qh)

    

def process_region_to_adapt_scale(region: tuple[int], scale: float) -> tuple[int]:
    """
    处理一个region的大小，使其能够适配指定的缩放比例。
    即 w/scale, h/scale为整数。
    """
    x, y, w, h = region
    new_w = int(int(w / scale) * scale)
    new_h = int(int(h / scale) * scale)
    return [x, y, new_w, new_h]