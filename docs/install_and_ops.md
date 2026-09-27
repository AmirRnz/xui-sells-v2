# Operating & Installation Guide (xui-sells-v2)

This documentation provides complete instructions for single-command installation, host CLI administration, interactive menu operation, encrypted backup and disaster recovery, and system update procedures conforming strictly to **P22**, **P23**, and **P24**.

---

## 1. Single-Command Ubuntu Installation (P22)

`xui-sells-v2` is packaged as a unified, single-binary distribution with embedded static React web assets and an automated Docker Compose orchestrator.

### Prerequisites
- **Operating System:** Ubuntu 20.04, 22.04, or 24.04 LTS (x86_64 or aarch64/ARM64).
- **Permissions:** Root access or `sudo` privileges.
- **Upstream Service:** An accessible [3x-ui](https://github.com/MHSanaei/3x-ui) panel instance.

### Automated One-Line Install
Run the following command on your target server:

```bash
curl -fsSL https://raw.githubusercontent.com/AmirRnz/xui-sells-v2/master/scripts/install.sh | sudo bash
```

Alternatively, if you cloned the repository locally:
```bash
sudo bash scripts/install.sh
```

### What the Installer Does Automatically
1. **Verifies OS & Root Permissions:** Checks for Ubuntu/Debian compatibility.
2. **Detects Port Conflicts:** Checks if port `8080` is in use by an existing 3x-ui panel or VPN service, dynamically selecting an open port in the range `8081–8090` without conflicts.
3. **Installs Container Runtime:** Installs Docker CE and Docker Compose plugin if not already present.
4. **Initializes Environment:** Configures `/opt/xui-sells/.env` with secure random PostgreSQL credentials and data directory mounts.
5. **Sets Up Host CLI:** Installs a global wrapper script at `/usr/local/bin/xui-sells` so you can manage the application directly from any shell session.
6. **Launches System Services:** Starts the database, web panel, and bot supervisor via Docker Compose.

---

## 2. Interactive Terminal Wizard (`xui-sells menu`) (P22)

To manage instances, child bots, backups, or updates interactively, run:

```bash
xui-sells menu
```

### Main Menu Overview
```text
================================================
       xui-sells-v2 Management Menu (P22)      
================================================
1. View / Manage Instances
2. Add New Instance
3. Backup & Restore
4. Update System
5. Exit
Select an option [1-5]:
```

### A. View / Manage Instances (P22 Hierarchy Display)
Displays all registered instances. For each parent reseller instance, **the numeric child-bot count is prominently displayed**:

```text
------------------------------------------------
               Active Instances                 
------------------------------------------------
[ID: 1] Tehran VIP Reseller (reseller) | Admin: 987654321 | Status: Active | Child Bots: 2
   └── [Child ID: 2] Tehran Child Bot 1 (Token: 612345:******) | Status: Active
   └── [Child ID: 3] Tehran Child Bot 2 (Token: 698765:******) | Status: Active
[ID: 4] Direct Customer Bot (customer) | Admin: 112233445 | Status: Active
------------------------------------------------
Enter Instance ID to manage, or 'b' to go back:
```

Selecting an instance ID allows you to:
- Toggle between `Active` and `Disabled`.
- Edit instance name.
- Update Telegram bot token.
- Delete the instance and its child bots.

### B. Add New Instance Wizard (7-Step Conformance)
Guides the administrator through a sequential setup strictly adhering to P22:

1. **Select Instance Type:**
   - `1) Ordinary Customers (customer)`
   - `2) Reseller (reseller)`
2. **Instance Name:** Custom descriptive label (e.g. `VIP Reseller Bot`).
3. **Telegram Bot Token:** Obtained from [@BotFather](https://t.me/botfather).
4. **Administrator Telegram ID:** Numeric Telegram ID of the admin.
5. **3x-ui Panel URL:** Upstream panel address (e.g. `https://panel.example.com:2053`).
6. **3x-ui API Key / Cookie Credentials:** Panel authentication credentials.
7. **Default Bot Language:** `fa` (Persian / فارسی) or `en` (English).
8. **Reseller Web Panel Activation:** If instance type is `reseller`, prompts:
   `Bring Reseller Web Panel online? (y/n, default: y)`.

---

## 3. Non-Interactive CLI Reference

For automated scripts, CI/CD, or headless orchestration, `xui-sells` exposes dedicated subcommands:

### Instance Management
```bash
# List all instances and child bots
xui-sells instance list

# Add an ordinary customer bot
xui-sells instance add \
  --name "Retail Bot 1" \
  --type customer \
  --token "123456789:ABCDefgh..." \
  --admin-tg 99887766 \
  --panel-url "https://panel.example.com:2053" \
  --api-key "my-panel-api-key" \
  --lang fa

# Add a reseller bot
xui-sells instance add \
  --name "Reseller Master" \
  --type reseller \
  --token "987654321:XYZ..." \
  --admin-tg 55443322 \
  --panel-url "https://panel.example.com:2053" \
  --api-key "my-panel-api-key" \
  --lang fa

# View instance details as JSON
xui-sells instance get 1

# Edit instance attributes
xui-sells instance edit 1 --name "New Reseller Brand" --active true

# Delete an instance
xui-sells instance delete 1
```

### Encrypted Backups (P23)
```bash
# Create global system backup (encrypted with AES-256-GCM)
xui-sells backup create \
  --scope global \
  --password "StrongMasterPass123!" \
  --out /var/backups/xui-sells-global.tar.gz.enc

# Create scoped instance backup (preserves child bots without duplicating shared ledger)
xui-sells backup create \
  --scope instance \
  --instance-id 1 \
  --password "StrongMasterPass123!" \
  --out /var/backups/reseller-1-backup.tar.gz.enc

# Restore backup archive
xui-sells backup restore \
  --file /var/backups/xui-sells-global.tar.gz.enc \
  --password "StrongMasterPass123!"
```

### System Updates (P24)
```bash
# Check if new versions or schema migrations are available
xui-sells update --check-only

# Apply updates, pull latest images, and run migrations
xui-sells update
```

### Running Background Server Manually
```bash
# Start HTTP web panel and Telegram supervisor
xui-sells run --http-port 8080 --bind 0.0.0.0
```

---

## 4. Backup & Disaster Recovery Procedures (P23)

### Cryptographic Security Model
Every backup archive produced by `xui-sells-v2` is encrypted using authenticated **AES-256-GCM**:
- **Key Derivation:** PBKDF2 with HMAC-SHA-256, 64,000 iterations, and a cryptographically secure 16-byte random salt.
- **Envelope Integrity:** Includes a 12-byte random nonce, AES-GCM authentication tag, and an internal SHA-256 manifest hash of the uncompressed data.
- **Tamper Protection:** Any modification to ciphertext, headers, or salt results in immediate rejection (`ErrDecryptionFailed` or `ErrChecksumMismatch`).

### Global Disaster Recovery Procedure
1. Transfer the backup file (`*.tar.gz.enc`) to the new server.
2. Ensure `xui-sells` is installed.
3. Execute the restoration command:
   ```bash
   xui-sells backup restore --file /path/to/backup.tar.gz.enc --password "<YourPassword>"
   ```
4. Restart the runtime containers:
   ```bash
   docker compose restart
   ```

### Scoped Instance Migration (P23 Conformance)
When backing up a specific reseller instance:
- **Included:** Reseller bot profile, child customer bots, plans, active/expired services, customer orders, and support tickets.
- **Excluded (Zero Duplication):** Parent wallet ledger balances, ledger transactions, and credit reservations are **omitted** to ensure that shared financial state is never duplicated across instances.
- **Restoration:**
  ```bash
  xui-sells backup restore \
    --file /path/to/reseller-backup.tar.gz.enc \
    --password "<YourPassword>" \
    --target-instance-id <NewTargetID>
  ```

---

## 5. System Updates & Rollback Procedures (P24)

### Applying Updates
The update command automates repository pull, database schema validation, and service reload:

```bash
xui-sells update
```

Under Docker Compose:
```bash
cd /opt/xui-sells
git pull origin master
docker compose up -d --build
```

### Rollback Procedure
If a migration or update must be rolled back:
1. Stop the running containers:
   ```bash
   docker compose down
   ```
2. Checkout the desired previous Git tag or commit:
   ```bash
   git checkout tags/v1.9.0
   ```
3. Restore the pre-update backup archive:
   ```bash
   xui-sells backup restore --file /var/backups/pre-update-backup.tar.gz.enc --password "<Password>"
   ```
4. Bring services back up:
   ```bash
   docker compose up -d --build
   ```

---

## 6. Operational Troubleshooting & Health Checks

| Symptom | Cause | Solution |
| :--- | :--- | :--- |
| **Port Conflict on Port 8080** | Another web server (e.g. Nginx, 3x-ui panel) is listening on `8080`. | The installer automatically re-routes to `8081–8090`. Modify `WEB_PORT` in `/opt/xui-sells/.env` if a specific port is desired. |
| **Bot not responding to `/start`** | Invalid Telegram bot token or bot disabled in CLI. | Verify token via `xui-sells instance get <id>`, or test token validity directly with [@BotFather](https://t.me/botfather). |
| **Backup restore failed (`ErrDecryptionFailed`)** | Incorrect password or truncated archive. | Check password accuracy and verify file transfer size with `ls -lh`. |
| **3x-ui Provisioning Error** | Panel URL unreachable or API key expired. | Update panel credentials in `xui-sells menu` under instance settings; verify panel firewall allows incoming connections on the 3x-ui port. |
