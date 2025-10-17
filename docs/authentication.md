# WarpTail Authentication Configuration Guide

This guide covers all authentication options available in WarpTail, from no authentication to various OpenID Connect providers.

## Authentication Overview

WarpTail supports three main authentication modes:

1. **No Authentication** - Open access (development only)
2. **Password Authentication** - Simple built-in login
3. **OpenID Connect** - SSO with external providers

## Configuration Structure

The authentication configuration follows this YAML structure:

```yaml
application:
  authentication:  # Optional - remove entire section to disable auth
    baseURL: "https://your-warptail-domain.com"
    secretKey: "random-secret-for-session-encryption"
    provider:
      name: "Provider Display Name"
      type: "password" # or "openid"
      session_secret: "password-or-client-secret"
      # OpenID-specific fields (when type: "openid"):
      clientID: "your-client-id"
      providerURL: "https://your-oidc-provider.com"
```

## Option 1: No Authentication

**⚠️ Warning**: Only use this for development or internal networks.

Simply remove or comment out the entire `authentication` section:

```yaml
application:
  host: "0.0.0.0"
  port: 8080
  # No authentication section = no auth required

database:
  path: "/var/lib/warptail/warptail.db"
# ... rest of config
```

## Option 2: Password Authentication

Simple username/password login with configurable credentials:

```yaml
application:
  authentication:
    baseURL: "https://your-domain.com"
    secretKey: "your-random-secret-key"  # Generate with: openssl rand -base64 32
    provider:
      name: "WarpTail Admin"
      type: "password"
      session_secret: "your-secure-password"
```

- **Username**: Always `admin`
- **Password**: The value in `session_secret`

## Option 3: OpenID Connect Providers

### Google OAuth

```yaml
application:
  authentication:
    baseURL: "https://your-domain.com"
    secretKey: "your-random-secret-key"
    provider:
      name: "Google SSO"
      type: "openid"
      clientID: "123456789-abcdefg.apps.googleusercontent.com"
      providerURL: "https://accounts.google.com"
      session_secret: "your-google-client-secret"
```

**Setup Steps:**

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create or select a project
3. Enable "Google+ API" (or "Google Identity")
4. Go to "Credentials" → "Create Credentials" → "OAuth 2.0 Client IDs"
5. Application type: "Web application"
6. Authorized redirect URIs: `https://your-domain.com/auth/callback`
7. Copy Client ID and Client Secret

### Microsoft Azure AD

```yaml
application:
  authentication:
    baseURL: "https://your-domain.com"
    secretKey: "your-random-secret-key"
    provider:
      name: "Microsoft SSO"
      type: "openid"
      clientID: "12345678-1234-1234-1234-123456789abc"
      providerURL: "https://login.microsoftonline.com/your-tenant-id/v2.0"
      session_secret: "your-azure-client-secret"
```

**Setup Steps:**

