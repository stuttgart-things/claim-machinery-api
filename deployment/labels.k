"""
Common labels for claim-machinery-api resources
"""

import schema

# Config instance (can be overridden)
config: schema.ClaimMachineryAPI = schema.ClaimMachineryAPI {}

# Common labels applied to all resources
commonLabels = {
    "app.kubernetes.io/name": config.name
    "app.kubernetes.io/instance": config.name
    "app.kubernetes.io/component": "api"
    "app.kubernetes.io/part-of": "claim-machinery"
    **config.labels
}

# Selector labels for pods
selectorLabels = {
    "app.kubernetes.io/name": config.name
    "app.kubernetes.io/instance": config.name
}
