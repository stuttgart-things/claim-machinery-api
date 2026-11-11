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
| `pattern` | `string` | no | Regex pattern for string validation. Checked by the API on order (400). |
| `minLength` | `*int` | no | Minimum string length. Checked by the API on order (400). |
| `maxLength` | `*int` | no | Maximum string length. Checked by the API on order (400). |
| `showWhen` | `map[string][]string` | no | The parameter applies only when every named parameter has one of the listed values (see [Parameter Conditions](#parameter-conditions)). |
| `enumWhen` | `map[string]map[string][]string` | no | Allowed values depending on another parameter's value (see [Parameter Conditions](#parameter-conditions)). |

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

Since #135 the API checks the parameters of an order (`POST …/order`,
`…/order-pr`, the web UI) **before** KCL runs and answers **400** with the
fields that are wrong:

```json
{"error": "invalid parameters", "fields": {"size": "use_case dev allows large or xlarge, got small"}}
```

| Rule | API (before KCL) |
|------|------------------|
| `showWhen` | A value sent for an inactive parameter is ignored (dropped); its default, if any, still reaches KCL |
| `required` | An active required parameter needs a non-empty value, unless it has a `default` or `valueFrom` |
| `enum` / `enumWhen` | A value sent (each item of a list) must be allowed; with `enumWhen` the default is checked too |
| `pattern`, `minLength`, `maxLength` | Checked for string values sent |
| `type: boolean` | A value sent must be `true` / `false` (JSON bool or string) |

An empty string is "not set": it is not checked against enum/pattern and
the default or KCL decides, as before.

A failing `assert` (or schema `check`) of the KCL module answers **400** as
well, with only the assert's message — no stack trace, no ANSI codes:

```json
{"error": "use_case dev needs size large or xlarge, got small"}
```

Any other render failure (OCI pull, syntax, runtime error) stays **500**
(order as PR: 422).

### Parameter Conditions

Two optional fields per parameter (#135) let a template say *when* a
parameter applies and *which values* it allows given another parameter.
Clients that do not know them ignore them and show every field, as before.

```yaml
parameters:
  - name: use_case
    type: string
    default: baseos
    enum: [baseos, kind, dev, rke2, github-runner]

  - name: size
    type: string
    default: medium
    enum: [small, medium, large, xlarge]
    enumWhen:                 # <controlling param>: {<its value>: [allowed values]}
      use_case:
        dev: [large, xlarge]
        github-runner: [large, xlarge]
        rke2: [medium, large, xlarge]

  - name: kind_workers
    type: string
    default: "1"
    showWhen:                 # <controlling param>: [values that show this one]
      use_case: [kind]

  - name: dev_features
    type: array
    multiselect: true
    enum: [toolchain, docker, go, kind]
    showWhen:
      use_case: [dev]
```

**`showWhen: {<param>: [<value>, …]}`**
- The parameter is *active* only when every key matches (AND); a key
  matches when the controlling parameter's value is one of the listed
  values (OR). A multiselect controlling parameter matches when one of its
  selected values is listed.
- Inactive: clients hide it; the API ignores (drops) a value sent for it.
  Its `default`, if any, still goes to KCL, exactly as when a client never
  sent the field, so modules see what they saw before.
- An inactive parameter has no value for other conditions (chains resolve:
  `dev_kind_workers` with `showWhen: {dev_features: [kind]}` is hidden
  while `dev_features` is).

**`enumWhen: {<param>: {<value>: [<allowed>, …]}}`**
- While the controlling parameter has one of the listed values, the
  parameter allows only the given values. A value that is not listed
  leaves `enum` as it is. Several controlling parameters intersect.
- The lists should be subsets of `enum`; the loader logs a warning
  otherwise (and for a condition naming an unknown parameter).
- The default must be allowed too: with `use_case: dev` and no `size`
  sent, the default `medium` is refused (`… got medium (default)`), so
  clients should pick the first allowed value when the current one drops
  out.

**Common rules**
- The value a condition looks at is the one sent, else the controlling
  parameter's `default`. Values compare as strings: a boolean matches
  `"true"` / `"false"` (YAML `[true]` decodes as `"true"`).
- `hidden: true` parameters can control others through their default.
- Layout order is the order of `spec.parameters`; put controlling
  parameters (e.g. `use_case`) before the fields they control.
- The reference implementation is `internal/claimtemplate/conditions.go`
  (`ActiveParams`, `Parameter.AllowedValues`, `CheckParams`); the web UI's
  form script follows the same rules.

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
| `internal/api/handlers.go` | API parameter handling (validation, merge with defaults, error mapping) |
| `internal/claimtemplate/conditions.go` | `showWhen` / `enumWhen` evaluation and order validation (`CheckParams`) |
| `internal/render/kclerror.go` | KCL `assert` failures to `AssertError` (400) |
