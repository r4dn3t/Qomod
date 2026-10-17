from datetime import timedelta
import pytest
from app.security import (
    create_access_token,
    create_refresh_token,
    decode_token,
    hash_password,
    verify_password,
)


def test_password_hashing_and_verification():
    plain = "MySecretPassword123!"
    hashed = hash_password(plain)
    assert hashed != plain
    assert verify_password(plain, hashed) is True
    assert verify_password("wrong-password", hashed) is False


def test_create_and_decode_access_token():
    user_id = "user-1234"
    phone = "+12025550199"
    token, jti = create_access_token(user_id=user_id, phone=phone)

    claims = decode_token(token)
    assert claims["sub"] == user_id
    assert claims["phone"] == phone
    assert claims["type"] == "access"
    assert claims["jti"] == jti
    assert "exp" in claims


def test_create_and_decode_refresh_token():
    user_id = "user-1234"
    phone = "+12025550199"
    token, jti = create_refresh_token(user_id=user_id, phone=phone)

    claims = decode_token(token)
    assert claims["sub"] == user_id
    assert claims["type"] == "refresh"
    assert claims["jti"] == jti
