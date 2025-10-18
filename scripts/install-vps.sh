#!/bin/bash

# WarpTail VPS Installation Script
# This script installs WarpTail on Ubuntu/Debian systems

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
WARPTAIL_USER="warptail"
WARPTAIL_HOME="/var/lib/warptail"
CONFIG_DIR="/etc/warptail"
LOG_DIR="/var/log/warptail"
INSTALL_DIR="/usr/local/bin"

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [[ $EUID -eq 0 ]]; then
        print_error "This script should not be run as root. Please run as a regular user with sudo privileges."
        exit 1
    fi
}

check_sudo() {
    if ! sudo -n true 2>/dev/null; then
        print_error "This script requires sudo privileges. Please ensure you can run sudo commands."
        exit 1
    fi
}

check_os() {
    if [[ ! -f /etc/os-release ]]; then
        print_error "Cannot determine OS. This script supports Ubuntu and Debian."
        exit 1
    fi
    
    . /etc/os-release
    if [[ "$ID" != "ubuntu" && "$ID" != "debian" ]]; then
        print_error "This script only supports Ubuntu and Debian. Detected: $ID"
        exit 1
    fi
    
    print_success "Detected OS: $PRETTY_NAME"
}


create_user() {
    if id "$WARPTAIL_USER" &>/dev/null; then
        print_warning "User $WARPTAIL_USER already exists"
    else
        print_status "Creating system user: $WARPTAIL_USER"
        sudo useradd --system --shell /bin/false --home-dir "$WARPTAIL_HOME" --create-home "$WARPTAIL_USER"
    fi
    
    print_status "Creating directories..."
    sudo mkdir -p "$CONFIG_DIR" "$LOG_DIR" "$WARPTAIL_HOME"
    sudo chown -R "$WARPTAIL_USER:$WARPTAIL_USER" "$CONFIG_DIR" "$LOG_DIR" "$WARPTAIL_HOME"
    sudo chmod 755 "$CONFIG_DIR" "$LOG_DIR" "$WARPTAIL_HOME"
}