1. Go to [Azure Portal](https://portal.azure.com/)
2. Navigate to "Azure Active Directory" → "App registrations"
3. Click "New registration"
4. Name: "WarpTail"
5. Redirect URI: `https://your-domain.com/auth/callback`
6. Copy Application (client) ID and Directory (tenant) ID
7. Go to "Certificates & secrets" → create new client secret
8. Update `providerURL` with your tenant ID

### GitHub OAuth

```yaml
application:
  authentication:
    baseURL: "https://your-domain.com"
    secretKey: "your-random-secret-key"
    provider:
      name: "GitHub SSO"
      type: "openid"
      clientID: "your-github-client-id"
      providerURL: "https://github.com"
      session_secret: "your-github-client-secret"
```

**Setup Steps:**

1. Go to GitHub Settings → Developer settings → OAuth Apps
2. Click "New OAuth App"
3. Application name: "WarpTail"
4. Homepage URL: `https://your-domain.com`
5. Authorization callback URL: `https://your-domain.com/auth/callback`
6. Copy Client ID and generate Client Secret

### Auth0

```yaml
application:
  authentication:
    baseURL: "https://your-domain.com"
    secretKey: "your-random-secret-key"
    provider:
      name: "Auth0 SSO"
      type: "openid"
      clientID: "your-auth0-client-id"
      providerURL: "https://your-tenant.auth0.com"
      session_secret: "your-auth0-client-secret"
```

**Setup Steps:**

1. Go to [Auth0 Dashboard](https://manage.auth0.com/)
2. Applications → Create Application
3. Choose "Regular Web Applications"
4. Go to Settings tab
5. Set Allowed Callback URLs: `https://your-domain.com/auth/callback`
6. Copy Client ID, Client Secret, and Domain

### Okta

```yaml
application:
  authentication:
    baseURL: "https://your-domain.com"
    secretKey: "your-random-secret-key"
    provider:
      name: "Okta SSO"
      type: "openid"
      clientID: "your-okta-client-id"
      providerURL: "https://your-org.okta.com"
      session_secret: "your-okta-client-secret"
```

**Setup Steps:**

1. Go to Okta Admin Console
2. Applications → Create App Integration
3. Sign-in method: "OIDC - OpenID Connect"
4. Application type: "Web Application"
5. Sign-in redirect URIs: `https://your-domain.com/auth/callback`
6. Copy Client ID and Client Secret

### Keycloak

```yaml
application:
  authentication:
    baseURL: "https://your-domain.com"
    secretKey: "your-random-secret-key"
    provider:
      name: "Keycloak SSO"
      type: "openid"
      clientID: "warptail"
      providerURL: "https://your-keycloak.com/realms/your-realm"
      session_secret: "your-keycloak-client-secret"
```

**Setup Steps:**

1. Login to Keycloak Admin Console
2. Select your realm
3. Clients → Create Client
4. Client ID: "warptail"
5. Client authentication: "On"
6. Valid redirect URIs: `https://your-domain.com/auth/callback`
7. Copy Client Secret from Credentials tab

### Generic OpenID Connect

For any other OIDC-compliant provider:

```yaml
application:
  authentication:
    baseURL: "https://your-domain.com"
    secretKey: "your-random-secret-key"
    provider:
      name: "Custom OIDC"
      type: "openid"
      clientID: "your-client-id"
      providerURL: "https://your-provider.com"  # Or discovery URL
      session_secret: "your-client-secret"
```

## Configuration Fields Reference

### Required Fields

- **baseURL**: Your WarpTail's public URL (needed for OAuth callbacks)
- **secretKey**: Random string for session encryption (generate with `openssl rand -base64 32`)

### Provider Fields

- **name**: Display name shown in UI
- **type**: Either `"password"` or `"openid"`
- **session_secret**: 
  - For password auth: your login password
  - For OpenID: the OAuth client secret

### OpenID-Specific Fields

- **clientID**: OAuth application client ID
- **providerURL**: OIDC provider base URL or discovery endpoint

## Common Issues & Troubleshooting

### OAuth Callback Issues

**Problem**: "Invalid redirect URI" or callback errors

**Solutions**:
1. Ensure `baseURL` in config matches your actual domain
2. Add exact callback URL to provider: `https://your-domain.com/auth/callback`
3. Check for trailing slashes and http vs https

### Session Issues

**Problem**: Constant re-login required

**Solutions**:
1. Ensure `secretKey` is properly set and doesn't change
2. Check system time is synchronized
3. Verify cookies aren't being blocked

### Provider Connection Issues

**Problem**: "Failed to connect to provider"

**Solutions**:
1. Verify `providerURL` is correct
2. Check network connectivity from WarpTail server
3. Ensure client ID and secret are correct
4. Check provider-specific requirements (scopes, etc.)

## Security Considerations

### For Password Authentication

- Use strong passwords (minimum 16 characters)
- Change default passwords immediately
- Consider rotating passwords regularly

### For OpenID Authentication

- Use HTTPS only for production
- Keep client secrets secure
- Regularly rotate OAuth credentials
- Restrict OAuth application permissions to minimum required

### General Security

- Always use HTTPS in production
- Keep `secretKey` secure and random
- Monitor authentication logs
- Consider IP restrictions if needed

## Migration Between Auth Methods

### From No Auth to Password Auth

1. Add authentication section to config
2. Restart WarpTail
3. Access with new credentials

### From Password to OpenID

1. Set up OpenID provider first
2. Update config with OpenID settings
3. Test with a separate user if possible
4. Restart WarpTail

### Fallback Strategy

Always keep a backup authentication method:

```yaml
# Example: Keep admin password as backup
application:
  authentication:
    baseURL: "https://your-domain.com"
    secretKey: "your-random-secret-key"
    provider:
      name: "Google SSO"
      type: "openid"
      clientID: "your-google-client-id"
      providerURL: "https://accounts.google.com"
      session_secret: "your-google-client-secret"
      # Keep this info documented for emergency access:
      # Emergency password auth: change type to "password" 
      # and use session_secret as password
```

## Testing Authentication

### Test OpenID Setup

1. Configure provider but don't restart WarpTail yet
2. Test redirect URL manually: `https://your-provider.com/auth?client_id=YOUR_CLIENT_ID&redirect_uri=https://your-domain.com/auth/callback&response_type=code`
3. Should redirect to your callback URL
4. If successful, restart WarpTail and test login

### Verify Configuration

```bash
# Check WarpTail logs for auth issues
sudo journalctl -u warptail -f | grep -i auth

# Test config syntax
sudo -u warptail /usr/local/bin/warptail --config /etc/warptail/config.yaml --dry-run
```