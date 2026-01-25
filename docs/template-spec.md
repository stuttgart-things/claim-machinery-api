# ClaimTemplate Specification

Complete specification for Claim Machinery API templates with all available fields and options.

## Template Structure

```yaml
apiVersion: resources.stuttgart-things.com/v1alpha1
kind: ClaimTemplate
metadata:
  name: <template-name>
  title: <human-readable-title>
  description: <template-description>
  tags:
    - <tag1>
    - <tag2>
spec:
  type: <resource-type>
  source: <oci-registry-path>
  tag: <version-tag>
  parameters:
    - <parameter-definitions>
```

## Metadata Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Unique template identifier (lowercase, alphanumeric, hyphens) |
| `title` | string | ❌ | Human-readable template title |
| `description` | string | ❌ | Template purpose and functionality description |
| `tags` | array[string] | ❌ | Categorization and search tags |

## Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | ✅ | Resource type identifier |
| `source` | string | ✅ | OCI registry path for KCL module (e.g., `oci://ghcr.io/org/module`) |
| `tag` | string | ❌ | Version tag for the OCI module |
| `parameters` | array[Parameter] | ❌ | Template parameter definitions |

## Parameter Fields

### Core Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Parameter identifier (used in API calls and KCL rendering) |
| `title` | string | ✅ | Display label for UI forms |
| `description` | string | ❌ | Parameter explanation and usage guidance |
| `type` | string | ✅ | Data type: `string`, `integer`, `boolean`, `array` |
| `default` | any | ❌ | Default value (type must match `type` field) |
| `required` | boolean | ❌ | Whether parameter is mandatory (default: `false`) |

### Validation Fields

| Field | Type | Applies To | Description |
|-------|------|------------|-------------|
| `enum` | array[string] | string | List of allowed values (creates dropdown in UIs) |
| `pattern` | string | string | Regex pattern for validation |
| `minLength` | integer | string | Minimum string length |
| `maxLength` | integer | string | Maximum string length |

### UI Enhancement Fields

| Field | Type | Applies To | Description |
|-------|------|------------|-------------|
| `hidden` | boolean | all | Hide from UI forms, always use default value (platform-defined parameters) |
| `allowRandom` | boolean | enum fields | Add "🎲 Random" option to enum dropdowns for random selection |

## Parameter Types

### string

```yaml
- name: vmName
  title: VM Name
  type: string
  required: true
  default: app-server
  pattern: "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
  description: "Name of the virtual machine"
```

### integer

```yaml
- name: count
  title: VM Count
  type: integer
  default: 1
  description: "Number of VMs to create"
```

### boolean

```yaml
- name: enableBackup
  title: Enable Backup
  type: boolean
  default: true
  description: "Enable automated backup"
```

### array

```yaml
- name: tags
  title: Tags
  type: array
  default:
    - production
    - web
  description: "Resource tags"
```

### enum (dropdown)

```yaml
- name: size
  title: T-Shirt Size
  type: string
  enum:
    - S
    - M
    - L
    - XL
  default: M
  description: "VM size preset"
```

## Special Features

### Hidden Parameters

Platform engineers can define infrastructure parameters that are **always hidden from users** and use pre-configured values:

```yaml
- name: datacenter
  title: Datacenter
  type: string
  required: true
  default: /LabUL/
  hidden: true
  description: "vSphere datacenter (LabUL)"
```

**Use cases:**
- Pre-configured infrastructure settings (datacenter, resource pools)
- Secret references (Terraform tfvars secret names)
- Provider configurations
- Internal routing parameters

### Allow Random Selection

For enum parameters, enable random selection from available options:

```yaml
- name: network
  title: Network
  type: string
  required: true
  default: /LabUL/network/MGMT-10.31.101
  enum:
    - /LabUL/network/MGMT-10.31.101
    - /LabUL/network/LAB-10.31.102
    - /LabUL/network/LAB-10.31.103
    - /LabUL/network/LAB-10.31.104
  description: "vSphere network in LabUL"
  allowRandom: true
```

**Behavior:**
- CLI tools add "🎲 Random" as first option in dropdown
- When selected, a random value from enum list is chosen
- Useful for load distribution, testing, or when specific choice doesn't matter

## Complete Example Template

Based on `vspherevm-labul.yaml`:

