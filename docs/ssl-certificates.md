# ACME HTTPS Certificates

This document explains how to configure and use ACME (Automatic Certificate Management Environment) for automatic HTTPS certificate management in warptail.

## Overview

ACME is a protocol that allows for automated certificate issuance and renewal. warptail integrates with ACME providers like Let's Encrypt to automatically obtain and manage SSL/TLS certificates for your domains.

## Configuration

Add the following ACME configuration to your warptail configuration file:

```yaml
acme:
    enabled: true
    ssl_port: 443
    certificates_dir: "/etc/warptail/certs"
    portal_domain: "your-domain.com"
```

### Configuration Options

| Option | Type | Required | Default | Description |
|--------|------|----------|---------|-------------|
| `enabled` | boolean | Yes | `false` | Enable/disable ACME certificate management |
| `ssl_port` | integer | No | `443` | Port for HTTPS traffic |
| `certificates_dir` | string | No | `./certs` | Directory to store certificates |
| `portal_domain` | string | Yes | - | Primary domain for the portal |

## Prerequisites

1. **Domain Configuration**: Ensure your domain points to the server running warptail
2. **Port Access**: Ports 80 and 443 must be accessible from the internet
3. **File Permissions**: warptail must have write access to the certificates directory

## Setup Instructions

### 1. Configure Domain

Ensure your domain's DNS A record points to your server's public IP address:

```bash
# Verify DNS resolution
nslookup your-domain.com
```

### 2. Update Configuration

Edit your warptail configuration file:

```yaml

acme:
    enabled: true
    ssl_port: 443
    certificates_dir: "/etc/warptail/certs"
    portal_domain: "your-domain.com"
```

### 3. Create Certificates Directory

```bash
sudo mkdir -p /etc/warptail/certs
sudo chown -R warptail:warptail /etc/warptail/certs
sudo chmod 700 /etc/warptail/certs
```

### 4. Start warptail

```bash
sudo systemctl start warptail
```

## How It Works

1. **HTTP Challenge**: ACME uses HTTP-01 challenge by default
2. **Port 80 Redirect**: warptail automatically listens on port 80 for ACME challenges
3. **Certificate Storage**: Certificates are stored in the specified directory
4. **Auto-Renewal**: Certificates are automatically renewed before expiration

## Certificate Management

### Automatic Renewal

Certificates are automatically renewed when they're within 30 days of expiration. No manual intervention is required.

### Manual Certificate Check

```bash
# Check certificate expiration
openssl x509 -in /etc/warptail/certs/your-domain.com.crt -noout -dates

# View certificate details
openssl x509 -in /etc/warptail/certs/your-domain.com.crt -noout -text
```

## Troubleshooting

### Common Issues

#### 1. Port 80/443 Not Accessible

**Error**: `Failed to bind to port 80` or `Failed to bind to port 443`

**Solution**:
- Ensure no other services are using these ports
- Check firewall settings
- Verify the user has permission to bind to privileged ports

```bash
# Check what's using port 80/443
sudo netstat -tulpn | grep :80
sudo netstat -tulpn | grep :443

# Stop conflicting services
sudo systemctl stop apache2  # or nginx
```

#### 2. DNS Resolution Issues

**Error**: `DNS resolution failed for domain`

**Solution**:
- Verify DNS A record points to your server
- Check domain propagation
- Ensure domain is accessible from the internet

```bash
# Test DNS resolution from external perspective
dig +short your-domain.com @8.8.8.8
```

#### 3. Certificate Directory Permissions

**Error**: `Permission denied writing to certificates directory`

**Solution**:
```bash
sudo chown -R warptail:warptail /etc/warptail/certs
sudo chmod 700 /etc/warptail/certs
```

#### 4. Rate Limiting

**Error**: `Too many certificates already issued`

**Solution**:
- Let's Encrypt has rate limits (20 certificates per week per domain)
- Wait for the rate limit window to reset
- Consider using staging environment for testing

### Debug Mode

Enable debug logging to troubleshoot ACME issues:

```yaml
logging:
  level: debug
```

Check logs for ACME-related messages:

```bash
sudo journalctl -u warptail -f | grep -i acme
```

## Security Considerations

1. **Certificate Storage**: Ensure certificate directory has proper permissions (700)
2. **Private Keys**: Private keys should be readable only by warptail user
3. **Backup**: Regularly backup certificate directory
4. **Monitoring**: Monitor certificate expiration dates

## Example Complete Configuration

```yaml
application:
  host: "0.0.0.0"
  port: 8080

acme:
  enabled: true
  ssl_port: 443
  certificates_dir: "/etc/warptail/certs"
  portal_domain: "warptail.example.com"

logging:
  level: info
  format: json
```

## Testing

### Staging Environment

For testing, you can use Let's Encrypt staging environment to avoid rate limits:

```yaml

acme:
    enabled: true
    ssl_port: 443
    certificates_dir: "/tmp/warptail-certs"
    portal_domain: "test.your-domain.com"
    # Note: staging configuration would be handled in code
```

### Verify HTTPS

After setup, verify HTTPS is working:

```bash
# Test HTTPS connection
curl -I https://your-domain.com

# Check certificate details
curl -vI https://your-domain.com 2>&1 | grep -A 10 "SSL certificate"
```

## Support

If you encounter issues with ACME certificate management:

1. Check the troubleshooting section above
2. Review warptail logs for error messages
3. Verify your domain and network configuration
4. Consult Let's Encrypt documentation for ACME-specific issues

For additional help, please refer to the main warptail documentation or file an issue in the project repository.