install_warptail() {
    print_status "Installing WarpTail binary..."
    
    # Get latest release
    LATEST_VERSION=$(curl -s https://api.github.com/repos/robrotheram/warptail/releases/latest | grep -o '"tag_name": "[^"]*"' | sed 's/"tag_name": "//;s/"//')
    
    if [[ -z "$LATEST_VERSION" ]]; then
        print_error "Failed to get latest version from GitHub"
        exit 1
    fi
    
    print_status "Downloading WarpTail version: $LATEST_VERSION"
    
    # Download binary
    DOWNLOAD_URL="https://github.com/robrotheram/warptail/releases/download/${LATEST_VERSION}/warptail-amd64"
    
    if ! wget -q "$DOWNLOAD_URL" -O warptail-linux-amd64; then
        print_error "Failed to download WarpTail binary"
        exit 1
    fi
    
    # Install binary
    sudo mv warptail-linux-amd64 "$INSTALL_DIR/warptail"
    sudo chmod +x "$INSTALL_DIR/warptail"
    sudo chown root:root "$INSTALL_DIR/warptail"
    
    print_success "WarpTail binary installed successfully"
}

create_config() {
    print_status "Creating configuration file..."
    
    # Generate a random password
    RANDOM_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
    
    sudo tee "$CONFIG_DIR/config.yaml" > /dev/null <<EOF
tailscale:
  auth_key: "YOUR_TAILSCALE_AUTH_KEY"
  hostname: "warptail-proxy"

application:
  host: "0.0.0.0"
  port: 8080
  authentication:
    baseURL: "http://localhost:8080"
    session_secret: "$(openssl rand -base64 32)"
    provider:
      basic:
        email: "admin@warptail.local"
        password: "$RANDOM_PASSWORD"    
logging:
  format: "json"
  level: "info"
  output: "file"
  path: "$LOG_DIR"

database:
  connection_type: sqlite
  connection: file:$WARPTAIL_HOME/warptail.db?cache=shared

services: []
EOF

    sudo chown "$WARPTAIL_USER:$WARPTAIL_USER" "$CONFIG_DIR/config.yaml"
    sudo chmod 600 "$CONFIG_DIR/config.yaml"
    
    print_success "Configuration file created"
    print_warning "Generated admin password: $RANDOM_PASSWORD"
    print_warning "Please save this password and update the Tailscale auth key in $CONFIG_DIR/config.yaml"
}

create_systemd_service() {
    print_status "Creating systemd service..."
    
    sudo tee /etc/systemd/system/warptail.service > /dev/null <<EOF
[Unit]
Description=WarpTail Tailscale Proxy
Documentation=https://github.com/robrotheram/warptail
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$WARPTAIL_USER
Group=$WARPTAIL_USER
ExecStart=$INSTALL_DIR/warptail
WorkingDirectory=$WARPTAIL_HOME
Environment=CONFIG_PATH=$CONFIG_DIR/config.yaml

# Restart policy
Restart=always
RestartSec=5
StartLimitInterval=60s
StartLimitBurst=3

# Output to journald
StandardOutput=journal
StandardError=journal
SyslogIdentifier=warptail

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$WARPTAIL_HOME $LOG_DIR
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
EOF

    sudo systemctl daemon-reload
    sudo systemctl enable warptail
    
    print_success "Systemd service created and enabled"
}

configure_firewall() {
    print_status "Configuring firewall..."
    
    if command -v ufw >/dev/null 2>&1; then
        sudo ufw allow 8080/tcp comment 'WarpTail Dashboard'
        sudo ufw allow 80/tcp comment 'HTTP'
        sudo ufw allow 443/tcp comment 'HTTPS'
        
        if ! sudo ufw status | grep -q "Status: active"; then
            print_warning "UFW is not active. To enable it, run: sudo ufw enable"
        fi
    else
        print_warning "UFW not found. Please manually configure firewall to allow ports 8080, 80, and 443"
    fi
}

setup_log_rotation() {
    print_status "Setting up log rotation..."
    
    sudo tee /etc/logrotate.d/warptail > /dev/null <<EOF
$LOG_DIR/*.log {
    daily
    missingok
    rotate 52
    compress
    delaycompress
    notifempty
    create 644 $WARPTAIL_USER $WARPTAIL_USER
    postrotate
        systemctl reload warptail
    endscript
}
EOF
    
    print_success "Log rotation configured"
}

print_summary() {
    echo
    echo "=================================="
    echo "  WarpTail Installation Complete!"
    echo "=================================="
    echo
    print_success "WarpTail has been successfully installed!"
    echo
    echo "Next steps:"
    echo "1. Edit the configuration file: sudo nano $CONFIG_DIR/config.yaml"
    echo "2. Add your Tailscale auth key (replace YOUR_TAILSCALE_AUTH_KEY)"
    echo "3. Start WarpTail: sudo systemctl start warptail"
    echo "4. Check status: sudo systemctl status warptail"
    echo "5. Access dashboard: http://$(curl -s ifconfig.me):8080"
    echo
    echo "Login credentials:"
    echo "  Username: admin"
    echo "  Password: (check the generated password above)"
    echo
    echo "Useful commands:"
    echo "  View logs: sudo journalctl -u warptail -f"
    echo "  Access logs: sudo tail -f $LOG_DIR/access.log"
    echo "  Error logs: sudo tail -f $LOG_DIR/error.log"
    echo
    echo "Documentation: https://github.com/robrotheram/warptail"
    echo
}

main() {
    echo "======================================="
    echo "  WarpTail VPS Installation Script"
    echo "======================================="
    echo
    
    check_root
    check_sudo
    check_os
    
    print_status "Starting WarpTail installation..."
    
    create_user
    install_warptail
    create_config
    create_systemd_service
    configure_firewall
    setup_log_rotation
    
    print_summary
}

# Run main function
main "$@"