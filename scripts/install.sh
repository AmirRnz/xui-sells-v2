#!/usr/bin/env bash
# ==============================================================================
# XUI-SELLS-V2 One-Command Ubuntu Installer
# Requirement P22 & Section 5 Specification Conformance
# ==============================================================================

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

INSTALL_DIR="/opt/xui-sells"
BINARY_LINK="/usr/local/bin/xui-sells"
DEFAULT_WEB_PORT=8080

echo -e "${CYAN}${BOLD}"
echo "====================================================================="
echo "               XUI-SELLS-V2 SYSTEM INSTALLER                         "
echo "        Integrated VPN Selling System for Customers & Resellers     "
echo "====================================================================="
echo -e "${NC}"

# 1. Root Check
if [[ $EUID -ne 0 ]]; then
   echo -e "${RED}[ERROR] This script must be run as root (or with sudo).${NC}" 
   exit 1
fi

# 2. OS Verification
if [[ -f /etc/os-release ]]; then
    . /etc/os-release
    OS_NAME=$ID
    OS_VERSION=$VERSION_ID
    echo -e "${BLUE}[INFO] Detected OS: ${NAME} ${VERSION}${NC}"
else
    echo -e "${RED}[ERROR] Cannot detect operating system release.${NC}"
    exit 1
fi

if [[ "$OS_NAME" != "ubuntu" && "$OS_NAME" != "debian" ]]; then
    echo -e "${YELLOW}[WARNING] This installer is optimized for Ubuntu Linux. Proceeding anyway...${NC}"
fi

# 3. Port Conflict Detection (P22 & Section 5)
echo -e "${BLUE}[INFO] Checking for port conflicts with existing 3x-ui / VPN services...${NC}"
WEB_PORT=$DEFAULT_WEB_PORT

if ss -tuln | grep -q ":${WEB_PORT} "; then
    echo -e "${YELLOW}[WARNING] Port ${WEB_PORT} is currently in use by another service on this host!${NC}"
    # Find an open port
    for p in {8081..8090}; do
        if ! ss -tuln | grep -q ":${p} "; then
            WEB_PORT=$p
            echo -e "${GREEN}[INFO] Assigned non-conflicting web port: ${WEB_PORT}${NC}"
            break
        fi
    done
fi

# 4. Dependency Checks (Docker & Docker Compose)
echo -e "${BLUE}[INFO] Checking runtime dependencies...${NC}"
if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}[INFO] Docker not found. Installing Docker CE...${NC}"
    apt-get update -y
    apt-get install -y ca-certificates curl gnupg lsb-release
    mkdir -p /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
    apt-get update -y
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
    systemctl enable --now docker
    echo -e "${GREEN}[OK] Docker successfully installed.${NC}"
fi

# 5. Initialize Installation Directory
echo -e "${BLUE}[INFO] Setting up installation in ${INSTALL_DIR}...${NC}"

REPO_URL="https://github.com/AmirRnz/xui-sells-v2.git"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" 2>/dev/null && pwd || echo "")"
if [[ -d "${INSTALL_DIR}/.git" ]]; then
    echo -e "${BLUE}[INFO] Updating existing installation in ${INSTALL_DIR}...${NC}"
    git -C "${INSTALL_DIR}" pull --ff-only || true
elif [[ -n "${SCRIPT_DIR}" && -f "${SCRIPT_DIR}/docker-compose.yml" && "${SCRIPT_DIR}" != "${INSTALL_DIR}/scripts" ]]; then
    mkdir -p "${INSTALL_DIR}"
    cp -r "${SCRIPT_DIR}/../"* "${INSTALL_DIR}/"
else
    echo -e "${BLUE}[INFO] Fetching system files from ${REPO_URL}...${NC}"
    if ! command -v git &> /dev/null; then
        apt-get update -y && apt-get install -y git
    fi
    TMP_CLONE=$(mktemp -d)
    git clone --depth 1 "${REPO_URL}" "${TMP_CLONE}"
    mkdir -p "${INSTALL_DIR}"
    cp -a "${TMP_CLONE}/." "${INSTALL_DIR}/"
    rm -rf "${TMP_CLONE}"
fi

mkdir -p "${INSTALL_DIR}/data" "${INSTALL_DIR}/scripts"

# Create environment configuration
if [[ ! -f "${INSTALL_DIR}/.env" ]]; then
    POSTGRES_PASS=$(tr -dc A-Za-z0-9 </dev/urandom | head -c 24 || true)
    cat <<EOF > "${INSTALL_DIR}/.env"
WEB_PORT=${WEB_PORT}
POSTGRES_USER=xui_user
POSTGRES_PASSWORD=${POSTGRES_PASS}
POSTGRES_DB=xui_sells
APP_ENV=production
DATA_DIR=${INSTALL_DIR}/data
EOF
    chmod 600 "${INSTALL_DIR}/.env"
fi

# 6. Create Host CLI Wrapper (xui-sells command)
echo -e "${BLUE}[INFO] Creating global CLI entrypoint at ${BINARY_LINK}...${NC}"
cat <<'EOF' > "${BINARY_LINK}"
#!/usr/bin/env bash
INSTALL_DIR="/opt/xui-sells"
if [[ -t 0 ]]; then
    # Interactive mode (terminal attached)
    docker compose -f "${INSTALL_DIR}/scripts/docker-compose.yml" --project-directory "${INSTALL_DIR}" exec -it app /usr/local/bin/xui-sells "$@"
else
    # Non-interactive / headless mode
    docker compose -f "${INSTALL_DIR}/scripts/docker-compose.yml" --project-directory "${INSTALL_DIR}" exec -T app /usr/local/bin/xui-sells "$@"
fi
EOF
chmod +x "${BINARY_LINK}"

# 7. Start System Containers
echo -e "${BLUE}[INFO] Starting system containers...${NC}"
cd "${INSTALL_DIR}"
docker compose -f scripts/docker-compose.yml up -d --build

# 8. Success Banner & Instructions
echo -e "${GREEN}${BOLD}"
echo "====================================================================="
echo "   INSTALLATION COMPLETED SUCCESSFULLY!                              "
echo "====================================================================="
echo -e "${NC}"
echo -e "You can now manage the system using the CLI command: ${BOLD}xui-sells${NC}"
echo -e "To open the interactive instance management wizard, simply run:"
echo -e "  ${CYAN}${BOLD}xui-sells menu${NC}"
echo ""
echo -e "Web Panel access is available on: ${BOLD}http://<YOUR_SERVER_IP>:${WEB_PORT}${NC}"
echo "====================================================================="

