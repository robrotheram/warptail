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

### Configuration

WarpTail uses a `config.yaml` file for all configuration management. The configuration covers settings for Tailscale authentication, dashboard access, and routing rules for exposing services.

#### Example `config.yaml`:
```yaml
tailscale:
  auth_key: tskey-auth-XXXXXXXXXXXXXXXXXXXXXXXXXXX
  hostname: warptail

dashboard:
  enabled: true
  username: admin
  password: mallard

# Specify Routes
routes:
    # Example HTTP Route
  - enabled: true
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
      
# Optional Kubernetes-specific configuration
kubernetes:
  namespace: warptail
  ingress_name: warptail-routes
  service_name: warptail-service
  ingress_class: traefik


```

- **`tailscale.auth_key`**: Tailscale authentication key to authorize access to your tailnet.
- **`tailscale.hostname`**: The hostname used for your WarpTail instance on the tailnet.
- **`dashboard.enabled`**: Enables or disables the WarpTail dashboard.
- **`dashboard.username`** / **`dashboard.password`**: Credentials for accessing the WarpTail dashboard.
- **`kubernetes`**: Kubernetes-specific settings for managing ingress, services, and routing.
- **`routes`**: Define the services within your tailnet that you want to expose. Each route specifies a domain name, the protocol (`http`, `tcp`, `udp`), and the internal machine's IP address and port.

---

## Running WarpTail on Docker

When running WarpTail in Docker, you'll need to mount the `config.yaml` to the container and decide between specifying all proxy ports upfront or using host networking for dynamic routing.

### 1. Specifying Ports Upfront

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

### 2. Using Host Networking

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