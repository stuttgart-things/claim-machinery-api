# ClaimTemplate Schema Reference

Schema reference for ClaimTemplate YAML files used by the claim-machinery-api.
This is the single source of truth for template authors.

## 1. Top-Level Template Structure

```yaml
apiVersion: sthings.io/v1alpha1     # or resources.stuttgart-things.com/v1alpha1
kind: ClaimTemplate
metadata:
  name: string           # required - unique template identifier
  title: string          # optional - human-readable title
  description: string    # optional - template description
  tags: [string]         # optional - searchable tags
  labels:                # accepted in YAML but not processed by the API
    key: value
spec:
  type: string           # required - resource type identifier
  source: string         # required - OCI source (e.g. oci://ghcr.io/...)
  tag: string            # optional - OCI tag/version
  parameters: []         # required - list of Parameter definitions
```

### Metadata Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | yes | Unique template identifier (lowercase, alphanumeric, hyphens) |
| `title` | `string` | no | Human-readable template title |
| `description` | `string` | no | Template purpose and functionality description |
| `tags` | `[string]` | no | Categorization and search tags |
| `labels` | `map[string]string` | no | Key-value labels. **Note:** accepted in YAML but the `ClaimTemplateMetadata` struct does not define this field, so labels are silently ignored by the API. |

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | `string` | yes | Resource type identifier (e.g. `vspherevm`, `database`, `volumeclaim`) |
| `source` | `string` | yes | OCI registry path for the KCL module (e.g. `oci://ghcr.io/org/module`) |
| `tag` | `string` | no | Version tag for the OCI module |
| `parameters` | `[Parameter]` | yes | Template parameter definitions |

## 2. Parameter Fields

All fields from the `Parameter` struct (`internal/claimtemplate/claimtemplate.go`):

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | yes | Parameter identifier, used as the KCL `-D` key and API JSON key |
| `title` | `string` | yes | Display label shown in CLI forms and UIs |
| `description` | `string` | no | Help text shown alongside the field |
| `type` | `string` | yes | One of: `string`, `boolean`, `integer`, `number`, `array` |
| `default` | `any` | no | Default value used when the user does not provide one. Type should match the `type` field. |
| `required` | `bool` | no | Whether the parameter must be provided (default: `false`). CLI appends `*` to required field titles. |
| `enum` | `[string]` | no | List of allowed values. Renders as a select dropdown in the CLI. |
| `hidden` | `bool` | no | If `true`, the parameter is not shown in forms and always uses its default value silently. |
| `allowRandom` | `bool` | no | If `true`, prepends a "Random" option to enum dropdowns. Only meaningful when `enum` is set. |
| `multiselect` | `bool` | no | If `true`, allows selecting multiple values from `enum` options. Renders as a multi-select checkbox list. Only meaningful when `enum` is set. |
| `pattern` | `string` | no | Regex pattern for string validation. Shown in CLI description but not enforced at runtime yet. |
| `minLength` | `*int` | no | Minimum string length. Available in schema; validation not enforced at runtime yet. |
| `maxLength` | `*int` | no | Maximum string length. Available in schema; validation not enforced at runtime yet. |

## 3. Parameter Types and Behavior

| Type | Modifiers | CLI Widget | API JSON Type | KCL `-D` Format |
|------|-----------|-----------|---------------|-----------------|
| `string` | _(none)_ | Text input | `"value"` | `-D key=value` |
| `string` | `enum` | Select dropdown | `"value"` | `-D key=value` |
| `string` | `enum` + `allowRandom` | Select dropdown (with "Random" option) | `"value"` | `-D key=value` (resolved value) |
| `string` | `enum` + `multiselect` | Multi-select checkboxes | `["a","b"]` | `-D 'key=["a","b"]'` |
| `boolean` | _(none)_ | Select (true/false) | `true` / `false` | `-D key=true` |
| `integer` | _(none)_ | Number input (validated) | `123` | `-D key=123` |
| `number` | _(none)_ | Number input | `123` | `-D key=123` |
| `array` | _(none)_ | Textarea / default list | `["a","b"]` | `-D 'key=["a","b"]'` |
| `array` | `enum` + `multiselect` | Multi-select checkboxes | `["a","b"]` | `-D 'key=["a","b"]'` |

**Notes:**
- `integer` and `number` both represent numeric values. The CLI validates `integer` inputs with `strconv.Atoi`. The `number` type is accepted by the Go struct but the CLI currently handles it through the default (string input) path.
- When `allowRandom` is set and the user selects "Random", the CLI resolves it to a random value from `enum` before rendering. The KCL `-D` flag receives the resolved value, not the marker.
- Multiselect values are passed as `[]string` internally and formatted as KCL list literals (e.g. `["a","b"]`).

## 4. Validation Rules

| Rule | CLI | API | KCL |
|------|-----|-----|-----|
| `required` | Appends `*` to title; no hard block | Not enforced | Not enforced |
| `enum` | Enforced via dropdown (user can only pick listed values) | Not validated | Not enforced |
| `pattern` | Displayed in field description; **not enforced at runtime** (noted as TODO) | Not validated | Not enforced |
| `minLength` | Available in schema; **not enforced at runtime** | Not validated | Not enforced |
| `maxLength` | Available in schema; **not enforced at runtime** | Not validated | Not enforced |
| `type: integer` | Validated with `strconv.Atoi` (rejects non-numeric input) | Not validated | KCL schema may enforce |
| `type: boolean` | Enforced via select (true/false only) | Not validated | KCL schema may enforce |

