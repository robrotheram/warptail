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

## Configuration

WarpTail uses a `config.yaml` file for all configuration management. The configuration covers settings for Tailscale authentication, dashboard access, and routing rules for exposing services.

### Sample Configuration File
```yaml
tailscale:
  auth_key: tskey-auth-XXXXXXXXXXXXXXXXXXXXXXXXXXX
  hostname: warptail
logging:
  format: stdout    # Choose "json" or consol
  level: info       # Logging level: info, warn, error
  output: stdout    # Choose the output, "stdout" for console output or a filepath
# Specify Routes
routes:
    # Example HTTP Route
  - enabled: true
    private: true
    name: immich.example.io
    type: http
    machine:
      address: 127.0.0.1
      port: 30041

    # Example TCP Route
  - enabled: true
    name: minecraft server
    type: tcp
    listen: 25565
    machine:
      address: 127.0.0.1
      port: 25565      

database:
  connection_type: postgres # Options: sqlite, postgres, mysql
  connection: postgresql://<user>:<password>@<host>/<dbname>?sslmode=require # Connection string for the database

application:
  port: 8080  # The primary port the application listens on
  authentication:
    baseURL: http://localhost:8001
    secretKey: <REPLACE_WITH_SECRET_KEY>
    provider: # Optional SSO provider configuration
      name: zitadle
      type: openid # Only openID with PKCE support is supported
      clientID: "<CLIENT_ID>"
      providerURL: "https://auth.exceptionerror.io"
  acme:
    enabled: true  # Enable ACME SSL configuration
    ssl_port: 443  # Port for SSL/TLS connections
    certificates_dir: /etc/proxy/certs  # Directory to store SSL certificates
    portal_domain: warptail.excptionerror.local  # SSL domain for certificate provisioning

# Optional Kubernetes-specific configuration
kubernetes:
  namespace: warptail
  ingress_name: warptail-routes
  service_name: warptail-service
  ingress_class: traefik


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