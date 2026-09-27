# System Architecture & Technical Specifications

## 1. Architectural Overview

The system is structured as a Go modular monolith. All business logic, transaction lifecycles, ledger mutations, entitlements, and external 3x-ui provisioning reside within a shared core. Thin presentation adapters (Telegram bots, HTTP web panel, CLI commands) interact with the core through typed service interfaces.

```
                           +---------------------------+
                           |  Host CLI (cmd/xui-sells) |
                           +-------------+-------------+
                                         |
     +-----------------------------------+-----------------------------------+
     |                                   |                                   |
+----+--------------------+   +----------+-------------+   +-----------------+-----------------+
| Telegram Adapter        |   | HTTP Adapter & Web     |   | Durable Workers (River)           |
| - Customer Bot Engine   |   | - REST API             |   | - Provisioning retries            |
| - Reseller Bot Engine   |   | - Embedded React Panel |   | - Order expiration reconciler     |
| - Child Bot Instances   |   | - Session Auth         |   | - Notification dispatcher         |
+----+--------------------+   +----------+-------------+   +-----------------+-----------------+
     |                                   |                                   |
     +-----------------------------------+-----------------------------------+
                                         |
                      +------------------v------------------+
                      | Shared Application Core             |
                      | - Pricing & Duration Engine         |
                      | - Wallet Ledger & Shared Credit     |
                      | - Order & Approval State Machines   |
                      | - Provisioning & ID Rotation        |
                      | - Trial Quotas & Independence       |
                      | - Entitlement & Tier Resolver       |
                      | - Ticket & Referral Workflows       |
                      | - Unified Statistics & Metrics      |
                      +------------------+------------------+
                                         |
                      +------------------v------------------+
                      | Infrastructure Layer                |
                      | - 3x-ui Client (Bearer Auth)        |
                      | - PostgreSQL Storage (pgx, Goose)   |
                      | - AES-GCM Encrypted Backup / Media  |
                      +-------------------------------------+
```

## 2. Monorepo Directory Layout

```
xui-sells-v2/
├── 3x-ui_openapi.json            # Authoritative 3x-ui OpenAPI specification
├── docs/                         # Authoritative contracts, traceability, decision logs
│   ├── 3xui_contract.md
│   ├── anti_duplication_log.md
│   ├── architecture.md
│   ├── decision_log.md
│   ├── install_and_ops.md
│   └── traceability_matrix.md
├── cmd/
│   └── xui-sells/                # CLI entrypoint (Cobra)
│       ├── main.go
│       ├── menu.go               # Interactive terminal wizard
│       ├── instance.go           # Instance management commands
│       ├── backup.go             # Global & instance backup/restore
│       ├── update.go             # System self-updater
│       └── run.go                # Monolith daemon runner
├── migrations/                   # Versioned SQL migrations (Goose)
│   ├── 001_initial_schema.sql
│   └── ...
├── internal/
│   ├── domain/                   # Pure domain models and value objects
│   │   ├── money/                # Exact arithmetic, Toman formatting
│   │   ├── instance.go           # Instance, ResellerProfile
│   │   ├── plan.go               # Plan, Duration, Discount models
│   │   ├── service.go            # VPN Subscription client models
│   │   ├── order.go              # Order, PaymentMethod, Status
│   │   ├── wallet.go             # Ledger transaction, Reservation
│   │   ├── trial.go              # TrialRecord, TierVariant
│   │   ├── refund.go             # RefundRequest, Status
│   │   ├── ticket.go             # Ticket, TicketMessage
│   │   └── user.go               # Bot user identity & language
│   ├── infra/                    # External integrations and persistence
│   │   ├── db/                   # Database connection pool (pgxpool)
│   │   ├── xui/                  # 3x-ui client implementation & tests
│   │   └── storage/              # Local media & encrypted file storage
│   ├── app/                      # Shared business services (Application layer)
│   │   ├── instance/             # Instance registry & supervisor service
│   │   ├── pricing/              # Quotation & user count prorating
│   │   ├── wallet/               # Atomic ledger & reseller credit reservation
│   │   ├── order/                # Order creation, direct receipt, approval workflow
│   │   ├── provisioning/         # 3x-ui provisioning, renewal, subId rotation
│   │   ├── trial/                # Per-plan trial eligibility & quota engine
│   │   ├── entitlement/          # Reseller tier privileges & feature gates
│   │   ├── refund/               # Refund eligibility & processing
│   │   ├── referral/             # Referral commission calculations
│   │   ├── ticket/               # Support ticket management
│   │   ├── broadcast/            # Broadcast message dispatcher
│   │   ├── stats/                # Unified reporting & metrics
│   │   ├── i18n/                 # Localization catalogs (FA / EN)
│   │   └── backup/               # Backup archive creation & restoration
│   └── adapter/                  # Presentation adapters
│       ├── telegram/             # go-telegram/bot implementation
│       │   ├── supervisor.go     # Multi-instance bot lifecycle runner
│       │   ├── router.go         # Command & callback query router
│       │   ├── customer.go       # Customer purchase & service menus
│       │   ├── reseller.go       # Reseller onboarding, memberships, child-bot setup
│       │   ├── admin.go          # /admin menus (plans, settings, users, approvals)
│       │   └── qrcode.go         # PNG QR code generator
│       └── http/                 # Chi HTTP router & Reseller Web Panel API
│           ├── server.go         # HTTP server with embedded frontend
│           ├── auth.go           # Telegram ID + password login, Argon2id, sessions
│           ├── handlers.go       # REST endpoints for web panel
│           └── middleware.go     # Session auth, CSRF, security headers
├── web/                          # Reseller Web Panel (React + TS + Vite + Tailwind)
│   ├── package.json
│   ├── vite.config.ts
│   └── src/
│       ├── components/           # UI components (shadcn/ui style, RTL support)
│       ├── pages/                # Dashboard, Services, Orders, Approvals, ChildBots
│       └── api/                  # Typed API client
├── scripts/
│   ├── install.sh                # One-command Ubuntu installation script
│   └── docker-compose.yml        # Container deployment spec
├── go.mod
└── go.sum
```

## 3. Dependency Manifest & Compatibility Rationale

- **Go Version:** `1.24+` (building seamlessly on installed `go1.26.4 windows/amd64` and Linux target).
- **Chi Router:** `github.com/go-chi/chi/v5` - Lightweight, standard `http.Handler` compatible.
- **PostgreSQL Driver:** `github.com/jackc/pgx/v5` - Modern, high-performance PostgreSQL driver with connection pooling (`pgxpool`) and atomic transaction support.
- **Goose Migrations:** `github.com/pressly/goose/v3` - Standard versioned database migrations.
- **Password Hashing:** `golang.org/x/crypto/argon2` - Reseller web panel password hashing.
- **QR Code Generation:** `github.com/skip2/go-qrcode` - Fast in-memory PNG QR code generator for subscription links.
- **Telegram Bot SDK:** `github.com/go-telegram/bot` - Clean, context-aware Telegram Bot API wrapper.
- **CLI Framework:** `github.com/spf13/cobra` - Industry standard CLI builder with interactive prompt integration.
- **Frontend Stack:** React 19, TypeScript 5.8, Vite 6, Tailwind CSS 4, Apache ECharts.
