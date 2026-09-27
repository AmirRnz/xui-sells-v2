# XUI-SELLS-V2

> **Enterprise-grade Telegram Sales, Reseller Network & Management Platform for 3x-ui Panels**

`xui-sells-v2` is an all-in-one Go modular monolith and React 19 web platform designed to sell VPN subscriptions through Telegram bots to two distinct buyer categories:
1. **Ordinary Customers**
2. **Resellers** who sell subscriptions to their own customer base through reseller-owned child bots.

The entire system—customer bots, reseller bots, child customer bots, web panel, database migrations, and administration CLI—operates from a single unified codebase.

---

## 🚀 One-Command Ubuntu Installation

Install the entire platform and dependencies with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/AmirRnz/xui-sells-v2/master/scripts/install.sh | sudo bash
```

The installer automatically:
- Checks OS compatibility (Ubuntu 20.04, 22.04, 24.04 LTS).
- Detects port conflicts with existing 3x-ui / VPN services on port `8080` and dynamically selects a safe port.
- Installs Docker CE and Docker Compose if needed.
- Configures persistent data directories and secure PostgreSQL credentials.
- Compiles or starts the system containers.
- Links the global CLI tool at `/usr/local/bin/xui-sells`.

---

## ⚡ Interactive Terminal Console (`xui-sells menu`)

Once installed, open the interactive administration console anytime:

```bash
xui-sells menu
```

### Menu Features:
1. **Add Instance**: 7-step wizard to provision new customer or reseller bot instances.
2. **View / Manage Instances**: Lists running bots and displays the **numeric child-bot count** for each parent reseller bot.
3. **Backup & Restore**: Create and restore AES-256-GCM encrypted global or scoped instance archives.
4. **Update System**: One-click software update and migration runner.

---

## 🌟 Key Architecture & Product Capabilities

### 1. Pure 3x-ui OpenAPI Fidelity
- Built strictly against the authoritative [`3x-ui_openapi.json`](3x-ui_openapi.json) contract.
- Authenticates via Bearer API tokens (`Authorization: Bearer <API_TOKEN>`).
- Supports multi-inbound attachments, `limitIp` mapping for concurrent user limits, and live traffic resets upon renewal.
- Automatically queries `/panel/api/setting/all` to read `subURI` and `subPath` to deliver canonical subscription URLs and QR codes.

### 2. Reseller Network & Child Bots
- **First-Action Reseller Onboarding**: Reseller bots enforce setting a unique service/brand name before unlocking bot commands; this name is used as the 3x-ui client `group`.
- **In-Bot Child Bot Creation**: Resellers create their own customer bots directly inside Telegram by submitting a bot token from `@BotFather`.
- **Shared Reseller Credit**: Resellers maintain a wholesale credit balance. Customer checkouts in child bots place atomic holds on the parent reseller's wallet. If the reseller has insufficient funds, the customer receives an alert to contact support, and the reseller is notified immediately.
- **Reseller Memberships**: Free, Pro, and Ultimate tiers gating trial quotas, refund windows, plan access, and feature privileges.
- **Tier-Specific Trial Variants**: Ultimate resellers can choose between Free, Pro, or Ultimate trial parameters.

### 3. Subscription ID Rotation on User Count Decrease
- When a customer decreases their concurrent user count (`limitIp`), the system automatically **rotates the `subId` upstream in 3x-ui**, immediately revoking the previous link to prevent device limit bypass, and delivers a fresh subscription URL and QR code.

### 4. Reseller Web Panel (React 19 + TypeScript + Vite + ECharts)
- Modern dark-mode management interface embedded directly into the Go binary.
- Resellers set their password inside Telegram and log in using their numeric Telegram ID.
- Features dynamic multi-series traffic charts, revenue trends, service tables with QR modals, order approvals, and full Persian **RTL** (`dir="rtl"`) / English **LTR** switching.

### 5. Financial & Ledger Invariants
- Zero floating-point arithmetic: uses exact integer money representation.
- Centralized Toman input scaling: input `200` represents `200 هزار تومان` (200,000 Toman); 5 months costs `1 میلیون تومان` (never `1000 هزار تومان`).
- Strict Persian bank card payment structure:
  ```text
  شماره کارت جهت واریز:
  <CARD_NUMBER>
  نام صاحب کارت: <CARDHOLDER_NAME>
  ```
- Anti-double-counting referral rewards: commission percentages are awarded only on completed VPN orders, never on wallet top-ups.

### 6. Encrypted Disaster Recovery & Migrations
- Global and scoped instance backups using authenticated **AES-256-GCM** encryption with PBKDF2/SHA-256 key derivation and SHA-256 integrity checksums.
- Scoped instance migrations preserve parent-child relations while excluding shared financial ledgers to prevent duplicating state.

---

## 📚 Technical Documentation

- 📖 [Installation & Operating Guide](docs/install_and_ops.md)
- 🏗️ [System Architecture](docs/architecture.md)
- 📋 [Traceability Matrix (P01–P25)](docs/traceability_matrix.md)
- 🔌 [3x-ui Integration Contract](docs/3xui_contract.md)
- ⚖️ [Decision Log (Section 6)](docs/decision_log.md)
- 🛡️ [Anti-Duplication Milestone Gate Log](docs/anti_duplication_log.md)

---

## 🛠️ Verification & Test Suite

Run the complete test suite across all 18 Go packages:

```bash
go test -v ./...
```

Build the React Web Panel frontend bundle:

```bash
cd web && npm run build
```

Compile the standalone production binary:

```bash
go build -o bin/xui-sells ./cmd/xui-sells
```

---

## 📄 License
This project is proprietary and confidential. All rights reserved.
