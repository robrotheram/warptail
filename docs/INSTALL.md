# WarpTail Installation Guide

This comprehensive guide will help you install and configure WarpTail on a Server running Ubuntu or Debian.

## Prerequisites

- Ubuntu 20.04+ or Debian 11+
- Root or sudo access to the server
- A Tailscale account with an auth key
- Domain name pointed to your VPS (optional but recommended)

## Quick Installation Script

For a quick installation, you can use this script:

```bash
curl -fsSL https://raw.githubusercontent.com/robrotheram/warptail/main/scripts/install-vps.sh | bash
```

## Manual Installation

### 1. System Preparation

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install dependencies
sudo apt install -y curl wget git build-essential unzip

# Install Go 1.21+
GO_VERSION="1.21.5"
wget https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/profile
source /etc/profile
```

### 2. Create System User

```bash
# Create warptail system user
sudo useradd --system --shell /bin/false --home-dir /var/lib/warptail --create-home warptail

# Create directories
sudo mkdir -p /etc/warptail /var/log/warptail /var/lib/warptail
sudo chown -R warptail:warptail /etc/warptail /var/log/warptail /var/lib/warptail
sudo chmod 755 /etc/warptail /var/log/warptail /var/lib/warptail
```

### 3. Install WarpTail

#### Option A: Download Binary (Recommended)

```bash
# Get latest version
LATEST_VERSION=$(curl -s https://api.github.com/repos/robrotheram/warptail/releases/latest | grep tag_name | cut -d '"' -f 4)
wget "https://github.com/robrotheram/warptail/releases/download/${LATEST_VERSION}/warptail-linux-amd64"

# Install binary
sudo mv warptail-linux-amd64 /usr/local/bin/warptail
sudo chmod +x /usr/local/bin/warptail
sudo chown root:root /usr/local/bin/warptail
```

#### Option B: Build from Source

```bash
# Clone repository
git clone https://github.com/robrotheram/warptail.git
cd warptail

# Build dashboard (requires Node.js)
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs
cd dashboard && npm install && npm run build && cd ..

# Build Go binary
go build -ldflags="-s -w" -o warptail .
sudo mv warptail /usr/local/bin/
sudo chmod +x /usr/local/bin/warptail
```

### 4. Configuration

#### Create Main Config File

```bash
sudo tee /etc/warptail/config.yaml > /dev/null <<'EOF'
tailscale:
  auth_key: "YOUR_TAILSCALE_AUTH_KEY"
  hostname: "warptail-proxy"

application:
  host: "0.0.0.0"
  port: 8080
  authentication:
    baseURL: "http://localhost:8080"
    secretKey: "GENERATE_RANDOM_SECRET_HERE"
   #  provider:
   #    name: "WarpTail Admin"
   #    type: "password"
   #    session_secret: "change-this-secure-password"

logging:
  format: "json"
  level: "info"
  output: "file"
  path: "/var/log/warptail"

database:
  path: "/var/lib/warptail/warptail.db"

services: []
EOF

sudo chown warptail:warptail /etc/warptail/config.yaml
sudo chmod 600 /etc/warptail/config.yaml
```

#### Environment File (Optional)

```bash
sudo tee /etc/warptail/environment > /dev/null <<'EOF'
CONFIG_PATH=/etc/warptail/config.yaml
WARPTAIL_LOG_LEVEL=info
EOF

sudo chown warptail:warptail /etc/warptail/environment
sudo chmod 644 /etc/warptail/environment
```

### 5. Systemd Service

```bash
sudo tee /etc/systemd/system/warptail.service > /dev/null <<'EOF'
[Unit]
Description=WarpTail Tailscale Proxy
Documentation=https://github.com/robrotheram/warptail
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=warptail
Group=warptail
ExecStart=/usr/local/bin/warptail
WorkingDirectory=/var/lib/warptail
EnvironmentFile=-/etc/warptail/environment
Environment=CONFIG_PATH=/etc/warptail/config.yaml

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
ReadWritePaths=/var/lib/warptail /var/log/warptail
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable warptail
```

### 6. Firewall Configuration

#### UFW (Ubuntu Firewall)

```bash
# Allow SSH (if not already configured)
sudo ufw allow ssh

# Allow WarpTail dashboard
sudo ufw allow 8080/tcp comment 'WarpTail Dashboard'

# Allow HTTP/HTTPS for proxied services
sudo ufw allow 80/tcp comment 'HTTP'
sudo ufw allow 443/tcp comment 'HTTPS'

# Enable firewall
sudo ufw --force enable
sudo ufw status verbose
```

#### iptables (Alternative)

```bash
# Allow WarpTail dashboard
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT

# Allow HTTP/HTTPS
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT

# Save rules (Ubuntu/Debian)
sudo iptables-save | sudo tee /etc/iptables/rules.v4
```

### 7. Start and Verify

```bash
# Start WarpTail
sudo systemctl start warptail

# Check status
sudo systemctl status warptail

# View logs
sudo journalctl -u warptail -f --no-pager

# Check if it's listening
sudo ss -tlnp | grep :8080
```

### 8. Initial Setup

1. **Get your Tailscale auth key**:
   - Go to [Tailscale Admin Console](https://login.tailscale.com/admin/settings/keys)
   - Generate a new auth key (reusable, no expiry recommended)

2. **Update configuration**:
   ```bash
   sudo nano /etc/warptail/config.yaml
   # Replace YOUR_TAILSCALE_AUTH_KEY with your actual key
   # Generate a random secret: openssl rand -base64 32
   # Change the default password
   
   # Restart after changes
   sudo systemctl restart warptail
   ```

3. **Access dashboard**:
   - Open `http://YOUR_VPS_IP:8080` in your browser
   - Login with your configured credentials

## Authentication Configuration

WarpTail supports multiple authentication methods. The authentication provider is optional - you can disable authentication entirely or use various OpenID providers.

### Authentication Options

#### Option 1: Password Authentication (Default)

```yaml
application:
  authentication:
    baseURL: "https://your-domain.com"  # Your WarpTail URL
    secretKey: "your-random-secret-key-here"
    provider:
      name: "WarpTail Admin"
      type: "password"
      session_secret: "your-secure-password"
```

#### Option 2: Google OAuth

```yaml
application:
  authentication:
    baseURL: "https://your-domain.com"
    secretKey: "your-random-secret-key-here"
    provider:
      name: "Google SSO"
      type: "openid"
      clientID: "your-google-client-id.apps.googleusercontent.com"
      providerURL: "https://accounts.google.com"
      session_secret: "your-session-secret"
```

**Setup Steps for Google OAuth:**

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the "Google+ API"
4. Go to "Credentials" → "Create Credentials" → "OAuth 2.0 Client IDs"
5. Set application type to "Web application"
6. Add authorized redirect URIs:
   - `https://your-domain.com/auth/callback`
7. Copy the Client ID to your config

#### Option 3: Microsoft Azure AD

```yaml
application:
  authentication:
    baseURL: "https://your-domain.com"
    secretKey: "your-random-secret-key-here"
    provider:
      name: "Microsoft SSO"
      type: "openid"
      clientID: "your-azure-application-id"
      providerURL: "https://login.microsoftonline.com/your-tenant-id/v2.0"
      session_secret: "your-session-secret"
```

**Setup Steps for Azure AD:**

1. Go to [Azure Portal](https://portal.azure.com/)
2. Navigate to "Azure Active Directory" → "App registrations"
3. Click "New registration"
4. Set redirect URI to `https://your-domain.com/auth/callback`
5. Copy the Application (client) ID and Directory (tenant) ID
6. Update the providerURL with your tenant ID

#### Option 4: GitHub OAuth

```yaml
application:
  authentication:
    baseURL: "https://your-domain.com"
    secretKey: "your-random-secret-key-here"
    provider:
      name: "GitHub SSO"
      type: "openid"
      clientID: "your-github-client-id"
      providerURL: "https://github.com"
      session_secret: "your-session-secret"
```

**Setup Steps for GitHub OAuth:**

1. Go to GitHub Settings → Developer settings → OAuth Apps
2. Click "New OAuth App"
3. Set Authorization callback URL to `https://your-domain.com/auth/callback`
4. Copy the Client ID

#### Option 5: Generic OpenID Connect

```yaml
application:
  authentication:
    baseURL: "https://your-domain.com"
    secretKey: "your-random-secret-key-here"
    provider:
      name: "Custom OIDC"
      type: "openid"
      clientID: "your-client-id"
      providerURL: "https://your-oidc-provider.com"
      session_secret: "your-session-secret"
```

**For other OIDC providers (Keycloak, Auth0, Okta, etc.):**

1. Create a new application in your OIDC provider
2. Set the redirect URI to `https://your-domain.com/auth/callback`
3. Use the provider's discovery URL or base URL
4. Copy the client ID

### Authentication Configuration Notes

- **baseURL**: Must match your actual WarpTail URL (important for OAuth callbacks)
- **secretKey**: Generate with `openssl rand -base64 32` - used for session encryption
- **session_secret**: 
  - For `password` type: This is your login password
  - For `openid` type: This is your OAuth client secret
- **providerURL**: The OIDC provider's base URL or discovery endpoint

For more detailed authentication setup including troubleshooting, see the [Authentication Configuration Guide](authentication.md).

## Production Setup

WarpTail can run in two modes for production:

1. **Standalone Mode**: WarpTail handles all traffic directly (simpler setup)
2. **Reverse Proxy Mode**: WarpTail behind Nginx/Caddy/Traefik (more features)

### Standalone Mode (No Nginx)

For simple setups, you can run WarpTail directly without a reverse proxy:

#### Configuration for Standalone

```yaml
application:
  host: "0.0.0.0"
  port: 80  # HTTP traffic
  # port: 443  # HTTPS traffic (requires SSL certificates)
```

#### Enable HTTP/HTTPS Ports

```bash
# Allow WarpTail to bind to privileged ports (80/443)
sudo setcap 'cap_net_bind_service=+ep' /usr/local/bin/warptail

# Update firewall
sudo ufw delete allow 8080/tcp  # Remove dashboard port if not needed
sudo ufw allow 80/tcp comment 'WarpTail HTTP'
sudo ufw allow 443/tcp comment 'WarpTail HTTPS'
```

#### Systemd Service for Standalone

Update the systemd service to allow binding to privileged ports:

```bash
sudo tee /etc/systemd/system/warptail.service > /dev/null <<'EOF'
[Unit]
Description=WarpTail Tailscale Proxy
Documentation=https://github.com/robrotheram/warptail
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=warptail
Group=warptail
ExecStart=/usr/local/bin/warptail
WorkingDirectory=/var/lib/warptail
Environment=CONFIG_PATH=/etc/warptail/config.yaml

# Restart policy
Restart=always
RestartSec=5
StartLimitInterval=60s
StartLimitBurst=3

# Output to journald
StandardOutput=journal
StandardError=journal
SyslogIdentifier=warptail

# Security (with network capabilities)
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/warptail /var/log/warptail
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl restart warptail
```

#### SSL Certificates for Standalone HTTPS

WarpTail has its own internal SSL implementation and can automatically handle HTTPS certificates. You have several options:

WarpTail can automatically obtain and renew SSL certificates using ACME/Let's Encrypt:

```yaml
# Add to your config.yaml
application:
  host: "0.0.0.0"
  port: 443
  ssl:
    enabled: true
    auto_cert: true
    domains:
      - "yourdomain.com"
      - "*.yourdomain.com"  # Optional: wildcard support
    email: "your-email@example.com"  # Required for Let's Encrypt
```


### Monitoring and Logging

#### Log Rotation

```bash
sudo tee /etc/logrotate.d/warptail <<'EOF'
/var/log/warptail/*.log {
    daily
    missingok
    rotate 52
    compress
    delaycompress
    notifempty
    create 644 warptail warptail
    postrotate
        systemctl reload warptail
    endscript
}
EOF
```

#### Prometheus Monitoring

WarpTail exposes metrics at `/metrics`. Configure your Prometheus to scrape:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'warptail'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```


## Troubleshooting

### Common Issues

1. **WarpTail won't start**:
   ```bash
   # Check logs
   sudo journalctl -u warptail -n 50 --no-pager
   
   # Verify config
   sudo -u warptail /usr/local/bin/warptail --config /etc/warptail/config.yaml --dry-run
   ```

2. **Can't access dashboard**:
   ```bash
   # Check if service is running
   sudo systemctl status warptail
   
   # Check if port is open
   sudo ss -tlnp | grep :8080
   
   # Check firewall
   sudo ufw status
   ```

3. **Tailscale connection issues**:
   ```bash
   # Check Tailscale status
   sudo tailscale status
   
   # Re-authenticate if needed
   sudo tailscale up --authkey=YOUR_NEW_AUTH_KEY
   ```

4. **Services not accessible**:
   ```bash
   # Check access logs
   sudo tail -f /var/log/warptail/access.log
   
   # Check error logs
   sudo tail -f /var/log/warptail/error.log
   
   # Test connectivity from WarpTail to target service
   sudo -u warptail curl -v http://target-ip:port
   ```

### Performance Tuning

```bash
# Increase file limits for warptail user
sudo tee -a /etc/security/limits.conf <<EOF
warptail soft nofile 65536
warptail hard nofile 65536
EOF

# Optimize kernel parameters
sudo tee -a /etc/sysctl.conf <<EOF
# Network optimizations for proxy
net.core.somaxconn = 65536
net.core.netdev_max_backlog = 5000
net.ipv4.tcp_max_syn_backlog = 65536
EOF

sudo sysctl -p
```

### Security Hardening

```bash
# Restrict config file access
sudo chmod 600 /etc/warptail/config.yaml
sudo chown root:warptail /etc/warptail/config.yaml

# Set up fail2ban for repeated login failures
sudo apt install -y fail2ban

sudo tee /etc/fail2ban/jail.d/warptail.conf <<'EOF'
[warptail]
enabled = true
port = 8080
filter = warptail
logpath = /var/log/warptail/error.log
maxretry = 5
bantime = 3600
EOF

sudo tee /etc/fail2ban/filter.d/warptail.conf <<'EOF'
[Definition]
failregex = .*authentication failed.*client: <HOST>
ignoreregex =
EOF

sudo systemctl restart fail2ban
```

## Maintenance

### Regular Tasks

1. **Update WarpTail**:
   ```bash
   sudo systemctl stop warptail
   # Download new version
   sudo systemctl start warptail
   ```

2. **Monitor logs**:
   ```bash
   # Check for errors
   sudo grep -i error /var/log/warptail/error.log
   
   # Monitor access patterns
   sudo tail -f /var/log/warptail/access.log
   ```

3. **Database maintenance**:
   ```bash
   # Backup before maintenance
   sudo -u warptail cp /var/lib/warptail/warptail.db /var/lib/warptail/warptail.db.backup
   
   # SQLite vacuum (compact database)
   sudo -u warptail sqlite3 /var/lib/warptail/warptail.db "VACUUM;"
   ```

For more advanced configuration options, see the [Advanced Proxy Configuration Guide](./docs/advanced-proxy.md).