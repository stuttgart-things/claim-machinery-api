# claim-machinery-api Kubernetes Deployment

KCL module for deploying claim-machinery-api on Kubernetes.

## Render Manifests

```bash
# Default configuration (outputs YAML)
kcl run main.k

# Output as JSON
kcl run main.k --format json
```

## Override Variables

Use `-D` flag to override configuration at render time:

```bash
# Override single variable
kcl run main.k -D config.replicas=3

# Override multiple variables
kcl run main.k -D config.namespace=production -D config.replicas=3

# Override image
kcl run main.k -D config.image="ghcr.io/stuttgart-things/claim-machinery-api:v1.0.0"

# Enable ingress with custom host
kcl run main.k -D config.ingressEnabled=True -D config.ingressHost="api.example.com"

# Enable TLS
kcl run main.k \
  -D config.ingressEnabled=True \
  -D config.ingressHost="api.example.com" \
  -D config.ingressTlsEnabled=True \
  -D config.ingressTlsSecretName="api-tls" # pragma: allowlist secret

# Production-like setup
kcl run main.k \
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
kcl run main.k -o manifests.yaml

# Render with overrides to file
kcl run main.k \
  -D config.namespace=production \
  -D config.replicas=3 \
  -o production.yaml

# Render with ingress enabled
kcl run main.k \
  -D config.ingressEnabled=True \
  -D config.ingressHost="api.sva.dev" \
  -D config.ingressTlsEnabled=True \
  -o manifests-with-ingress.yaml

# Render to environment-specific files
kcl run main.k -D config.namespace=dev -o deploy-dev.yaml
kcl run main.k -D config.namespace=staging -D config.replicas=2 -o deploy-staging.yaml
kcl run main.k -D config.namespace=prod -D config.replicas=3 -o deploy-prod.yaml
```

## Apply to Cluster

```bash
# Apply from rendered file
kubectl apply -f manifests.yaml

# Render and apply directly (pipe)
kcl run main.k | kubectl apply -f -

# Render with overrides and apply
kcl run main.k -D config.namespace=production | kubectl apply -f -

# Dry-run (client-side validation)
kcl run main.k | kubectl apply --dry-run=client -f -

# Dry-run (server-side validation)
kcl run main.k | kubectl apply --dry-run=server -f -

# Delete resources
kcl run main.k | kubectl delete -f -
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
| `config.ingressAnnotations` | {str:str} | `{}` | Ingress annotations (e.g., cert-manager) |
| `config.templatesDir` | string | `/app/templates` | Templates directory (TEMPLATES_DIR env var) |
| `config.templateProfilePath` | string | `/app/config/profile.yaml` | Template profile path (TEMPLATE_PROFILE_PATH env var) |
| `config.templateProfile` | string | `` | Template profile YAML content (mounted as file) |
| `config.port` | string | `8080` | Application port (PORT env var) |
| `config.logFormat` | string | `text` | Log format (LOG_FORMAT env var: text/json) |
| `config.debug` | bool | `False` | Enable debug mode (DEBUG env var) |
| `config.extraEnvVars` | {str:str} | `{}` | Extra environment variables for ConfigMap |
| `config.secrets` | {str:str} | `{}` | Secret key-value pairs (base64 encoded) |
| `config.serviceAccountAnnotations` | {str:str} | `{}` | ServiceAccount annotations |
| `config.labels` | {str:str} | `{}` | Additional labels for resources |
| `config.annotations` | {str:str} | `{}` | Additional annotations for resources |

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

## Dagger Deployment

Use the `kubernetes-deployment` Dagger module to apply manifests to a Kubernetes cluster.

### Render and Apply with Params

```bash
# Render manifests with KCL and save to file
kcl run main.k \
  -D config.namespace=production \
  -D config.replicas=3 \
  -D config.image="ghcr.io/stuttgart-things/claim-machinery-api:v1.0.0" \
  -o /tmp/manifests.yaml

# Apply manifests using Dagger with params
dagger -m github.com/stuttgart-things/blueprints/kubernetes-deployment@v1.44.0 call \
  apply-manifests \
  --source-files "/tmp/manifests.yaml" \
  --namespace production \
  --kube-config env:KUBECONFIG \
  --progress plain
```

### Render with Environment Variables

The `TEMPLATES_DIR` environment variable is configured via ConfigMap. Override it using `-D config.templatesDir`:

```bash
# Set custom templates directory (sets TEMPLATES_DIR in ConfigMap)
kcl run main.k \
  -D config.namespace=production \
  -D config.templatesDir="/app/custom-templates" \
  -o /tmp/manifests.yaml

# Set template profile path and content (creates additional ConfigMap + volume mount)
kcl run main.k \
  -D config.namespace=production \
  -D config.templatesDir="/app/templates" \
  -D config.templateProfilePath="/app/config/profile.yaml" \
  -D 'config.templateProfile="---\ntemplates:\n  - https://example.com/template.yaml\n"' \
  -o /tmp/manifests.yaml

# Add extra environment variables to ConfigMap
kcl run main.k \
  -D config.namespace=production \
  -D 'config.extraEnvVars={"CUSTOM_VAR": "value", "ANOTHER_VAR": "another-value"}' \
  -o /tmp/manifests.yaml

# Complete example with all environment settings
kcl run main.k \
  -D config.namespace=production \
  -D config.templatesDir="/app/templates" \
  -D config.templateProfilePath="/app/config/profile.yaml" \
  -D 'config.templateProfile="---\ntemplates:\n  - https://raw.githubusercontent.com/org/repo/main/template.yaml\n"' \
  -D config.logFormat="json" \
  -D config.debug=True \
  -o /tmp/manifests.yaml

# Apply with Dagger
dagger -m github.com/stuttgart-things/blueprints/kubernetes-deployment@v1.44.0 call \
  apply-manifests \
  --source-files "/tmp/manifests.yaml" \
  --namespace production \
  --kube-config env:KUBECONFIG \
  --progress plain
```

### Apply with Source URLs

```bash
# Apply manifests directly from URLs
dagger -m github.com/stuttgart-things/blueprints/kubernetes-deployment@v1.44.0 call \
  apply-manifests \
  --source-urls "https://raw.githubusercontent.com/stuttgart-things/claim-machinery-api/main/deployment/manifests.yaml" \
  --namespace default \
  --kube-config env:KUBECONFIG \
  --progress plain
```

### Apply with File Directory

```bash
# Apply all YAML files from a directory
dagger -m github.com/stuttgart-things/blueprints/kubernetes-deployment@v1.44.0 call \
  apply-manifests \
  --source-files "deployment/" \
  --manifest-pattern "*.yaml" \
  --namespace default \
  --kube-config env:KUBECONFIG \
  --progress plain
```

### Delete Resources

```bash
# Delete manifests using operation flag
dagger -m github.com/stuttgart-things/blueprints/kubernetes-deployment@v1.44.0 call \
  apply-manifests \
  --source-files "/tmp/manifests.yaml" \
  --operation delete \
  --namespace production \
  --kube-config env:KUBECONFIG \
  --progress plain
```

### Dagger Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `--source-files` | - | Local file or directory path containing manifests |
| `--source-urls` | - | Comma-separated URLs to manifest files |
| `--manifest-pattern` | `*.yaml` | Glob pattern for matching manifest files |
| `--operation` | `apply` | Kubernetes operation (`apply` or `delete`) |
| `--namespace` | `default` | Target Kubernetes namespace |
| `--kube-config` | - | Kubeconfig secret (use `env:KUBECONFIG` for environment variable) |
