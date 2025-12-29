"""
Single Instance Guard Module

Prevents multiple instances of the application from running simultaneously
using QSharedMemory for instance detection and QLocalSocket/QLocalServer for IPC.
"""

import sys
import os
import time
import getpass
from PyQt6.QtCore import QObject, pyqtSignal, QSharedMemory, QIODevice
from PyQt6.QtNetwork import QLocalSocket, QLocalServer

from src.logger import info, warning, error


class SingleInstanceGuard(QObject):
    """
    Single instance guard using QSharedMemory and QLocalSocket/QLocalServer.

    This class ensures only one instance of the application can run per user.
    If a second instance is detected, it can notify the existing instance via IPC.
    """

    # Signal emitted when another instance tries to start
    instance_started = pyqtSignal(str)

    def __init__(self, base_app_id: str):
        """
        Initialize the single instance guard.

        Args:
            base_app_id: Base application identifier (e.g., "nightreign-overlay-helper")
        """
        super().__init__()

        # Generate user-specific app ID for per-user isolation
        username = getpass.getuser()
        self.app_id = f"{base_app_id}-{username}"

        info(f"Initializing SingleInstanceGuard with ID: {self.app_id}")

        # Shared memory for instance detection
        self.shared_memory = QSharedMemory(self.app_id)

        # IPC server for receiving notifications from new instances
        self.local_server = None

        # Track if this is the primary instance
        self._is_primary = False

    def is_primary_instance(self) -> bool:
        """
        Check if this is the primary (first) instance.

        Returns:
            True if this is the primary instance, False if another instance exists
        """
        # Try to create shared memory
        if self.shared_memory.create(1):
            # Successfully created - we are the primary instance
            self._is_primary = True
            info(f"Primary instance created with ID: {self.app_id}")
            return True

        # Failed to create - check if it's a valid existing instance or stale lock
        info("Shared memory already exists, checking if instance is valid...")

        if self._try_attach_and_detach():
            # Valid existing instance detected
            info("Valid existing instance detected")
            self._notify_existing_instance()
            return False
        else:
            # Stale lock detected - try to recover
            warning("Stale shared memory detected, attempting recovery...")
            if self._recover_from_stale_memory():
                self._is_primary = True
                info("Successfully recovered from stale shared memory")
                return True
            else:
                error("Failed to recover from stale shared memory")
                return False

    def _try_attach_and_detach(self) -> bool:
        """
        Verify if shared memory is valid by attempting to attach and detach.

        Returns:
            True if shared memory is valid (instance is running), False if stale
        """
        if self.shared_memory.attach():
            self.shared_memory.detach()
            return True
        return False

    def _recover_from_stale_memory(self) -> bool:
        """
        Attempt to recover from stale shared memory.

        This handles cases where the previous instance crashed without cleanup.

        Returns:
            True if recovery successful, False otherwise
        """
        # Strategy 1: Short delay and retry
        time.sleep(0.1)
        if self.shared_memory.create(1):
            info("Recovered by retry after delay")
            return True

        # Strategy 2: Linux-specific cleanup of /dev/shm
        if sys.platform.startswith('linux'):
            native_key = self.shared_memory.nativeKey()
            if native_key:
                shm_path = f"/dev/shm/{native_key}"
                if os.path.exists(shm_path):
                    try:
                        info(f"Attempting to remove stale shared memory: {shm_path}")
                        os.remove(shm_path)
                        # Try creating again
                        if self.shared_memory.create(1):
                            info("Successfully recovered by removing stale /dev/shm file")
                            return True
                    except PermissionError as e:
                        error(f"Permission denied removing stale shared memory: {e}")
                    except Exception as e:
                        error(f"Error removing stale shared memory: {e}")

        return False

    def _notify_existing_instance(self):
        """
        Notify the existing instance that a new instance attempted to start.

        Sends an "ACTIVATE" message via QLocalSocket.
        """
        socket = QLocalSocket()
        socket.connectToServer(self.app_id)

        if socket.waitForConnected(1000):
            info(f"Connected to existing instance: {self.app_id}")
            message = "ACTIVATE"
            socket.write(message.encode('utf-8'))
            socket.waitForBytesWritten(1000)
            socket.disconnectFromServer()
            info("Sent ACTIVATE message to existing instance")
        else:
            warning(f"Failed to connect to existing instance: {socket.errorString()}")

        socket.deleteLater()

    def setup_ipc_server(self):
        """
        Setup IPC server to listen for notifications from new instances.

        This should only be called by the primary instance after QApplication is created.
        """
        if not self._is_primary:
            warning("setup_ipc_server() called on non-primary instance")
            return

        self.local_server = QLocalServer()

        # Remove any previous server instances
        QLocalServer.removeServer(self.app_id)

        if self.local_server.listen(self.app_id):
            info(f"Local server started: {self.app_id}")
            self.local_server.newConnection.connect(self._handle_new_connection)
        else:
            error(f"Failed to start local server: {self.local_server.errorString()}")

    def _handle_new_connection(self):
        """
        Handle incoming connection from a new instance.

        Reads the message and emits the instance_started signal.
        """
        socket = self.local_server.nextPendingConnection()
        if socket:
            socket.waitForReadyRead(1000)
            data = socket.readAll().data().decode('utf-8')
            info(f"Received message from another instance: {data}")

            # Emit signal to notify application
            self.instance_started.emit(data)

            socket.disconnectFromServer()
            socket.deleteLater()

    def cleanup(self):
        """
        Clean up resources (shared memory and local server).

        Should be called when the application exits.
        """
        info("Cleaning up SingleInstanceGuard resources...")

        # Close IPC server
        if self.local_server:
            self.local_server.close()
            info("Local server closed")

        # Detach and delete shared memory
        if self.shared_memory.isAttached():
            self.shared_memory.detach()
            info("Shared memory detached")
