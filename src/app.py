import sys
import time
import os
import win32event
import win32api
import win32gui
import winerror
from PyQt6.QtCore import QThread, Qt, pyqtSignal
from PyQt6.QtGui import QIcon, QAction, QCursor
from PyQt6.QtWidgets import (
    QApplication, QSystemTrayIcon, QMenu
)

from src.ui.input import InputWorker
from src.ui.overlay import OverlayWidget
from src.ui.map_overlay import MapOverlayWidget
from src.ui.hp_overlay import HpOverlayWidget
from src.ui.settings import SettingsWindow
from src.updater import Updater
from src.common import APP_FULLNAME, APP_VERSION, ICON_PATH
from src.logger import info, warning, error

# 模块级变量，防止 Mutex 句柄被垃圾回收释放
_singleton_mutex = None


def log_system_and_screen_info(app: QApplication):
    try:
        import platform
        system = platform.system()
        release = platform.release()
        version = platform.version()
        info(f"Operating System: {system} {release} ({version})")
    except Exception as e:
        warning(f"Error getting OS info: {e}")

    try:
        import mss
        with mss.mss() as sct:
            monitors = sct.monitors
            info(f"MSS Detected {len(monitors)-1} monitor(s):")
            for i, monitor in enumerate(monitors[1:], start=1):
                info(f"    Monitor {i}: {monitor['width']}x{monitor['height']} at ({monitor['left']},{monitor['top']})")
    except Exception as e:
        warning(f"Error getting monitor info: {e}")

    try:
        screens = app.screens()
        info(f"QApplication detected {len(screens)} screen(s):")
        for i, screen in enumerate(screens, start=1):
            size = screen.size()
            pos = screen.geometry().topLeft()
            dpi = screen.logicalDotsPerInch()
            device_pixel_ratio = screen.devicePixelRatio()
            info(f"    Screen {i}: {size.width()}x{size.height()} at ({pos.x()},{pos.y()}), DPI: {dpi}, Device Pixel Ratio: {device_pixel_ratio}")
    except Exception as e:
        warning(f"Error getting screens from QApplication: {e}")


if __name__ == "__main__":
    info("=" * 40)
    info(f"Starting app v{APP_VERSION}...")

    # 单例检测：通过 Windows Named Mutex 确保只运行一个实例
    try:
        _singleton_mutex = win32event.CreateMutex(None, False, "nightreign-overlay-helper-singleton")
        if win32api.GetLastError() == winerror.ERROR_ALREADY_EXISTS:
            info("检测到已有实例运行，尝试激活已有实例窗口...")
            hwnd = win32gui.FindWindow(None, f"{APP_FULLNAME} - 设置")
            if hwnd:
                win32gui.ShowWindow(hwnd, 9)  # SW_RESTORE
                win32gui.SetForegroundWindow(hwnd)
                info("已激活已有实例的设置窗口。")
            else:
                info("已有实例正在运行（设置窗口未打开），本实例将退出。")
            sys.exit(0)
    except Exception as e:
        warning(f"单例检测失败，将继续启动: {e}")

    QApplication.setAttribute(Qt.ApplicationAttribute.AA_Use96Dpi)
    QApplication.setHighDpiScaleFactorRoundingPolicy(Qt.HighDpiScaleFactorRoundingPolicy.PassThrough)

    app = QApplication(sys.argv)

    log_system_and_screen_info(app)
    
    # 防止因没有窗口而导致程序退出
    app.setQuitOnLastWindowClosed(False)

    # 创建对象
    input = InputWorker()
    overlay = OverlayWidget()
    map_overlay = MapOverlayWidget()
    hp_overlay = HpOverlayWidget()

    updater = Updater(input, overlay, map_overlay, hp_overlay)
    settings_window = SettingsWindow(overlay, map_overlay, updater, input)
    
    # 创建系统托盘图标和菜单
    tray_icon = QSystemTrayIcon()
    tray_icon.setIcon(QIcon(ICON_PATH))
    tray_icon.setToolTip(APP_FULLNAME)

    menu = QMenu()
    settings_action = QAction("设置")
    def show_settings():
        settings_window.show()
        settings_window.activateWindow()
        settings_window.raise_()
    settings_action.triggered.connect(show_settings)
    menu.addAction(settings_action)
    quit_action = QAction("退出")
    quit_action.triggered.connect(app.quit)
    menu.addAction(quit_action)
    menu.addSeparator()
    tray_icon.setContextMenu(menu)
    tray_icon.show()
    
    def show_menu_at_cursor_pos():
        cursor_pos = QCursor.pos()
        menu.move(cursor_pos)
        menu.show()
    def on_menu_show():
        overlay.is_menu_opened = True
        map_overlay.is_menu_opened = True
        updater.is_menu_opened = True
        # info("Menu opened")
    def on_menu_hide():
        overlay.is_menu_opened = False
        map_overlay.is_menu_opened = False
        updater.is_menu_opened = False
        # info("Menu closed")

    overlay.right_click_signal.connect(show_menu_at_cursor_pos)
    overlay.right_click_signal.connect(on_menu_show)
    menu.aboutToShow.connect(on_menu_show)
    menu.aboutToHide.connect(on_menu_hide)

    # 启动输入监听
    input_thread = QThread()
    input.moveToThread(input_thread)
    input_thread.started.connect(input.run)
    input_thread.start()
    
    # 设置并启动后台检测器
    updater_thread = QThread()
    updater.moveToThread(updater_thread)
    updater_thread.started.connect(updater.run)
    updater_thread.start()

    # 清理：程序退出时，停止worker并等待线程结束
    def on_quit():
        info("Stopping worker thread...")
        updater.stop()
        updater_thread.quit()
        if not updater_thread.wait(1000):
            print("Updater thread did not exit in time. Forcing termination.")
            updater_thread.terminate()
        else:
            info("Updater thread stopped.")
        input.stop()
        input_thread.quit()
        if not input_thread.wait(1000):
            print("Input thread did not exit in time. Forcing termination.")
            input_thread.terminate()
        else:
            info("Input thread stopped.")
        info("All Thread stopped.")

        tray_icon.deleteLater()

    app.aboutToQuit.connect(on_quit)
    
    overlay.show()

    try:
        exit_code = app.exec() 
        info(f"QApp event loop finished with exit code {exit_code}.")
    except Exception as e:
        exit_code = 1
        error(f"Exception in app exec: {e}")

    settings_window.save_settings()

    time.sleep(1)
    os._exit(exit_code)
