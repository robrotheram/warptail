<p align="center"> 
  <img  src="dashboard/public/logo.png" width="200" />
</p>


# WarpTail

WarpTail is a tool designed to simplify proxying connections from the internet to services hosted on your Tailscale tailnet. It offers secure and seamless access to private services on your tailnet using proxy techniques and supports both Docker and Kubernetes environments.

## Features
- Easy setup to expose services from your Tailscale tailnet to the internet.
- YAML-based configuration for flexibility.
- Dynamic port routing and management.
- Built-in dashboard for monitoring and control.
- Automated ingress management and traffic routing in Kubernetes.


## Diagram

```mermaid
flowchart LR;
A[Internet] -->|Inbound Traffic| B[WarpTail Proxy];

subgraph Public Deployment
direction RL
        B -.->C[Tailscale Tailnet];
    
end
subgraph Internal Deployment
direction direction LR
    C == Wireguard tunnel  ==> D[Tailscale Tailnet]
    D-.-> I[Private Service];
end
```
---

## Getting Started

### Prerequisites
- A Tailscale account with a valid authentication key.
- A service running inside your tailnet that you want to expose to the internet.
- Docker or Kubernetes setup.

## Installation

### VPS Installation (Ubuntu/Debian)

This guide walks you through installing WarpTail on a VPS running Ubuntu or Debian.

#### Step 1: System Preparation

First, update your system and install required dependencies:

```bash
# Update system packages
sudo apt update && sudo apt upgrade -y

# Install required packages
sudo apt install -y curl wget git build-essential

# Install Go (required for building from source)
curl -fsSL https://golang.org/dl/go1.21.linux-amd64.tar.gz | sudo tar -xzC /usr/local
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

#### Step 2: Create WarpTail User

Create a dedicated user for running WarpTail:

```bash
# Create warptail user
sudo useradd -r -m -s /bin/bash warptail

# Create necessary directories
sudo mkdir -p /etc/warptail
sudo mkdir -p /var/log/warptail
sudo mkdir -p /var/lib/warptail

# Set ownership
sudo chown -R warptail:warptail /etc/warptail /var/log/warptail /var/lib/warptail
```

#### Step 3: Download and Install WarpTail

You can either download a pre-built binary or build from source.

**Option A: Download Pre-built Binary**
```bash
# Download the latest release (replace with actual version)
wget https://github.com/robrotheram/warptail/releases/latest/download/warptail-linux-amd64
sudo mv warptail-linux-amd64 /usr/local/bin/warptail
sudo chmod +x /usr/local/bin/warptail
```

**Option B: Build from Source**
```bash
# Clone the repository
git clone https://github.com/robrotheram/warptail.git
cd warptail

# Build the application
go build -o warptail .

# Install the binary
sudo mv warptail /usr/local/bin/
sudo chmod +x /usr/local/bin/warptail
```

#### Step 4: Configure WarpTail

Create the main configuration file:

```bash
sudo tee /etc/warptail/config.yaml > /dev/null <<EOF
tailscale:
  authkey: "YOUR_TAILSCALE_AUTH_KEY"
  hostname: "warptail-proxy"

application:
  host: "0.0.0.0"
  port: 8080
  authentication:
    type: "password"  # or "openid"
    name: "WarpTail Admin"
    password: "your-secure-password"

logging:
  format: "json"
  level: "info"
  output: "file"
  path: "/var/log/warptail"

database:
  path: "/var/lib/warptail/warptail.db"

services: []
EOF

# Set proper ownership and permissions
sudo chown warptail:warptail /etc/warptail/config.yaml
sudo chmod 600 /etc/warptail/config.yaml
```

**Important**: Replace `YOUR_TAILSCALE_AUTH_KEY` with your actual Tailscale auth key from the [Tailscale Admin Console](https://login.tailscale.com/admin/settings/keys).

#### Step 5: Create Systemd Service

Create a systemd service file for automatic startup:

```bash
sudo tee /etc/systemd/system/warptail.service > /dev/null <<EOF
[Unit]
Description=WarpTail Proxy Service
Documentation=https://github.com/robrotheram/warptail
After=network.target
Wants=network.target

