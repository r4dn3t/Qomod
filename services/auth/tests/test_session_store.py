import time
import pytest
from app.session_store import SessionStore


@pytest.mark.asyncio
async def test_session_store_lifecycle():
    store = SessionStore()
    user_id = "user-abc"
    jti = "jti-123"

    # Register
    await store.register_refresh_token(jti, user_id)
    assert await store.is_refresh_token_valid(jti) is True

    # Revoke single token
    await store.revoke_token(jti)
    assert await store.is_refresh_token_valid(jti) is False


@pytest.mark.asyncio
async def test_user_session_revocation():
    store = SessionStore()
    user_id = "user-xyz"
    jti1 = "jti-1"
    jti2 = "jti-2"

    await store.register_refresh_token(jti1, user_id)
    await store.register_refresh_token(jti2, user_id)

    issued_before = time.time() - 10
    assert await store.is_token_revoked("acc-jti", user_id, issued_before) is False

    # Invalidate all user sessions
    await store.revoke_all_user_sessions(user_id)

    # Tokens issued before the revocation must now be invalid
    assert await store.is_token_revoked("acc-jti", user_id, issued_before) is True
    assert await store.is_refresh_token_valid(jti1) is False
    assert await store.is_refresh_token_valid(jti2) is False