**Summary:** The CLI is the primary validation surface today. The API merges user-supplied parameters with defaults (`app.BuildParameterValues`) but does not validate types, patterns, or required fields. KCL schemas may enforce their own constraints during rendering.

## 5. Special Features

### Hidden Parameters

Parameters with `hidden: true` are skipped during CLI form rendering. Their `default` value is used automatically. This is useful for platform-defined infrastructure values that users should not modify.

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
- Infrastructure configuration (datacenter, resource pools, datastores)
- Secret references (Terraform tfvars secret names)
- Provider configurations
- Internal routing parameters

### Random Selection

Parameters with `allowRandom: true` and an `enum` list get a "Random" option prepended to the dropdown. When selected, the CLI picks a random value from `enum` before passing it to KCL.

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
  allowRandom: true
```

**Behavior:**
- CLI prepends "Random" as the first option in the dropdown
- On selection, a random value from `enum` is chosen and logged
- The resolved value (not the marker) is sent to KCL

### Multiselect

Parameters with `multiselect: true` and an `enum` list render as a multi-select checkbox form. The user can pick one or more values. The result is a `[]string` passed to KCL as a list literal.

```yaml
- name: playbooks
  title: Playbooks
  description: Select one or more playbooks to run
  type: array
  multiselect: true
  enum:
    - baseos
    - deploy-docker
    - configure-network
  default:
    - baseos
```

**Behavior:**
- CLI renders a `huh.MultiSelect` widget with checkboxes
- Default values are pre-selected
- Selected values are passed as `[]string`
- KCL receives `-D 'playbooks=["baseos","deploy-docker"]'`

## 6. Full Example Templates

### String, Enum, Pattern, and Validation

Demonstrates `string` parameters with `enum`, `pattern`, `minLength`, and `maxLength`:

```yaml
---
apiVersion: sthings.io/v1alpha1
kind: ClaimTemplate
metadata:
  name: postgresql
  title: PostgreSQL Database Claim
  description: Creates a PostgreSQL database via Crossplane
  tags:
    - database
    - crossplane
    - postgresql
spec:
  type: database
  source: oci://ghcr.io/stuttgart-things/claim-xplane-volumeclaim
  tag: 0.1.1
  parameters:
    - name: instanceClass
      title: Instance Class
      type: string
      required: true
      default: db.t3.micro
      enum:
        - db.t3.micro
        - db.t3.small
        - db.t3.medium
        - db.m5.large

    - name: namespace
      title: Kubernetes Namespace
      type: string
      required: true
      default: databases
      pattern: "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
      minLength: 1
      maxLength: 63

    - name: databaseName
      title: Database Name
      type: string
      required: true
      default: mydb
      pattern: "^[a-zA-Z_][a-zA-Z0-9_]*$"
      minLength: 1
      maxLength: 63

    - name: enableEncryption
      title: Enable Encryption
      type: boolean
      default: true

    - name: tags
      title: Resource Tags
      type: array
```

### Array, Enum, and Multiselect

Demonstrates `array` type with `multiselect`:

```yaml
---
apiVersion: resources.stuttgart-things.com/v1alpha1
kind: ClaimTemplate
metadata:
  name: multiselect-test
  title: Multiselect Test Template
  description: Template for testing multiselect parameter support
spec:
  type: test
  source: oci://ghcr.io/stuttgart-things/test
  tag: 0.1.0
  parameters:
    - name: playbooks
      title: Playbooks
      description: Select one or more playbooks to run
      type: array
      multiselect: true
      enum:
        - baseos
        - deploy-docker
        - configure-network
      default:
        - baseos

    - name: name
      title: Resource Name
      type: string
      default: test-resource
```

### Hidden Parameters and AllowRandom

Demonstrates `hidden`, `allowRandom`, and a mix of user-facing and platform parameters:

```yaml
---
apiVersion: resources.stuttgart-things.com/v1alpha1
kind: ClaimTemplate
metadata:
  name: vspherevm-labul
  title: vSphere VM - LabUL
  description: Creates a vSphere VM in the LabUL lab environment
  tags:
    - vsphere
    - vm
    - crossplane
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

    - name: size
      title: T-Shirt Size
      type: string
      enum: [S, M, L, XL, XXL]

    - name: name
      title: VM Name
      type: string
      required: true
      default: app-server
      pattern: "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"

    - name: network
      title: Network
      type: string
      required: true
      default: /LabUL/network/MGMT-10.31.101
      enum:
        - /LabUL/network/MGMT-10.31.101
        - /LabUL/network/LAB-10.31.102
        - /LabUL/network/LAB-10.31.103
      allowRandom: true

    # Hidden platform parameters
    - name: datacenter
      title: Datacenter
      type: string
      required: true
      default: /LabUL/
      hidden: true

    - name: resourcePool
      title: Resource Pool
      type: string
      required: true
      default: /LabUL/host/Cluster-V6.7/Resources
      hidden: true

    - name: tfvarsSecretName
      title: TFVars Secret Name
      type: string
      required: true
      default: vsphere-tfvars-labul
      hidden: true
```

## Source Reference

| File | Purpose |
|------|---------|
| `internal/claimtemplate/claimtemplate.go` | Go type definitions (`ClaimTemplate`, `Parameter` structs) |
| `cmd/render.go` | CLI form rendering (hidden, allowRandom, multiselect behavior) |
| `internal/render/kcl.go` | KCL `-D` flag formatting and list literal rendering |
| `internal/app/renderer.go` | Default value building and template rendering |
| `internal/api/handlers.go` | API parameter handling (merge with defaults, no validation) |