[Service]
Type=simple
User=warptail
Group=warptail
ExecStart=/usr/local/bin/warptail
WorkingDirectory=/var/lib/warptail
Environment=CONFIG_PATH=/etc/warptail/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=warptail

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/warptail /var/log/warptail

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd and enable the service
sudo systemctl daemon-reload
sudo systemctl enable warptail
```

#### Step 6: Configure Firewall

Open the necessary ports:

```bash
# Allow WarpTail dashboard port
sudo ufw allow 8080/tcp

# Allow HTTP and HTTPS traffic (if you plan to proxy web services)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Enable firewall if not already enabled
sudo ufw --force enable
```

#### Step 7: Start WarpTail

Start the service and check its status:

```bash
# Start WarpTail
sudo systemctl start warptail

# Check status
sudo systemctl status warptail

# View logs
sudo journalctl -u warptail -f
```

#### Step 8: Access the Dashboard

Once WarpTail is running:

1. Open your web browser and navigate to `http://YOUR_VPS_IP:8080`
2. Log in with the credentials you set in the config file
3. Configure your first service through the web interface

#### Step 9: Configure Reverse Proxy (Optional but Recommended)

For production use, it's recommended to put WarpTail behind a reverse proxy like Nginx.

**Install Nginx:**
```bash
sudo apt install -y nginx
```

**Create Nginx configuration:**
```bash
sudo tee /etc/nginx/sites-available/warptail > /dev/null <<EOF
server {
    listen 80;
    server_name your-domain.com;  # Replace with your domain

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

# Enable the site
sudo ln -s /etc/nginx/sites-available/warptail /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

**Optional: Add SSL with Certbot:**
```bash
# Install Certbot
sudo apt install -y snapd
sudo snap install --classic certbot
sudo ln -s /snap/bin/certbot /usr/bin/certbot

# Get SSL certificate
sudo certbot --nginx -d your-domain.com
```

#### Step 10: Configure Your First Service

1. Log into the WarpTail dashboard
2. Click "Add Service" 
3. Configure your service:
   - **Name**: My Web App
   - **Type**: HTTPS
   - **Domain**: myapp.yourdomain.com
   - **Target**: Your Tailscale device IP and port
   - **Private**: Enable if you want authentication

#### Troubleshooting

**Check WarpTail logs:**
```bash
sudo journalctl -u warptail -n 50
```

**Check access/error logs:**
```bash
sudo tail -f /var/log/warptail/access.log
sudo tail -f /var/log/warptail/error.log
```

**Verify Tailscale connection:**
```bash
# Check if WarpTail appears in your Tailscale admin console
# The hostname should be "warptail-proxy" (or what you configured)
```

**Common Issues:**
- **Port 8080 not accessible**: Check firewall rules and VPS security groups
- **Tailscale auth fails**: Verify your auth key is valid and not expired
- **Services not reachable**: Ensure target services are accessible from within your Tailnet

#### Updating WarpTail

To update WarpTail to a newer version:

```bash
# Stop the service
sudo systemctl stop warptail

# Download new version (or rebuild from source)
wget https://github.com/robrotheram/warptail/releases/latest/download/warptail-linux-amd64
sudo mv warptail-linux-amd64 /usr/local/bin/warptail
sudo chmod +x /usr/local/bin/warptail

# Start the service
sudo systemctl start warptail
```

## Configuration

### Quick VPS Installation

For a quick installation on Ubuntu/Debian VPS:

```bash
curl -fsSL https://raw.githubusercontent.com/robrotheram/warptail/main/scripts/install-vps.sh | bash
```

Or follow the detailed [VPS Installation Guide](docs/vps-installation.md).

### Authentication Options

WarpTail supports multiple authentication methods:

- **No Authentication**: Remove the `authentication` section entirely
- **Password Authentication**: Simple username/password login
- **OpenID Connect**: Support for Google, Microsoft, GitHub, and custom OIDC providers

See the [installation guide](docs/vps-installation.md#authentication-configuration) for detailed configuration examples.

### Manual Configuration

WarpTail uses a `config.yaml` file for all configuration management. The configuration covers settings for Tailscale authentication, dashboard access, and routing rules for exposing services.

### Sample Configuration File
```yaml
tailscale:
  auth_key: "tskey-auth-XXXXXXXXXXXXXXXXXXXXXXXXXXX"
  hostname: "warptail-proxy"

