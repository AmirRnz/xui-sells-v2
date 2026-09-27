# 3x-ui API Integration Contract

This contract is derived strictly and exclusively from `3x-ui_openapi.json` (OpenAPI 3.0.3) included in the repository. All subagents and modules interacting with 3x-ui must adhere to the models and endpoints specified here.

## 1. Authentication & Base Connection

- **URL:** Supplied panel URL, e.g., `http://127.0.0.1:2053` or `https://panel.example.com`.
- **Authentication Scheme:** `bearerAuth`.
  - HTTP header: `Authorization: Bearer <API_TOKEN>`
  - Generated in 3x-ui via `Settings -> Security -> API Token` (or `/panel/api/setting/apiTokens/create`).
  - No cookie-based browser session is used for machine API calls.

## 2. Inbound Operations

### 2.1. List Inbounds
- **Endpoint:** `GET /panel/api/inbounds/list`
- **Response Shape:**
  ```json
  {
    "success": true,
    "msg": "",
    "obj": [
      {
        "id": 1,
        "up": 0,
        "down": 0,
        "total": 0,
        "remark": "VLESS-Reality",
        "enable": true,
        "expiryTime": 0,
        "listen": "",
        "port": 443,
        "protocol": "vless",
        "tag": "inbound-443",
        "streamSettings": "...",
        "clientStats": []
      }
    ]
  }
  ```

### 2.2. Get Inbound by ID
- **Endpoint:** `GET /panel/api/inbounds/get/{id}`

## 3. Client (Subscription) Operations

### 3.1. Add Client
- **Endpoint:** `POST /panel/api/clients/add`
- **Request Body:**
  ```json
  {
    "client": {
      "email": "user123_sub1",
      "subId": "random_sub_id_16_chars",
      "id": "uuid-v4-string",
      "flow": "xtls-rprx-vision",
      "totalGB": 53687091200,
      "expiryTime": 1735689600000,
      "limitIp": 2,
      "limitHwid": 0,
      "tgId": 123456789,
      "group": "reseller_service_name_or_default_group",
      "comment": "Sub ID: 42 | Plan: VIP",
      "enable": true
    },
    "inboundIds": [1, 2]
  }
  ```
- **Field Details:**
  - `email`: Unique identifier string for the client.
  - `subId`: Unique subscription ID for URL generation (`/{subPath}{subid}`).
  - `id`: UUID string (v4) for VLESS/VMess protocols.
  - `totalGB`: Traffic limit in **bytes** (despite name `totalGB`, the OpenAPI specification and example show bytes, e.g., `53687091200` = 50 GB).
  - `expiryTime`: Unix timestamp in **milliseconds** (e.g. `1735689600000`). `0` indicates no expiration.
  - `limitIp`: User Count! Maps directly to the IP limit in 3x-ui.
  - `tgId`: Numeric Telegram user ID (or `0` if unassigned).
  - `group`: 3x-ui client grouping name. Configured on ordinary instances; chosen during reseller onboarding on reseller instances.
  - `enable`: `true` to enable client traffic.
  - `inboundIds`: Array of integer IDs for inbounds to attach the client to.
- **Response:**
  ```json
  {
    "success": true,
    "msg": "Client added",
    "obj": {}
  }
  ```

### 3.2. Get Client by Email
- **Endpoint:** `GET /panel/api/clients/get/{email}`
- **Response Shape:**
  ```json
  {
    "success": true,
    "msg": "",
    "obj": {
      "client": { ... },
      "inboundIds": [1, 2],
      "traffic": {
        "up": 1024000,
        "down": 2048000,
        "total": 53687091200,
        "expiryTime": 1735689600000
      }
    }
  }
  ```

### 3.3. Update Client
- **Endpoint:** `POST /panel/api/clients/update/{email}`
- **Request Body:** Full `Client` object with updated fields:
  ```json
  {
    "email": "user123_sub1",
    "subId": "new_or_existing_sub_id",
    "id": "uuid-v4-string",
    "totalGB": 53687091200,
    "expiryTime": 1738368000000,
    "limitIp": 3,
    "tgId": 123456789,
    "group": "group_name",
    "enable": true
  }
  ```
- **Note:** In 3x-ui, update is a replace operation. The payload must include the desired state.

### 3.4. Delete Client
- **Endpoint:** `POST /panel/api/clients/del/{email}`
- **Response:** `{ "success": true, "msg": "Client deleted" }`

### 3.5. Reset Client Traffic
- **Endpoint:** `POST /panel/api/clients/resetTraffic/{email}`
- **Response:** `{ "success": true, "msg": "Traffic reset" }`

### 3.6. SubLinks & Protocol URLs
- **Endpoint:** `GET /panel/api/clients/subLinks/{subId}`
  - Returns array of raw protocol connection strings: `["vless://...", "vmess://..."]`.
- **Endpoint:** `GET /panel/api/clients/links/{email}`
  - Returns array of connection links for client across all attached inbounds.

## 4. Subscription URL Construction

To construct the universal subscription URL delivered to the customer with its QR code:
1. Fetch panel settings via `POST /panel/api/setting/all`.
2. Inspect `subURI` and `subPath`:
   - If `subURI` is populated (e.g. `https://sub.myvpn.com:8443/sub/`), the subscription link is:
     `<subURI><subId>` (handling trailing slashes correctly).
   - If `subURI` is empty, construct from panel base URL and `subPath` (default `/sub/`):
     `<panelURL><subPath><subId>`.
3. Generating QR Code:
   - Generate standard PNG QR code for the subscription URL using Go standard library / `skip2/go-qrcode`.

## 5. Subscription ID Rotation on User Count Decrease (P05)
When a customer decreases the user count (`limitIp`):
1. A new cryptographic `subId` is generated.
2. The client is updated via `POST /panel/api/clients/update/{email}` with the new `subId` and lower `limitIp`.
3. The old subscription URL becomes invalid upstream immediately (the previous `subId` returns 404 on `/{subPath}{old_subid}`).
4. The bot informs the customer that their previous link is now inactive, displays the new subscription URL, and attaches the fresh QR code.
