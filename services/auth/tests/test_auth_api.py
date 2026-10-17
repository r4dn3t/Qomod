import pytest
from httpx import ASGITransport, AsyncClient
from app.main import app


@pytest.mark.asyncio
async def test_healthz():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.get("/healthz")
        assert resp.status_code == 200
        assert resp.json()["status"] == "ok"


@pytest.mark.asyncio
async def test_login_and_verify_flow():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        # 1. Login with demo credentials
        login_resp = await client.post(
            "/api/v1/auth/login",
            json={
                "cellphone_number": "+12025550199",
                "password": "StrongPassword123!"
            }
        )
        assert login_resp.status_code == 200
        tokens = login_resp.json()
        assert "access_token" in tokens
        assert "refresh_token" in tokens

        access_token = tokens["access_token"]
        refresh_token = tokens["refresh_token"]

        # 2. Verify Access Token
        verify_resp = await client.post(
            "/api/v1/auth/verify",
            json={"token": access_token}
        )
        assert verify_resp.status_code == 200
        assert verify_resp.json()["valid"] is True

        # 3. Access Protected /me endpoint
        me_resp = await client.get(
            "/api/v1/auth/me",
            headers={"Authorization": f"Bearer {access_token}"}
        )
        assert me_resp.status_code == 200
        assert me_resp.json()["cellphone_number"] == "+12025550199"

        # 4. Refresh token
        refresh_resp = await client.post(
            "/api/v1/auth/refresh",
            json={"refresh_token": refresh_token}
        )
        assert refresh_resp.status_code == 200
        new_tokens = refresh_resp.json()
        assert "access_token" in new_tokens
        assert new_tokens["access_token"] != access_token

        # 5. Logout
        logout_resp = await client.post(
            "/api/v1/auth/logout",
            json={"refresh_token": new_tokens["refresh_token"]},
            headers={"Authorization": f"Bearer {new_tokens['access_token']}"}
        )
        assert logout_resp.status_code == 200
        assert logout_resp.json()["success"] is True


@pytest.mark.asyncio
async def test_invalid_login():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.post(
            "/api/v1/auth/login",
            json={
                "cellphone_number": "+19999999999",
                "password": "wrong"
            }
        )
        assert resp.status_code == 401
