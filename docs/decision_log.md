# Decision Log & Clarifications

This document logs the architectural and business decisions addressing the ten areas highlighted in Section 6 of the authoritative prompt.

---

### D01. Trial Accounting & Interaction Rules (P02, P17, P18)
- **Cooldown vs. Window Quota:** Both rules are enforced conjunctively.
  - A user cannot request a trial on Plan X if they obtained one within the last `trial_cooldown_seconds` (e.g., 24 hours).
  - Additionally, the user cannot exceed `trial_max_per_window` (e.g., 2 trials) within the rolling window `trial_window_seconds` (e.g., 7 days = 604,800 seconds).
- **Window Type:** Rolling window calculated against `trial_records.created_at >= now - window_seconds`.
- **Per-Plan Independence:** Enforced strictly by filtering `trial_records` on `plan_id`. Consuming a trial on Plan A has zero impact on Plan B.
- **Reseller Daily Cap:** Reseller tiers enforce an aggregate daily cap across all plans (Free: 10/day, Pro: 25/day, Ultimate: 100/day, configurable). Reseller-issued trials check both the plan-specific eligibility and the reseller's aggregate daily limit.
- **Trial Variant Selection:** An Ultimate reseller can choose among Free, Pro, or Ultimate trial variants when creating a trial. Free resellers only receive the Free variant; Pro resellers can choose Free or Pro.
- **Child-Bot Trials:** When a customer requests a trial in a child bot, the child bot uses the variant configured by the reseller for that plan (within the reseller's tier allowance), and the trial counts against the reseller's daily trial allowance.

---

### D02. Pricing, Durations, and Scale (P03, P08, P09)
- **Duration Scale:** 1 month is standardized as 30 calendar days (720 hours = 2,592,000 seconds). Durations are configured as integer multiples of months (e.g., 1, 2, 3, 6, 12 months).
- **Quotation Formula:**
  $$\text{effective\_users} = \max(\text{users}, \text{base\_users})$$
  $$\text{extra\_users} = \text{effective\_users} - \text{base\_users}$$
  $$\text{monthly\_price} = \text{base\_price} + (\text{extra\_users} \times \text{extra\_user\_price})$$
  $$\text{discount\_pct} = \text{discounts}[\text{duration\_months}] \quad (\text{default } 0\%)$$
  $$\text{total} = \text{round}\left(\text{monthly\_price} \times \text{duration\_months} \times \frac{100 - \text{discount\_pct}}{100}\right)$$
- **Toman Arithmetic & Display:**
  - Base input `200` means 200,000 Toman.
  - Stored in the database as exact integer units (`int64` in full Toman units, e.g., 200,000).
  - Persian display formats thousands as `هزار تومان` and millions as `میلیون تومان` (e.g., `200 هزار تومان`, `1 میلیون تومان`).
  - English display formats as `200 thousand toman` and `1 million toman`.
- **User Count Changes:**
  - **Increase:** The customer pays the prorated difference for the remaining active days in their current period:
    $$\Delta \text{users} \times \text{extra\_user\_price} \times \frac{\text{remaining\_seconds}}{30 \times 86400}$$
  - **Decrease:** No refund or credit is issued (standard SaaS policy). The user count is lowered, a new `subId` is generated, and the subscription link is immediately rotated to enforce the new device limit.

---

### D03. Paid Service Limits & Renewal Behavior (P03, P05, P08)
- **Traffic Allowance:** Each plan defines `traffic_bytes` (e.g., 50 GB = $50 \times 1024^3$ bytes).
- **Renewal Stacking:**
  - If a service is renewed before expiration ($\text{expiry} > \text{now}$), the new duration adds directly to the existing expiration date ($\text{new\_expiry} = \text{current\_expiry} + \text{duration\_ms}$).
  - If renewed after expiration, $\text{new\_expiry} = \text{now} + \text{duration\_ms}$.
  - Traffic counters are reset in 3x-ui upon renewal (`POST /panel/api/clients/resetTraffic/{email}`) and the total allowance is refreshed.

---

### D04. Refund Settlement (P14)
- **Refund Eligibility:** Allowed if requested within `refund_window_days` from original purchase date.
- **Refund Amount:** Full purchase price is refunded to the customer's wallet balance inside that bot.
- **Service Revocation:** Upon administrator approval of the refund, the client is disabled/deleted in 3x-ui.
- **Referral Clawback:** Any referral reward credited for that purchase is debited from the referrer's wallet.

---

### D05. Referral Settlement (P15)
- **Reward Percentage:** Configured by administrator per instance (`instances.referral_percentage`, e.g., 10%).
- **Earning Trigger:** Credited upon successful completion/fulfillment of a purchase or renewal order.
- **Anti-Double-Counting:** Wallet funding (top-ups) never grants referral rewards. Only finalized purchases of VPN services generate commissions.
- **Destination:** Credited directly to the referrer's internal bot wallet ledger.

---

### D06. Reseller Economics & Shared Wallet (P19, P20)
- **Wholesale Obligation:** Resellers purchase subscriptions at parent wholesale plan prices.
- **Child Customer Purchases:**
  - Customer pays according to the plan prices configured in the child bot.
  - For customer wallet payments, customer's child-bot wallet is debited.
  - Concurrently, the reseller's wallet in the parent instance must have sufficient balance for the wholesale cost.
  - The wholesale cost is atomically reserved from the reseller's wallet.
  - If the reseller's wallet balance is insufficient: the order is blocked, the customer receives "Insufficient provider credit. Please contact support.", and the reseller receives an immediate notification in the parent reseller bot.

---

### D07. Membership Lifecycle & Tiers (P17, P18)
- **Tiers:** Free (tier 0), Pro (tier 1), Ultimate (tier 2).
- **Privileges:**
  - Free: Default trial quota (10/day), 1-day refund window, basic plans.
  - Pro: Higher trial quota (25/day), 2-day refund window, access to Pro plans, child bot capability.
  - Ultimate: Maximum trial quota (100/day), 3-day refund window, Ultimate plans, choice of all 3 trial variants, web panel access.
- **Downgrade/Expiration:** If a membership expires, tier reverts to Free. Existing provisioned clients remain active, but new operations are bounded by Free tier limits.

---

### D08. Reseller Management Powers in My Services (P16)
- Resellers in their My Services section have enhanced actions for clients they own:
  1. View live client traffic usage (upload, download, total).
  2. Reset client traffic counters (`POST /panel/api/clients/resetTraffic/{email}`).
  3. Toggle enable/disable client.
  4. Force rotate `subId` and regenerate subscription link.
  5. Renew or extend client subscription.
- Strict isolation: Resellers can only inspect and manage clients associated with their own `group_name`.

---

### D09. Settings Inheritance & Historical Integrity (P19)
- **Child Bot Inheritance:**
  - Inherits: 3x-ui panel URL, 3x-ui API key, reseller numeric Telegram ID as admin.
  - Overridable: Bot token, default language, bank card info, support ID, custom pricing markup.
- **Historical Immutability:** Historical orders and ledger transactions store the exact terms, prices, and snapshots at purchase time. Editing a plan never alters previous order records.

---

### D10. Backup & Migration Boundaries (P22, P23)
- **Global Backup:** Complete database dump + system configuration + encrypted storage of tokens and uploaded media receipts.
- **Instance Backup:** Self-contained archive for a single bot instance (including plans, users, services, orders, tickets, and transactions). Parent backups include references to child instances.
- **Safe Cutover:** During migration, old instance listeners are stopped before the new instance starts polling to prevent dual-processing of updates or double payments.
