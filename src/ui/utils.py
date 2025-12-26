from PyQt6.QtWidgets import QWidget, QApplication
from PyQt6.QtCore import Qt
from src.logger import info, warning, error


def is_window_in_foreground(window_title: str) -> bool:
    """
    检查应用是否有激活的窗口（跨平台）。

    Args:
        window_title: 窗口标题（保留参数以兼容现有代码）

    Returns:
        bool: 应用是否有激活窗口
    """
    try:
        app = QApplication.instance()
        return app is not None and app.activeWindow() is not None
    except Exception as e:
        warning(f"Error checking window state: {e}")
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