# License Server API 文档

Base URL: `https://your-domain.com`

## 认证

管理接口需要在请求头中携带 Bearer Token：

```
Authorization: Bearer <ADMIN_TOKEN>
```

`/api/license/issue` 不需要认证。

---

## POST /api/license/issue

客户端获取授权。服务端自动识别客户端 IP（支持 CF-Connecting-IP / X-Forwarded-For / X-Real-IP）。

**请求**：无需 body

**成功响应** `200`：

```json
{
  "license_id": "550e8400-e29b-41d4-a716-446655440000",
  "allowed_ips": ["69.63.204.139"],
  "issued_at": "2026-04-28T10:00:00Z",
  "expires_at": null,
  "signature": "base64-encoded-ed25519-signature"
}
```

响应头 `X-Detected-IP` 返回服务端检测到的客户端 IP。

**失败响应** `403`：

```json
{
  "error": "该IP未被授权",
  "detected_ip": "69.63.204.139"
}
```

---

## GET /api/license/list

获取所有授权记录。需要认证。

**响应** `200`：

```json
[
  {
    "license_id": "550e8400-e29b-41d4-a716-446655440000",
    "allowed_ips": ["1.2.3.4"],
    "issued_at": "2026-04-28T10:00:00Z",
    "expires_at": null,
    "signature": "base64-encoded-ed25519-signature"
  }
]
```

---

## POST /api/license/create

新增授权。需要认证。

**请求体**：

```json
{
  "allowed_ips": ["1.2.3.4", "5.6.7.8"],
  "expires_at": "2027-01-01T00:00:00Z"
}
```

`expires_at` 传 `null` 或不传表示永久有效。

**响应** `200`：返回完整的 License JSON（含签名）。

---

## POST /api/license/update

修改授权。需要认证。

**请求体**：

```json
{
  "license_id": "550e8400-e29b-41d4-a716-446655440000",
  "allowed_ips": ["1.2.3.4", "5.6.7.8"],
  "expires_at": "2027-01-01T00:00:00Z"
}
```

**响应** `200`：

```json
{ "message": "updated" }
```

---

## POST /api/license/revoke

吊销授权。需要认证。

**请求体**：

```json
{
  "license_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**响应** `200`：

```json
{ "message": "revoked" }
```

---

## Ed25519 签名说明

签名载荷 = `json.Marshal` 不含 `signature` 字段的 License struct（字段顺序：license_id → allowed_ips → issued_at → expires_at）。

客户端验证：

```go
pubKeyBytes, _ := base64.StdEncoding.DecodeString(publicKeyBase64)
sigBytes, _ := base64.StdEncoding.DecodeString(license.Signature)
payload, _ := json.Marshal(License{
    LicenseID:  license.LicenseID,
    AllowedIPs: license.AllowedIPs,
    IssuedAt:   license.IssuedAt,
    ExpiresAt:  license.ExpiresAt,
})
valid := ed25519.Verify(pubKeyBytes, payload, sigBytes)
```

公钥在服务端首次启动时生成，日志中会打印 base64 编码值。
