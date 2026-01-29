#!/bin/bash

# WarpTail VPS Installation/Upgrade Script
# This script installs or upgrades WarpTail on Ubuntu/Debian systems

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
LOG_DIR="/var/log/warptail"
INSTALL_DIR="/usr/local/bin"
CONFIG_PATH="/etc/warptail"

# Operation mode
OPERATION_MODE="install"

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

check_sudo_user() {
    # Check if running with sudo
    if [[ $EUID -eq 0 ]]; then
        # Running as root, check if it was invoked via sudo
        if [[ -z "$SUDO_USER" ]]; then
            print_error "This script should not be run directly as root. Please run with: sudo ./install.sh"
            exit 1
        fi
        # Running with sudo - this is correct
        echo "Running with sudo as user: $SUDO_USER"
    else
        # Not running as root, check if user has sudo privileges
        if ! sudo -n true 2>/dev/null; then
            print_error "This script requires sudo privileges. Please run with: sudo ./install.sh"
            exit 1
        fi
        print_error "Please run this script with sudo: sudo ./install.sh"
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

show_help() {
    echo "WarpTail VPS Installation/Upgrade Script"
    echo
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  -u, --upgrade  Upgrade existing WarpTail installation"
    echo "  -i, --install  Install WarpTail (default)"
    echo
    echo "Examples:"
    echo "  $0                 # Install WarpTail"
    echo "  $0 --install       # Install WarpTail"
    echo "  $0 --upgrade       # Upgrade WarpTail"
    echo
}

check_existing_installation() {
    if [[ -f "$INSTALL_DIR/warptail" ]] && systemctl list-unit-files | grep -q "warptail.service"; then
        return 0  # Installation exists
    else
        return 1  # No installation found
    fi
}

get_current_version() {
    if [[ -f "$INSTALL_DIR/warptail" ]]; then
        local current_version=$("$INSTALL_DIR/warptail" --version 2>/dev/null | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' || echo "unknown")
        echo "$current_version"
    else
        echo "not installed"
    fi
}


create_user() {
    if id "$WARPTAIL_USER" &>/dev/null; then
        print_warning "User $WARPTAIL_USER already exists"
    else
        print_status "Creating system user: $WARPTAIL_USER"
        sudo useradd --system --shell /bin/false --home-dir "$WARPTAIL_HOME" --create-home "$WARPTAIL_USER"
    fi
    
    print_status "Creating directories..."
    sudo mkdir -p "$LOG_DIR" "$WARPTAIL_HOME" "$CONFIG_PATH"
    sudo chown -R "$WARPTAIL_USER:$WARPTAIL_USER" "$LOG_DIR" "$WARPTAIL_HOME" "$CONFIG_PATH"
    sudo chmod 755 "$LOG_DIR" "$WARPTAIL_HOME" "$CONFIG_PATH"
}

install_warptail() {
    local is_upgrade="$1"
    local service_was_running=false
    
    if [[ "$is_upgrade" == "true" ]]; then
        print_status "Upgrading WarpTail binary..."
        
        # Check if service is running
        if systemctl is-active --quiet warptail; then
            service_was_running=true
            print_status "Stopping WarpTail service..."
            sudo systemctl stop warptail
        fi
        
        # Backup current binary
        if [[ -f "$INSTALL_DIR/warptail" ]]; then
            sudo cp "$INSTALL_DIR/warptail" "$INSTALL_DIR/warptail.backup"
            print_status "Current binary backed up to warptail.backup"
        fi
    else
        print_status "Installing WarpTail binary..."
    fi
    
    # Get current and latest versions
    local current_version=$(get_current_version)
    LATEST_VERSION=$(curl -s https://api.github.com/repos/robrotheram/warptail/releases/latest | grep -o '"tag_name": "[^"]*"' | sed 's/"tag_name": "//;s/"//')
    
    if [[ -z "$LATEST_VERSION" ]]; then
        print_error "Failed to get latest version from GitHub"
        exit 1
    fi
    
    if [[ "$is_upgrade" == "true" && "$current_version" != "not installed" && "$current_version" != "unknown" ]]; then
        print_status "Current version: $current_version"
        print_status "Latest version: $LATEST_VERSION"
        
        if [[ "$current_version" == "$LATEST_VERSION" ]]; then
            print_warning "Already running the latest version ($LATEST_VERSION)"
            if [[ "$service_was_running" == "true" ]]; then
                print_status "Starting WarpTail service..."
                sudo systemctl start warptail
            fi
            return 0
        fi
    fi
    
    print_status "Downloading WarpTail version: $LATEST_VERSION"
    
    # Download binary
    DOWNLOAD_URL="https://github.com/robrotheram/warptail/releases/download/${LATEST_VERSION}/warptail-amd64"
    
    if ! wget -q "$DOWNLOAD_URL" -O warptail-linux-amd64; then
        print_error "Failed to download WarpTail binary"
        if [[ "$is_upgrade" == "true" && -f "$INSTALL_DIR/warptail.backup" ]]; then
            print_status "Restoring backup..."
            sudo mv "$INSTALL_DIR/warptail.backup" "$INSTALL_DIR/warptail"
            if [[ "$service_was_running" == "true" ]]; then
                sudo systemctl start warptail
            fi
        fi
        exit 1
    fi
    
    # Install binary
    sudo mv warptail-linux-amd64 "$INSTALL_DIR/warptail"
    sudo chmod +x "$INSTALL_DIR/warptail"
    sudo chown root:root "$INSTALL_DIR/warptail"
    
    # Remove backup if upgrade was successful
    if [[ "$is_upgrade" == "true" && -f "$INSTALL_DIR/warptail.backup" ]]; then
        sudo rm -f "$INSTALL_DIR/warptail.backup"
    fi
    
    if [[ "$is_upgrade" == "true" ]]; then
        print_success "WarpTail upgraded successfully to version: $LATEST_VERSION"
        
        # Restart service if it was running
        if [[ "$service_was_running" == "true" ]]; then
            print_status "Starting WarpTail service..."
            sudo systemctl daemon-reload
            sudo systemctl start warptail
            
            # Verify service started successfully
            sleep 2
            if systemctl is-active --quiet warptail; then
                print_success "WarpTail service restarted successfully"
            else
                print_error "Failed to restart WarpTail service"
                print_status "Check service status: sudo systemctl status warptail"
                print_status "Check logs: sudo journalctl -u warptail -f"
            fi
        else
            print_status "Service was not running. Use 'sudo systemctl start warptail' to start it."
        fi
    else
        print_success "WarpTail binary installed successfully"
    fi
}

create_config() {
    print_status "Creating configuration file..."
    
    # Check if config file already exists
    if [[ -f "$WARPTAIL_HOME/config.yaml" ]]; then
        print_warning "Configuration file already exists: $WARPTAIL_HOME/config.yaml"
        print_status "Keeping existing configuration file"
        return 0
    fi
    
    # Get server's public IPv4 address
    IPV4_ADDRESS=$(curl -s https://ifconfig.me)
    if [[ -z "$IPV4_ADDRESS" ]]; then
        print_error "Failed to retrieve public IPv4 address"
        IPV4_ADDRESS="localhost"
    fi

    
    sudo tee "$WARPTAIL_HOME/config.yaml" > /dev/null <<EOF
tailscale:
  auth_key: ""
  hostname: "warptail-proxy"

application:
    host: "0.0.0.0"
    port: 80

authentication:
    baseURL: "http://$IPV4_ADDRESS"
    session_secret: "$(openssl rand -base64 32)"
    provider:
        basic:
            email: admin@warptail.local
        
acme:
    enabled: true
    ssl_port: 443
    certificates_dir: "$WARPTAIL_HOME/certs"
    portal_domain: "your-domain.com"

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

    sudo chown "$WARPTAIL_USER:$WARPTAIL_USER" "$WARPTAIL_HOME/config.yaml"
    sudo chmod 600 "$WARPTAIL_HOME/config.yaml"
    sudo ln -s "$WARPTAIL_HOME/config.yaml" "$CONFIG_PATH/config.yaml"
    
    print_success "Configuration file created"
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
Environment=CONFIG_PATH=$WARPTAIL_HOME/config.yaml

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
    sudo systemctl start warptail
    
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

print_install_summary() {
    echo
    echo "=================================="
    echo "  WarpTail Installation Complete!"
    echo "=================================="
    echo
    print_success "WarpTail has been successfully installed!"
    echo
    echo "Next steps:"
    echo "1. Edit the configuration file: sudo nano $WARPTAIL_HOME/config.yaml"
    echo "2. Add your Tailscale auth key (replace YOUR_TAILSCALE_AUTH_KEY)"
    echo "3. Start WarpTail: sudo systemctl start warptail"
    echo "4. Check status: sudo systemctl status warptail"
    echo "5. Access dashboard: http://$(curl -s ifconfig.me):8080"
    echo
    echo "Login credentials:"
    echo "  Username: admin"
    echo "  Password: changeme"
    echo
    echo "Useful commands:"
    echo "  View logs: sudo journalctl -u warptail -f"
    echo "  Access logs: sudo tail -f $LOG_DIR/access.log"
    echo "  Error logs: sudo tail -f $LOG_DIR/error.log"
    echo
    echo "Documentation: https://github.com/robrotheram/warptail"
    echo
}

print_upgrade_summary() {
    echo
    echo "================================="
    echo "  WarpTail Upgrade Complete!"
    echo "================================="
    echo
    print_success "WarpTail has been successfully upgraded!"
    echo
    local current_version=$(get_current_version)
    echo "Current version: $current_version"
    echo
    echo "Service status:"
    if systemctl is-active --quiet warptail; then
        print_success "WarpTail service is running"
    else
        print_warning "WarpTail service is not running"
        echo "To start: sudo systemctl start warptail"
    fi
    echo
    echo "Useful commands:"
    echo "  Check status: sudo systemctl status warptail"
    echo "  View logs: sudo journalctl -u warptail -f"
    echo "  Restart service: sudo systemctl restart warptail"
    echo
    echo "Documentation: https://github.com/robrotheram/warptail"
    echo
}

perform_upgrade() {
    print_status "Starting WarpTail upgrade..."
    
    if ! check_existing_installation; then
        print_error "No existing WarpTail installation found."
        print_status "Use --install option to install WarpTail for the first time."
        exit 1
    fi
    
    install_warptail true
    
    print_upgrade_summary
}

perform_install() {
    print_status "Starting WarpTail installation..."
    
    if check_existing_installation; then
        print_warning "WarpTail appears to be already installed."
        print_status "Use --upgrade option to upgrade the existing installation."
        read -p "Do you want to continue with installation anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_status "Installation cancelled."
            exit 0
        fi
    fi
    
    create_user
    install_warptail false
    create_config
    create_systemd_service
    configure_firewall
    setup_log_rotation
    
    print_install_summary
}

main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -u|--upgrade)
                OPERATION_MODE="upgrade"
                shift
                ;;
            -i|--install)
                OPERATION_MODE="install"
                shift
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    echo "======================================="
    if [[ "$OPERATION_MODE" == "upgrade" ]]; then
        echo "  WarpTail VPS Upgrade Script"
    else
        echo "  WarpTail VPS Installation Script"
    fi
    echo "======================================="
    echo
    
    check_sudo_user
    check_os
    
    if [[ "$OPERATION_MODE" == "upgrade" ]]; then
        perform_upgrade
    else
        perform_install
    fi
}

# Run main function
main "$@"