application:
  host: "0.0.0.0"
  port: 8080
  authentication:  # Optional - remove entire section to disable auth
    baseURL: "https://your-domain.com"
    secretKey: "your-random-secret-key"
    provider:
      name: "WarpTail Admin"
      type: "password"  # or "openid" for SSO
      session_secret: "your-secure-password"
      # For OpenID providers, add:
      # clientID: "your-client-id"
      # providerURL: "https://your-oidc-provider.com"

database:
  path: "/var/lib/warptail/warptail.db"  # SQLite (default)
  # For PostgreSQL/MySQL:
  # connection_type: "postgres"
  # connection: "postgresql://user:password@host/dbname?sslmode=require"

logging:
  format: "json"    # Choose "json" or "console"
  level: "info"     # Logging level: info, warn, error
  output: "file"    # Choose "stdout" for console or file path
  path: "/var/log/warptail"

services:
  - name: "web application"
    enabled: true
    routes:
      - type: "https"
        private: false
        bot_protect: false
        domain: "myapp.example.com"
        machine:
          address: "192.168.1.100"
          port: 8080
        proxy_settings:  # Optional advanced routing
          timeout: 30
          rules:
            - path: "/api/"
              target_host: "api-server"
              target_port: 3000
              strip_path: true

  - name: "minecraft server"
    enabled: true
    routes:
      - type: "tcp"
        port: 25565
        machine:
          address: "192.168.1.200"
          port: 25565

# Optional Kubernetes-specific configuration
kubernetes:
  namespace: "warptail"
  ingress:
    name: "warptail-routes"
    class: "traefik"
  service:
    name: "warptail-service"


```
### Configuration Sections

### 1. Tailscale
The Tailscale section configures Tailscale VPN integration.

#### Parameters:
- **auth_key**: Authentication key for Tailscale.
- **hostname**: The hostname to register with Tailscale.


### 2. Logging
The logging section configures the logging behavior for the application.

#### Parameters:
- **format**: The format of the logs. Options:
  - `stdout`: Logs in a human-readable format.
  - `json`: Logs in JSON format.
- **level**: The logging level. Options:
  - `info`
  - `warn`
  - `error`
- **output**: The logging output destination. Options:
  - `stdout`: Logs to the console.
  - `<filepath>`: Logs to the specified file.

### 3. Routes
The routes section defines custom routing for HTTP and TCP traffic.

#### Parameters:
- **enabled**: Set to `true` to enable the route.
- **private** (Optional): For HTTP routes, specifies if the route is private and requires authentication.
- **name**: A unique name for the route.
- **type**: The type of route. Options:
  - `http`
  - `tcp`
- **listen** (TCP only): The port to listen on for TCP routes.
- **machine**:
  - **address**: The IP address of the machine to route traffic to.
  - **port**: The port on the machine to route traffic to.

### 4. Database
The database section defines the connection type and credentials for the application to access its database.

#### Parameters:
- **connection_type**: Specifies the type of database connection. Options include:
  - `sqlite`
  - `postgres`
  - `mysql`
- **connection**: The connection string for the database. Examples:
  - PostgreSQL: `postgresql://<user>:<password>@<host>/<dbname>?sslmode=require`
  - SQLite: `file:test.db?cache=shared`
  - MySQL: `mysql://<user>:<password>@tcp(<host>:<port>)/<dbname>`

### 5. Application
The application section contains settings for the primary application logic, including ports and authentication.

#### Parameters:
- **port**: The main port the application listens on.
- **authentication**:
  - **baseURL**: The base URL of the authentication service.
  - **secretKey**: A secret key for authentication and encryption purposes.
  - **provider** (Optional):
    - **name**: Name of the SSO provider (e.g., `zitadle`).
    - **type**: The type of SSO protocol used (must be `openid` with PKCE support).
    - **clientID**: Client ID for the SSO provider.
    - **providerURL**: The provider's URL.

