# WarpTail Installation Guide

This comprehensive guide will help you install and configure WarpTail on a Server running Ubuntu or Debian.

## Prerequisites

- Ubuntu 20.04+ or Debian 11+
- Root or sudo access to the server
- A Tailscale account with an auth key
- Domain name pointed to your VPS (optional but recommended)

## Quick Installation Script

### 🚀 One-Line Install

Install WarpTail on your VPS with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/robrotheram/warptail/main/scripts/install-vps.sh | sudo bash
```

**What this does:**
- Installs WarpTail binary to `/usr/local/bin`
- Creates system user and directories
- Sets up systemd service for auto-start
- Configures logging at `/var/log/warptail`

**Upgrade:**
```bash
curl -fsSL https://raw.githubusercontent.com/robrotheram/warptail/main/scripts/install-vps.sh | sudo bash -s -- --upgrade
```

> **Note:** Requires Ubuntu or Debian with sudo privileges.

For manual installation steps, see [INSTALL.md](./docs/INSTALL.md).

---

### Important files

#### Configuration
  - `/etc/warptail/config`

#### Logs
  - Server log `/var/log/warptail/warptail.log` 
  - Proxy access log `/var/log/warptail/access.log`
  - Proxy error log `/var/log/warptail/error.log`

Quick commands:
```bash
# View config
sudo less /etc/warptail/config

# Follow logs
sudo tail -f /var/log/warptail/access.log /var/log/warptail/error.log /var/log/warptail/warptail.log
```

#### Access the Dashboard

Once WarpTail is running:

1. Open your web browser and navigate to `http://YOUR_VPS_IP:80`
2. Log in with the credentials you set in the config file
3. Configure your first service through the web interface


## Authentication Configuration

WarpTail supports multiple authentication methods. The authentication provider is optional - you can disable authentication entirely or use various OpenID providers.

### Authentication Options

#### Option 1: Password Authentication (Default)

```yaml
authentication:
  baseURL: "https://your-domain.com"  # Your WarpTail URL
  secretKey: "your-random-secret-key-here"
  provider:
      basic:
          email: admin@warptail.local # Change to your email address
```

#### Option 2: Google OAuth

```yaml
authentication:
  baseURL: "https://your-domain.com"
  secretKey: "your-random-secret-key-here"
  provider:
    oidc:
      name: "Google SSO"
      issuer_url: "https://accounts.google.com"
      client_id: "your-google-client-id.apps.googleusercontent.com"
      client_secret: "your-client-secret"
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
authentication:
  baseURL: "https://your-domain.com"
  secretKey: "your-random-secret-key-here"
  provider:
    oidc:
      name: "Microsoft SSO"
      client_id: "your-azure-application-id"
      client_secret: "your-client-secret"
      issuer_url: "https://login.microsoftonline.com/your-tenant-id/v2.0"
      
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
authentication:
  baseURL: "https://your-domain.com"
  secretKey: "your-random-secret-key-here"
  provider:
    oidc:
      name: "GitHub SSO"
      type: "openid"
      client_id: "your-github-client-id"
      client_secret: "your-client-secret"
      issuer_url: "https://github.com"
```

**Setup Steps for GitHub OAuth:**

1. Go to GitHub Settings → Developer settings → OAuth Apps
2. Click "New OAuth App"
3. Set Authorization callback URL to `https://your-domain.com/auth/callback`
4. Copy the Client ID

#### Option 5: Generic OpenID Connect

```yaml
authentication:
  baseURL: "https://your-domain.com"
  secretKey: "your-random-secret-key-here"
  provider:
    oidc:
      name: "Custom OIDC"
      type: "openid"
      client_id: "your-client-id"
      client_secret: "your-client-secret"
      issuer_url: "https://your-oidc-provider.com"
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
```

#### SSL Certificates for Standalone HTTPS

WarpTail has its own internal SSL implementation and can automatically handle HTTPS certificates. You have several options:

WarpTail can automatically obtain and renew SSL certificates using ACME/Let's Encrypt:

```yaml
acme:
    enabled: true
    ssl_port: 443
    certificates_dir: "/tmp/warptail-certs"
    portal_domain: "test.your-domain.com" # Domain for the web insterface. Optional
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