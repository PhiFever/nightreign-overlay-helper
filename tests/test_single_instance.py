"""
Unit tests for SingleInstanceGuard

Tests the single instance functionality including lock acquisition,
detection of existing instances, and proper cleanup.
"""

import pytest
import time
from src.single_instance import SingleInstanceGuard


class TestSingleInstanceGuard:
    """Test suite for SingleInstanceGuard class"""

    def test_first_instance_acquires_lock(self):
        """
        Test that the first instance successfully acquires the lock.

        Verifies:
        - is_primary_instance() returns True for first instance
        - Cleanup properly releases the lock
        """
        guard = SingleInstanceGuard("test-app-001")

        try:
            # First instance should successfully acquire lock
            assert guard.is_primary_instance() is True, \
                "First instance should acquire lock"
        finally:
            # Clean up
            guard.cleanup()

    def test_second_instance_fails_to_acquire_lock(self):
        """
        Test that a second instance cannot acquire the lock while first instance holds it.

        Verifies:
        - First instance acquires lock
        - Second instance is_primary_instance() returns False
        - After first instance cleanup, new instance can acquire lock
        """
        guard1 = SingleInstanceGuard("test-app-002")
        guard2 = None

        try:
            # First instance acquires lock
            assert guard1.is_primary_instance() is True, \
                "First instance should acquire lock"

            # Second instance should fail to acquire lock
            guard2 = SingleInstanceGuard("test-app-002")
            assert guard2.is_primary_instance() is False, \
                "Second instance should NOT acquire lock while first exists"

        finally:
            # Clean up in reverse order
            if guard2:
                guard2.cleanup()
            guard1.cleanup()

    def test_cleanup_releases_lock(self):
        """
        Test that cleanup() properly releases the lock, allowing new instances.

        Verifies:
        - First instance acquires and releases lock
        - After cleanup, second instance can acquire lock
        """
        # First instance
        guard1 = SingleInstanceGuard("test-app-003")

        try:
            assert guard1.is_primary_instance() is True, \
                "First instance should acquire lock"
        finally:
            guard1.cleanup()

        # Small delay to ensure cleanup completes
        time.sleep(0.1)

        # Second instance (after first cleanup) should acquire lock
        guard2 = SingleInstanceGuard("test-app-003")

        try:
            assert guard2.is_primary_instance() is True, \
                "Second instance should acquire lock after first cleanup"
        finally:
            guard2.cleanup()

    def test_stale_lock_recovery(self):
        """
        Test recovery from stale shared memory (simulated crash scenario).

        Verifies:
        - If shared memory exists but no process is attached, new instance can recover
        - This simulates crash recovery where cleanup() was never called
        """
        guard1 = SingleInstanceGuard("test-app-004")

        # Acquire lock but DON'T call cleanup() - simulate crash
        assert guard1.is_primary_instance() is True, \
            "First instance should acquire lock"

        # Force detach without proper cleanup (simulate crash)
        if guard1.shared_memory.isAttached():
            guard1.shared_memory.detach()

        # Small delay
        time.sleep(0.1)

        # New instance should be able to detect stale lock and recover
        guard2 = SingleInstanceGuard("test-app-004")

        try:
            # This should succeed via stale lock recovery
            result = guard2.is_primary_instance()
            assert result is True, \
                "Second instance should recover from stale lock"
        finally:
            guard2.cleanup()

    def test_user_specific_app_id(self):
        """
        Test that app IDs are user-specific.

        Verifies:
        - Different base app IDs generate different final app IDs
        - App IDs include username for per-user isolation
        """
        guard1 = SingleInstanceGuard("test-app-A")
        guard2 = SingleInstanceGuard("test-app-B")

        try:
            # Different base IDs should result in different app_ids
            assert guard1.app_id != guard2.app_id, \
                "Different base app IDs should produce different final IDs"

            # Both should include username
            import getpass
            username = getpass.getuser()
            assert username in guard1.app_id, \
                "App ID should include username"
            assert username in guard2.app_id, \
                "App ID should include username"

        finally:
            guard1.cleanup()
            guard2.cleanup()