### 5.1. ACME SSL (Optional)
The ACME section enables automatic SSL certificate provisioning and management.

#### Parameters:
- **enabled**: Set to `true` to enable ACME configuration.
- **ssl_port**: The port used for SSL/TLS connections. Default: `443`.
- **certificates_dir**: Directory where the certificates are stored.
- **portal_domain**: Domain name for which the SSL certificate will be provisioned.

**Important:**
- ACME configuration is incompatible with Kubernetes deployments.
- When ACME is enabled, the application port must be set to `80` to ensure successful HTTP challenge validation.



### 6. Kubernetes (Optional)
The Kubernetes section is optional and used for deployments within a Kubernetes cluster. 

#### Parameters:
- **namespace**: The namespace where the resources are deployed. Leave empty to use the default namespace.
- **loadbalancer.name**: The name of the Kubernetes load balancer.
- **certificate**:
  - **name**: Name of the certificate resource.
  - **secret_name**: Name of the secret storing the certificate.
- **ingress**:
  - **name**: Name of the Kubernetes ingress resource.
  - **class**: Ingress class (e.g., `traefik`).
  - **service**: The backend service name.

**Note:**
- If `kubernetes` is configured, ACME configuration will be disabled automatically.


Here's a draft section for your README file:  

---

## Authentication Details

### Default Login (Admin)
When the proxy service starts, a default admin account is created for initial setup:  
- **Username:** `admin@warptail.local`  
- **Password:** Auto-generated and displayed in the console logs during the initial startup.  

> **Note:** Make sure to check the console output to retrieve the generated password. For security purposes, it is highly recommended to log in and change this password immediately after the first login.

### OpenID Authentication
If you are using OpenID for authentication, the default role assigned to authenticated users is `user`.  

To modify user roles:  
1. Log in to the portal using the admin account (`admin@warptail.local`).  
2. Navigate to the user management section.  
3. Update the roles as necessary.  

This setup ensures flexibility in managing access levels while maintaining a secure default configuration.  



Here's a section you can add to your README for installation:

---

## Installation Binary 

### Prerequisites
Before installing `warptail`, ensure that you have the following:
- A Linux-based operating system with `systemd` enabled.
- Administrator/root privileges to install services.

### Installing Warptail

To install `warptail` as a systemd service, follow these steps:

