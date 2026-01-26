# claim-machinery-api Kubernetes Deployment

KCL module for deploying claim-machinery-api on Kubernetes.

## Render Manifests

```bash
# Default configuration
kcl main.k

# Output as YAML
kcl main.k -o yaml
```

## Override Variables

Use `-D` flag to override configuration at render time:

```bash
# Override single variable
kcl main.k -D config.replicas=3

# Override multiple variables
kcl main.k -D config.namespace=production -D config.replicas=3

# Override image
kcl main.k -D config.image="ghcr.io/stuttgart-things/claim-machinery-api:v1.0.0"

# Enable ingress with custom host
kcl main.k -D config.ingressEnabled=True -D config.ingressHost="api.example.com"

# Enable TLS
kcl main.k \
  -D config.ingressEnabled=True \
  -D config.ingressHost="api.example.com" \
  -D config.ingressTlsEnabled=True \
  -D config.ingressTlsSecretName="api-tls" # pragma: allowlist secret

# Production-like setup
kcl main.k \
  -D config.namespace=production \
  -D config.replicas=3 \
  -D config.cpuRequest="250m" \
  -D config.cpuLimit="1000m" \
  -D config.memoryRequest="256Mi" \
  -D config.memoryLimit="512Mi" \
  -D config.logFormat="json" \
  -D config.ingressEnabled=True \
  -D config.ingressHost="claim-machinery-api.sva.dev" \
  -D config.ingressTlsEnabled=True
```

## Render to YAML Files

```bash
# Render to single YAML file
kcl main.k -o yaml > manifests.yaml

# Render with overrides to file
kcl main.k \
  -D config.namespace=production \
  -D config.replicas=3 \
  -o yaml > production.yaml

# Render with ingress enabled
kcl main.k \
  -D config.ingressEnabled=True \
  -D config.ingressHost="api.sva.dev" \
  -D config.ingressTlsEnabled=True \
  -o yaml > manifests-with-ingress.yaml

# Render to environment-specific files
kcl main.k -D config.namespace=dev -o yaml > deploy-dev.yaml
kcl main.k -D config.namespace=staging -D config.replicas=2 -o yaml > deploy-staging.yaml
kcl main.k -D config.namespace=prod -D config.replicas=3 -o yaml > deploy-prod.yaml
```

## Apply to Cluster

```bash
# Apply from rendered file
kubectl apply -f manifests.yaml

# Render and apply directly (pipe)
kcl main.k -o yaml | kubectl apply -f -

# Render with overrides and apply
kcl main.k -D config.namespace=production -o yaml | kubectl apply -f -

# Dry-run (client-side validation)
kcl main.k -o yaml | kubectl apply --dry-run=client -f -

# Dry-run (server-side validation)
kcl main.k -o yaml | kubectl apply --dry-run=server -f -

# Delete resources
kcl main.k -o yaml | kubectl delete -f -
```

## Available Variables

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `config.name` | string | `claim-machinery-api` | Application name |
| `config.namespace` | string | `default` | Kubernetes namespace |
| `config.image` | string | `ghcr.io/stuttgart-things/claim-machinery-api:latest` | Container image |
| `config.imagePullPolicy` | string | `IfNotPresent` | Image pull policy |
| `config.replicas` | int | `1` | Number of replicas |
| `config.cpuRequest` | string | `100m` | CPU request |
| `config.cpuLimit` | string | `500m` | CPU limit |
| `config.memoryRequest` | string | `128Mi` | Memory request |
| `config.memoryLimit` | string | `256Mi` | Memory limit |
| `config.serviceType` | string | `ClusterIP` | Service type |
| `config.servicePort` | int | `8080` | Service port |
| `config.containerPort` | int | `8080` | Container port |
| `config.ingressEnabled` | bool | `False` | Enable ingress |
| `config.ingressClassName` | string | `nginx` | Ingress class |
| `config.ingressHost` | string | `claim-machinery-api.example.com` | Ingress hostname |
| `config.ingressTlsEnabled` | bool | `False` | Enable TLS |
| `config.ingressTlsSecretName` | string | `claim-machinery-api-tls` | TLS secret name |
| `config.templatesDir` | string | `/app/templates` | Templates directory |
| `config.templateProfilePath` | string | `` | Template profile path |
| `config.port` | string | `8080` | Application port |
| `config.logFormat` | string | `text` | Log format (text/json) |
| `config.debug` | bool | `False` | Enable debug mode |

## Files

| File | Description |
|------|-------------|
| `schema.k` | Configuration schema |
| `labels.k` | Common labels |
| `serviceaccount.k` | ServiceAccount resource |
| `configmap.k` | ConfigMap resource |
| `secret.k` | Secret resource |
| `deploy.k` | Deployment resource |
| `service.k` | Service resource |
| `ingress.k` | Ingress resource |
| `main.k` | Entry point |
