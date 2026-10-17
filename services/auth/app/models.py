from typing import Any, Optional
from pydantic import BaseModel, Field


class LoginRequest(BaseModel):
    cellphone_number: str = Field(..., description="E.164 formatted cellphone number")
    password: str = Field(..., description="Plaintext password to authenticate")


class TokenResponse(BaseModel):
    access_token: str
    refresh_token: str
    token_type: str = "bearer"
    expires_in: int
    user_id: str


class RefreshTokenRequest(BaseModel):
    refresh_token: str


class VerifyTokenRequest(BaseModel):
    token: str


class VerifyTokenResponse(BaseModel):
    valid: bool
    user_id: Optional[str] = None
    claims: Optional[dict[str, Any]] = None
    error: Optional[str] = None


class LogoutRequest(BaseModel):
    refresh_token: Optional[str] = None


class UserProfile(BaseModel):
    id: str
    cellphone_number: str
    first_name: Optional[str] = None
    last_name: Optional[str] = None
    email: Optional[str] = None
    status: str