1. **Download the Binary**  
   Download the `warptail` binary from the [releases page](https://github.com/robrotheram/warptail/releases)

2. **Make the Binary Executable**  
   Run the following command to make the binary executable:
   ```bash
   chmod +x warptail
   ```

3. **Install the Service**  
   Use the `install` command to install `warptail` as a systemd service:
   ```bash
   ./warptail install
   ```

4. **Verify the Installation**  
   Check that the service is active and running:
   ```bash
   systemctl status warptail
   ```

### Updating Warptail

To update `warptail` to the latest version, simply run:
```bash
./warptail update
```

### Uninstalling Warptail

If you need to remove `warptail`, run:
```bash
./warptail uninstall
```




## Running WarpTail on Docker

When running WarpTail in Docker, you'll need to mount the `config.yaml` to the container and decide between specifying all proxy ports upfront or using host networking for dynamic routing.

Here are some Docker run examples for different configurations:

---

### 1. Running with SQLite (Without ACME)
This example uses SQLite for database storage and does not enable ACME.

```bash
docker run -d \
  -p 8080:8080 \
  -v /path/to/config.yaml:/app/config.yaml \
  -v /path/to/sqlite.db:/app/sqlite.db \
  -e CONFIG_FILE=/app/config.yaml \
  proxy-service:latest
```

**Notes:**
- The application listens on port 8080.
- Ensure the `database.connection_type` is set to `sqlite`.
- The SQLite database file is mounted from the host.

---

### 2. Running with PostgreSQL (Without ACME)
This example uses PostgreSQL as the database and disables ACME.

```bash
docker run -d \
  -p 8080:8080 \
  -v /path/to/config.yaml:/app/config.yaml \
  -e CONFIG_FILE=/app/config.yaml \
  proxy-service:latest
```

**Notes:**
- The application listens on port 8080.
- Ensure the `database.connection_type` is set to `postgres` and the connection string is correctly configured.

---

### 3. Running with ACME SSL (SQLite)
This example enables ACME for SSL and uses SQLite as the database.

```bash
docker run -d \
  -p 80:80 \
  -p 443:443 \
  -v /path/to/config.yaml:/app/config.yaml \
  -v /path/to/certs:/etc/proxy/certs \
  -v /path/to/sqlite.db:/app/sqlite.db \
  -e CONFIG_FILE=/app/config.yaml \
  proxy-service:latest
```

**Notes:**
- The application listens on port 80 for HTTP (required for ACME HTTP challenge).
- The SSL certificates are stored in `/path/to/certs`.
- SQLite database file is mounted from the host.

---

### 4. Running with ACME SSL (PostgreSQL)
This example enables ACME for SSL and uses PostgreSQL as the database.

```bash
docker run -d \
  -p 80:80 \
  -p 443:443 \
  -v /path/to/config.yaml:/app/config.yaml \
  -v /path/to/certs:/etc/proxy/certs \
  -e CONFIG_FILE=/app/config.yaml \
  proxy-service:latest
```

**Notes:**
- The application listens on port 80 for HTTP (required for ACME HTTP challenge).
- PostgreSQL connection string should be correctly configured in the `config.yaml`.

---


### 5. Specifying Ports Upfront

In this mode, you must specify all the ports you wish to proxy. Make sure your `config.yaml` has the correct port mappings defined in the `routes` section.

```bash
docker run -d \
  --name warptail \
  -e CONFIG_PATH=/app/config.yaml \
  -v /path/to/config.yaml:/app/config.yaml \

  -p 80:80 \
  -p 443:443 \
  -p 30041:30041 \
  ghcr.io/robrotheram/warptail:latest
```

- Mount the `config.yaml` file using `-v /path/to/config.yaml:/app/config.yaml`.
- Expose the ports defined in your configuration.

---

### 6. Using Host Networking

For dynamic port management, you can run WarpTail with Docker's host networking:

```bash
docker run -d \
  --name warptail \
  --network host \
  -e CONFIG_PATH=/app/config.yaml \
  -v /path/to/config.yaml:/app/config.yaml \
  ghcr.io/robrotheram/warptail:latest
```

Host networking allows WarpTail to dynamically route traffic without needing to expose individual ports.

---

### Key Considerations
1. **ACME Configuration**: 
   - Ensure `application.port` is set to `80` when enabling ACME.
   - SSL certificates will be stored in the directory specified by `acme.certificates_dir`.

2. **SQLite Storage**: 
   - Mount the SQLite database file to a persistent location.

3. **PostgreSQL Connection**: 
   - Verify the connection string and ensure the database server is accessible.

4. **Configuration File**:
   - Always provide the path to the `config.yaml` file using the `-v` flag and set the `CONFIG_FILE` environment variable.



## Running WarpTail on Kubernetes

WarpTail manages its own ingress and routes traffic through node-ports in Kubernetes. This requires creating a service account for it to handle ingress and service resources.

### 1. Setup Service Account

See `manifests` folder for example kubernetes manifiests


### 2. Accessing the Service

Once deployed, WarpTail will automatically configure ingress and route traffic through node-ports. Access your exposed services through your Kubernetes cluster's external IP using the node-port (e.g., `http://<cluster-ip>:30080` for HTTP).


Here's a README section for documenting the Prometheus metrics exposed by the Golang service **Warptail**:

---

Here’s a new section explaining how to configure WarpTail as a Kubernetes controller with a custom CRD (`WarpTailService`) to manage service configuration directly in Kubernetes:

---

## Kubernetes Controller with Custom CRD Support

WarpTail can be deployed as a Kubernetes controller, allowing users to manage WarpTail service configurations through a Custom Resource Definition (CRD). This approach enables a Kubernetes-native setup, where you can define services and routing rules directly within the cluster using custom resources.

### Custom Resource Definition (CRD)

The `WarpTailService` CRD allows you to define routing and service configurations using a Kubernetes resource. This makes it easy to manage services, automate deployments, and integrate with Kubernetes-native tools.

### Example CRD Configuration

To set up a WarpTail service using the `WarpTailService` CRD, create a YAML file defining the resource, specifying details such as the domain, protocol, machine IP, and port.

#### Example `WarpTailService` Resource
```yaml
apiVersion: warptail.exceptionerror.io/v1
kind: WarpTailService
metadata:
  name: jellyfin
  namespace: warptail
spec:
  routes:
    - type: http
      domain: https://jellyfin.exceptionerror.io/
      machine:
        address: 192.168.0.104
        port: 30013
```

In this example:
- **`apiVersion`**: Defines the API version for the `WarpTailService` resource.
- **`kind`**: Specifies the type of the resource, which is `WarpTailService`.
- **`metadata.name`**: Unique name for the service in Kubernetes.
- **`metadata.namespace`**: Namespace where the resource is defined (e.g., `warptail`).
- **`spec.routes`**: Specifies the routing configuration for the service.
  - **`type`**: Defines the protocol type (e.g., `http`, `tcp`).
  - **`domain`**: The external domain or URL that maps to the service.
  - **`machine.address`**: Internal IP address of the machine within the tailnet.
  - **`machine.port`**: The port on which the service runs internally.

### Deploying the CRD

To deploy the `WarpTailService` CRD, save the configuration to a YAML file (e.g., `jellyfin-service.yaml`) and apply it to your Kubernetes cluster:

```bash
kubectl apply -f jellyfin-service.yaml
```

### Managing Services with CRD

Once the `WarpTailService` CRD is deployed, the WarpTail Kubernetes controller will automatically manage the service:
- It will configure ingress rules based on the specified domains.
- Routes will be created dynamically, allowing access to the specified machine IP and port.
- Changes to the CRD will be automatically picked up, and the routing will be updated accordingly.

### Benefits of CRD-based Configuration
Using a CRD for WarpTail services provides several advantages:
- **Kubernetes-Native**: Manage WarpTail configurations alongside other Kubernetes resources.
- **Declarative Management**: Define all routing rules declaratively and store configurations in version-controlled YAML files.
- **Automated Updates**: Modify the CRD to update WarpTail’s routing dynamically without editing the `config.yaml`.

### Example: Listing and Managing WarpTail Services

To list all configured `WarpTailService` resources in the `warptail` namespace:

```bash
kubectl get warptailservice -n warptail
```

To view detailed information on a specific `WarpTailService`:

```bash
kubectl describe warptailservice jellyfin -n warptail
```

By using the `WarpTailService` CRD, you integrate WarpTail seamlessly within your Kubernetes ecosystem, making it easier to manage, deploy, and update your proxy services.

---

## Prometheus Metrics

**Warptail** exposes a set of Prometheus metrics for monitoring its services and routes. These metrics are available at the `/metrics` endpoint.

#### Custom Metrics

Below are the custom metrics available for Warptail, along with their descriptions and types:

- **`warptail_route_status`** (`gauge`):  
  Indicates the status of various routes in the Warptail service.

- **`warptail_service_enabled`** (`gauge`):  
  Shows if a particular Warptail service is enabled (1 if enabled, 0 otherwise).

- **`warptail_service_latency`** (`gauge`):  
  Displays the latency for the Warptail service in milliseconds.

- **`warptail_service_route_latency`** (`gauge`):  
  Shows the latency for specific routes in the Warptail service.

- **`warptail_service_total_received`** (`gauge`):  
  Tracks the total amount of data received by a specific Warptail service.

- **`warptail_service_total_sent`** (`gauge`):  
  Tracks the total amount of data sent by a specific Warptail service.

---

## Contributing
To contribute to WarpTail, please open an issue or submit a pull request. Contributions are always welcome!

---

## License
WarpTail is licensed under the MIT License. See `LICENSE` for more details.