import sys
import time
import os
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

    # Single instance check - MUST happen before QApplication creation
    from src.single_instance import SingleInstanceGuard

    instance_guard = SingleInstanceGuard("nightreign-overlay-helper")

    if not instance_guard.is_primary_instance():
        # Another instance is already running
        info("Another instance is already running. Exiting...")
        sys.exit(0)

    QApplication.setAttribute(Qt.ApplicationAttribute.AA_Use96Dpi)
    QApplication.setHighDpiScaleFactorRoundingPolicy(Qt.HighDpiScaleFactorRoundingPolicy.PassThrough)

    app = QApplication(sys.argv)

    log_system_and_screen_info(app)

    # 防止因没有窗口而导致程序退出
    app.setQuitOnLastWindowClosed(False)

    # Setup IPC server to receive activation requests from new instances
    instance_guard.setup_ipc_server()

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

    # Window activation callback for when new instance tries to start
    def activate_window(message):
        """Activate window when another instance tries to start"""
        info(f"Activating window due to new instance detection: {message}")

        # Check if settings window is visible
        if settings_window and settings_window.isVisible():
            # Restore from minimized state if needed
            if settings_window.isMinimized():
                settings_window.showNormal()

            # Bring window to front
            settings_window.show()
            settings_window.raise_()
            settings_window.activateWindow()
        else:
            # No window visible - show tray notification
            tray_icon.showMessage(
                APP_FULLNAME,
                "程序已在运行",
                QSystemTrayIcon.MessageIcon.Information,
                2000
            )

    # Connect instance_started signal to activation callback
    instance_guard.instance_started.connect(activate_window)

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

        # Cleanup single instance guard
        instance_guard.cleanup()

    app.aboutToQuit.connect(on_quit)
    
    overlay.show()

    try:
        exit_code = app.exec()
        info(f"QApp event loop finished with exit code {exit_code}.")
    except Exception as e:
        exit_code = 1
        error(f"Exception in app exec: {e}")
    finally:
        # Ensure cleanup happens even if exception occurs
        # Note: on_quit handler already calls instance_guard.cleanup()
        # This is a safety fallback in case on_quit doesn't run
        try:
            instance_guard.cleanup()
        except Exception as cleanup_error:
            error(f"Error during cleanup: {cleanup_error}")

    settings_window.save_settings()

    time.sleep(1)
    os._exit(exit_code)
