import asyncio
import time
from typing import Dict, Set


class SessionStore:
    """Thread-safe in-memory session manager supporting real-time token revocation."""

    def __init__(self) -> None:
        self._lock = asyncio.Lock()
        # jti -> user_id
        self._active_refresh_tokens: Dict[str, str] = {}
        # user_id -> set of active jtis
        self._user_refresh_tokens: Dict[str, Set[str]] = {}
        # jti -> expiry timestamp
        self._blacklisted_jtis: Dict[str, float] = {}
        # user_id -> revoked_all_before_timestamp
        self._user_revocations: Dict[str, float] = {}

    async def register_refresh_token(self, jti: str, user_id: str) -> None:
        async with self._lock:
            self._active_refresh_tokens[jti] = user_id
            if user_id not in self._user_refresh_tokens:
                self._user_refresh_tokens[user_id] = set()
            self._user_refresh_tokens[user_id].add(jti)

    async def is_refresh_token_valid(self, jti: str) -> bool:
        async with self._lock:
            if jti in self._blacklisted_jtis:
                return False
            user_id = self._active_refresh_tokens.get(jti)
            if not user_id:
                return False
            return True

    async def is_token_revoked(self, jti: str, user_id: str, issued_at: float) -> bool:
        """Checks if an access or refresh token has been explicitly blacklisted or invalidated."""
        async with self._lock:
            if jti in self._blacklisted_jtis:
                return True
            revoked_before = self._user_revocations.get(user_id)
            if revoked_before is not None and issued_at < revoked_before:
                return True
            return False

    async def revoke_token(self, jti: str) -> None:
        async with self._lock:
            self._blacklisted_jtis[jti] = time.time() + 86400 * 7
            user_id = self._active_refresh_tokens.pop(jti, None)
            if user_id and user_id in self._user_refresh_tokens:
                self._user_refresh_tokens[user_id].discard(jti)

    async def revoke_all_user_sessions(self, user_id: str) -> None:
        """Immediately invalidates all existing access and refresh tokens for a user."""
        async with self._lock:
            now = time.time()
            self._user_revocations[user_id] = now
            jtis = self._user_refresh_tokens.pop(user_id, set())
            for jti in jtis:
                self._blacklisted_jtis[jti] = now + 86400 * 7
                self._active_refresh_tokens.pop(jti, None)


session_store = SessionStore()
