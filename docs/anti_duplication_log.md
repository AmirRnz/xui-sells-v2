# Anti-Duplication Milestone Gate Log

This log is maintained by the Gemini 3.8 Flash High orchestrator. Before each milestone is accepted, an explicit code inspection is conducted to ensure that business logic is strictly implemented once in the shared core and never duplicated across Telegram adapters, HTTP handlers, CLI wizards, or worker tasks.

## Anti-Duplication Rules & Verification Invariants

1. **Pricing & Quotation:**
   - Single source of truth: `internal/app/pricing.CalculateQuote()`
   - Telegram customer wizard, admin client creator, web panel checkout, and CLI must all call `pricing.CalculateQuote()`.
2. **Trial Eligibility & Quotas:**
   - Single source of truth: `internal/app/trial.CheckEligibility()` and `internal/app/trial.ConsumeTrial()`
   - Per-plan windows, cooldowns, and reseller tier caps are evaluated in this single shared package.
3. **Reseller Credit & Shared Wallet:**
   - Single source of truth: `internal/app/wallet.ReserveResellerCredit()` and `internal/app/wallet.SettleReservation()`
   - Child-bot checkout and parent reseller checks use atomic PostgreSQL row locking via this package.
4. **Subscription Provisioning & ID Rotation:**
   - Single source of truth: `internal/app/provisioning.Service`
   - Adding clients, renewing, updating user count, rotating `subId` upon decrease, and generating QR codes are implemented solely in this service.
5. **Money & Currency Formatting:**
   - Single source of truth: `internal/domain/money.FormatToman()` and `internal/domain/money.FormatCurrency()`
   - Persian layout (`200 هزار تومان`, `1 میلیون تومان`) and English formatting are handled uniformly.
6. **Entitlements & Gating:**
   - Single source of truth: `internal/app/entitlement.Resolver`
   - Tier privileges (Free, Pro, Ultimate), feature flags, and max refund windows are resolved through this service for both bot and web panel.

---

## Milestone Gate Reviews

### Milestone 1: Core Foundation, Domain Schemas & 3x-ui Adapter
- **Status:** Completed
- **Scope Inspected:** `internal/domain/`, `internal/domain/money/`, `internal/infra/xui/`, `migrations/`
- **Adapters Checked:** Unit tests, Mock HTTP servers
- **Duplication Candidates Found:** None. Direct 3x-ui API mappings consolidated into `internal/infra/xui/client.go`. Exact monetary arithmetic and Persian/English Toman formatting consolidated in `internal/domain/money/money.go`.
- **Gate Result:** PASS (Zero duplicates identified, all tests passing)

### Milestone 2: Pricing, Wallet, Orders & Provisioning Vertical Slice
- **Status:** Completed
- **Scope Inspected:** `internal/app/pricing/`, `internal/app/wallet/`, `internal/app/order/`, `internal/app/provisioning/`, `internal/app/trial/`, `internal/app/entitlement/`
- **Adapters Checked:** Unit tests, Mock repositories, In-memory ledgers
- **Duplication Candidates Found:** None. Core logic consolidated into single sources of truth.
- **Gate Result:** PASS

### Milestone 3: Telegram Bot Engine (Customer, Reseller, Child Bot) & Support Services
- **Status:** Completed
- **Scope Inspected:** `internal/adapter/telegram/`, `internal/app/ticket/`, `internal/app/refund/`, `internal/app/stats/`, `internal/app/broadcast/`
- **Adapters Checked:** Unit tests, Mock telegram senders, Mock repositories
- **Duplication Candidates Found:** None. Handlers delegate directly to `pricing.CalculateQuote`, `provisioning.Service`, `trial.Service`, `wallet.Service`, `order.Service`, `i18n.Get`, `i18n.BankCardMessage`, and `i18n.UserCreditNotification`. Zero business logic duplicated across handlers.
- **Gate Result:** PASS (Zero duplicates identified, all tests passing)

### Milestone 4: Reseller Web Panel (React SPA Embed + Chi REST API)
- **Status:** Completed
- **Scope Inspected:** `internal/adapter/http/`, `web/embed.go`
- **Adapters Checked:** Chi HTTP server integration tests, session auth, tier gating, static SPA serving
- **Duplication Candidates Found:** None. Web panel routes delegate all entitlement gating to `entitlement.Resolver`, operational stats to `stats.Service`, and password hashing to standard Argon2id. Single source of truth preserved.
- **Gate Result:** PASS (Zero duplicates identified, all tests passing)

### Milestone 5: Host CLI, Backup/Restore & Unified Ubuntu Installer
- **Status:** Completed
- **Scope Inspected:** `cmd/xui-sells/`, `scripts/install.sh`, `internal/app/backup/`, `internal/app/instance/`, `docs/install_and_ops.md`
- **Adapters Checked:** Cobra CLI commands, interactive terminal wizard tests, AES-GCM encrypted backup/restore unit tests, roundtrip restoration
- **Duplication Candidates Found:** None. CLI commands and interactive wizards delegate to `backup.Service`, `instance.Store`, `telegram.Supervisor`, and `http.Server`. Scoped instance backups strictly omit shared ledger transactions/reservations (P23).
- **Gate Result:** PASS (Zero duplicates identified, all tests passing)