```yaml
---
apiVersion: resources.stuttgart-things.com/v1alpha1
kind: ClaimTemplate
metadata:
  name: labul
  title: vSphere VM - LabUL
  description: Creates a vSphere VM in the LabUL lab environment
  tags:
    - vsphere
    - vm
    - crossplane
    - terraform
    - labul
spec:
  type: vspherevm
  source: oci://ghcr.io/stuttgart-things/claim-xplane-vspherevm
  tag: 0.2.0
  parameters:
    # User-facing parameters
    - name: count
      title: VM Count
      type: integer
      default: 1
      description: "Number of VMs to create"

    - name: size
      title: T-Shirt Size
      type: string
      enum:
        - S
        - M
        - L
        - XL
        - XXL
      description: "VM size preset: S (2GB/32GB/2CPU), M (4GB/64GB/4CPU), L (8GB/128GB/8CPU), XL (16GB/256GB/16CPU), XXL (32GB/512GB/32CPU)"

    - name: name
      title: VM Name
      type: string
      required: true
      default: app-server
      pattern: "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
      description: "Name of the VM and claim resource"

    - name: network
      title: Network
      type: string
      required: true
      default: /LabUL/network/MGMT-10.31.101
      enum:
        - /LabUL/network/MGMT-10.31.101
        - /LabUL/network/LAB-10.31.102
        - /LabUL/network/LAB-10.31.103
        - /LabUL/network/LAB-10.31.104
      description: "vSphere network in LabUL"
      allowRandom: true

    - name: template
      title: VM Template
      type: string
      required: true
      default: sthings-u24
      enum:
        - sthings-u22
        - sthings-u24
      description: "vSphere VM template (Ubuntu 22.04 or 24.04)"

    - name: namespace
      title: Namespace
      type: string
      default: default
      description: "Kubernetes namespace for the claim"

    # Hidden platform parameters (infrastructure pre-configuration)
    - name: folderPath
      title: Folder Path
      type: string
      required: true
      default: /LabUL/vm/intc
      description: "vSphere folder path for the VM"
      hidden: true

    - name: datacenter
      title: Datacenter
      type: string
      required: true
      default: /LabUL/
      hidden: true
      description: "vSphere datacenter (LabUL)"

    - name: datastore
      title: Datastore
      type: string
      required: true
      default: /LabUL/datastore/UL-ESX-SAS-01
      hidden: true
      description: "vSphere datastore"

    - name: resourcePool
      title: Resource Pool
      type: string
      required: true
      default: /LabUL/host/Cluster-V6.7/Resources
      hidden: true
      description: "vSphere resource pool"

    - name: tfvarsSecretName
      title: TFVars Secret Name
      type: string
      required: true
      default: vsphere-tfvars-labul
      hidden: true
      description: "Name of the secret containing Terraform variables for LabUL"

    - name: tfvarsSecretNamespace
      title: TFVars Secret Namespace
      type: string
      default: crossplane-system
      hidden: true
      description: "Namespace of the Terraform variables secret"

    - name: tfvarsSecretKey
      title: TFVars Secret Key
      type: string
      default: terraform.tfvars
      hidden: true
      description: "Key in the secret containing Terraform variables"

    - name: providerRefName
      title: Provider Config Name
      type: string
      default: default
      hidden: true
      description: "Terraform provider config name"

    - name: providerRefKind
      title: Provider Config Kind
      type: string
      default: ClusterProviderConfig
      hidden: true
      description: "Terraform provider config kind"
```

## API Usage

### List Templates

```bash
curl http://localhost:8080/api/v1/claim-templates
```

### Get Template Details

```bash
curl http://localhost:8080/api/v1/claim-templates/labul
```

### Render Template (Order)

```bash
curl -X POST http://localhost:8080/api/v1/claim-templates/labul/order \
  -H "Content-Type: application/json" \
  -d '{"parameters":{"name":"test-vm","size":"M","network":"🎲 Random"}}'
```

## CLI Tools

Two interactive CLI tools support templates:

### Local KCL CLI

```bash
cd tests/cli
./claim-cli
```

Uses local KCL installation to render templates.

### API-Connected CLI

```bash
export CLAIM_API_URL=http://localhost:8080
cd tests/cli-api
./claim-cli-api
```

Calls API for template discovery and rendering.

Both CLIs:
- ✅ Skip hidden parameters automatically
- ✅ Show "🎲 Random" option for `allowRandom` enums
- ✅ Display forms with validation
- ✅ Generate default save paths: `/tmp/{templateName}-{resourceName}.yaml`

## Best Practices

### 1. Hidden Parameters

Use `hidden: true` for:
- Infrastructure configuration (datacenter, networks, resource pools)
- Secret references
- Provider configurations
- Parameters that shouldn't change per user

### 2. Random Selection

Use `allowRandom: true` for:
- Load balancing across multiple options
- Testing with varied configurations
- Parameters where specific choice doesn't impact functionality

### 3. Validation

Always add:
- `pattern` for strings with format requirements (DNS names, IPs, etc.)
- `enum` for constrained choices
- `required: true` for mandatory parameters
- Clear `description` for user guidance

### 4. Naming Conventions

- Template `name`: lowercase-with-hyphens
- Parameter `name`: camelCase or snake_case
- `title`: User-friendly display text
- `description`: Clear, actionable guidance

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 0.2.0 | 2026-01-25 | Added `hidden` and `allowRandom` fields |
| 0.1.0 | 2026-01-09 | Initial specification |
