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

# Enable Gateway API HTTPRoute (alternative to ingress)
kcl run main.k \
  -D config.httpRouteEnabled=True \
  -D config.httpRouteParentRefName="main-gateway" \
  -D config.httpRouteHostname="api.example.com"

# HTTPRoute with gateway in different namespace
kcl run main.k \
  -D config.httpRouteEnabled=True \
  -D config.httpRouteParentRefName="main-gateway" \
  -D config.httpRouteParentRefNamespace="gateway-system" \
  -D config.httpRouteHostname="api.example.com"

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
| `config.httpRouteEnabled` | bool | `False` | Enable Gateway API HTTPRoute |
| `config.httpRouteParentRefName` | string | `` | Gateway name (required when httpRouteEnabled) |
| `config.httpRouteParentRefNamespace` | string | `` | Gateway namespace (optional) |
| `config.httpRouteHostname` | string | `` | HTTPRoute hostname (defaults to ingressHost) |
| `config.httpRouteAnnotations` | {str:str} | `{}` | HTTPRoute annotations |
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
| `httproute.k` | HTTPRoute resource (Gateway API) |
| `main.k` | Entry point |

## Gateway API Example

The deployment supports [Gateway API](https://gateway-api.sigs.k8s.io/) HTTPRoute as an alternative to Ingress. This requires a Gateway resource deployed on the cluster (e.g. with Cilium).

### Example Gateway (Cilium)

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: whatever-gateway
  namespace: default
spec:
  gatewayClassName: cilium
  listeners:
    - name: https
      port: 443
      protocol: HTTPS
      hostname: "*.whatever.sthings-vsphere.labul.sva.de"
      tls:
        mode: Terminate
        certificateRefs:
          - kind: Secret
            name: wildcard-whatever-tls
      allowedRoutes:
        namespaces:
          from: All
    - name: http
      port: 80
      protocol: HTTP
      hostname: "*.whatever.sthings-vsphere.labul.sva.de"
      allowedRoutes:
        namespaces:
          from: All
```

> **Note:** Set `allowedRoutes.namespaces.from: All` on the Gateway listeners to allow HTTPRoutes from other namespaces. With `from: Same` (default), only routes in the Gateway's own namespace are accepted.

### Deploy with HTTPRoute

```bash
# Render and apply with Gateway API HTTPRoute
kcl run main.k \
  -D config.namespace=claim-machinery \
  -D config.httpRouteEnabled=True \
  -D config.httpRouteParentRefName="whatever-gateway" \
  -D config.httpRouteParentRefNamespace="default" \
  -D config.httpRouteHostname="claim-api.whatever.sthings-vsphere.labul.sva.de" \
  -D 'config.templateProfiles=["https://raw.githubusercontent.com/stuttgart-things/kcl/refs/heads/main/crossplane/claim-xplane-volumeclaim/templates/volumeclaim-simple.yaml","https://raw.githubusercontent.com/stuttgart-things/kcl/refs/heads/main/crossplane/claim-xplane-harborproject/templates/harborproject-simple.yaml"]' \
  -D 'config.extraEnvVars={"TEMPLATE_PROFILE_PATH": "/app/config/profile.yaml"}' \
  | yq '.manifests | (.[] | splitDoc)' - \
  | kubectl apply -f -
```

### Verify HTTPRoute

```bash
# Check HTTPRoute status (should show Accepted: True)
kubectl -n claim-machinery get httproute claim-machinery-api

# Check HTTPRoute details
kubectl -n claim-machinery get httproute claim-machinery-api -o yaml | yq '.status.parents[0].conditions'

# Test endpoint
curl http://claim-api.whatever.sthings-vsphere.labul.sva.de/health
```

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

## Kustomize Base Rendering

The KCL Dagger module provides a `render-kustomize-base` function that renders KCL output into a [kustomize](https://kustomize.io/) base directory. This lets you customize manifests at deploy time using standard kustomize overlays — without re-running KCL.

### Generate the Base

```bash
# Render kustomize base from local KCL source + parameters file
dagger call -m github.com/stuttgart-things/dagger/kcl render-kustomize-base \
  --source ./deployment \
  --parameters-file ./tests/kcl-deploy-profile.yaml \
  export --path=/tmp/kustomize-base

# Render from OCI source
dagger call -m github.com/stuttgart-things/dagger/kcl render-kustomize-base \
  --oci-source ghcr.io/stuttgart-things/kcl-claim-machinery-api \
  --parameters 'config.namespace=production,config.replicas=3' \
  export --path=/tmp/kustomize-base
```

### Output Structure

```
base/
  kustomization.yaml
  configmap-claim-machinery-api.yaml
  configmap-claim-machinery-api-profile.yaml
  deployment-claim-machinery-api.yaml
  ingress-claim-machinery-api.yaml
  namespace-claim-machinery.yaml
  service-claim-machinery-api.yaml
  serviceaccount-claim-machinery-api.yaml
```

Files are named `kind-name.yaml` (lowercased, sanitized). The generated `kustomization.yaml` lists all resource files.

### Validate

```bash
# Dry-run with kubectl
kubectl kustomize /tmp/kustomize-base/

# Or apply directly
kubectl apply -k /tmp/kustomize-base/
```

### Kustomize Overlays

Once you have the base, create overlay directories to customize per environment.

#### Recommended Directory Layout

```
deployment/
  kustomize/
    base/                          # output of render-kustomize-base
      kustomization.yaml
      deployment-claim-machinery-api.yaml
      service-claim-machinery-api.yaml
      ...
    overlays/
      dev/
        kustomization.yaml
        patch-ingress-host.yaml
      staging/
        kustomization.yaml
        patch-ingress-host.yaml
        patch-replicas.yaml
      production/
        kustomization.yaml
        patch-ingress-host.yaml
        patch-replicas.yaml
        patch-resources.yaml
```

#### Patch Ingress Host (JSON Patch)

Change the ingress hostname per environment:

`overlays/production/kustomization.yaml`:
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../base

patches:
  - path: patch-ingress-host.yaml
    target:
      kind: Ingress
      name: claim-machinery-api
```

`overlays/production/patch-ingress-host.yaml`:
```yaml
- op: replace
  path: /spec/rules/0/host
  value: claim-api.production.example.com
- op: replace
  path: /spec/tls/0/hosts/0
  value: claim-api.production.example.com
- op: replace
  path: /spec/tls/0/secretName
  value: claim-machinery-api-tls-production
```

```bash
kubectl apply -k overlays/production/
```

#### Scale Replicas and Resources (Strategic Merge Patch)

`overlays/production/patch-replicas.yaml`:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: claim-machinery-api
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: claim-machinery-api
          resources:
            requests:
              cpu: 250m
              memory: 256Mi
            limits:
              cpu: "1"
              memory: 512Mi
```

`overlays/production/kustomization.yaml`:
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../base

patches:
  - path: patch-ingress-host.yaml
    target:
      kind: Ingress
      name: claim-machinery-api
  - path: patch-replicas.yaml
```

#### Override ConfigMap Values

Change environment variables (port, log format, debug) without touching the deployment:

`overlays/staging/patch-configmap.yaml`:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: claim-machinery-api
data:
  PORT: "9090"
  LOG_FORMAT: json
  DEBUG: "true"
```

#### Switch from Ingress to Gateway API HTTPRoute

Remove the ingress resource and add an HTTPRoute in the overlay:

`overlays/gateway/kustomization.yaml`:
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../base
  - httproute.yaml

patches:
  - target:
      kind: Ingress
      name: claim-machinery-api
    patch: |
      $patch: delete
      apiVersion: networking.k8s.io/v1
      kind: Ingress
      metadata:
        name: claim-machinery-api
```

`overlays/gateway/httproute.yaml`:
```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: claim-machinery-api
  namespace: claim-machinery
spec:
  parentRefs:
    - name: whatever-gateway
      namespace: default
  hostnames:
    - claim-api.whatever.sthings-vsphere.labul.sva.de
  rules:
    - backendRefs:
        - name: claim-machinery-api
          port: 8080
```

```bash
kubectl apply -k overlays/gateway/
```

#### Change Namespace

Kustomize can override the namespace for all resources:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../base
namespace: my-custom-namespace
```

#### Add Labels or Annotations

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../base

commonLabels:
  environment: production
  team: platform

commonAnnotations:
  owner: platform-team@example.com
```
