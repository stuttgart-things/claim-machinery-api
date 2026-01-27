"""
Common labels for claim-machinery-api resources
"""

import schema

# Get command-line options
_image = option("config.image")
_namespace = option("config.namespace")
_ingressEnabled = option("config.ingressEnabled")
_ingressHost = option("config.ingressHost")
_ingressTlsEnabled = option("config.ingressTlsEnabled")
_ingressAnnotations = option("config.ingressAnnotations") or {}
_templateProfile = option("config.templateProfile")
_templateProfiles = option("config.templateProfiles") or []
_extraEnvVars = option("config.extraEnvVars") or {}

# Config instance with command-line overrides
config: schema.ClaimMachineryAPI = schema.ClaimMachineryAPI {
    if _image:
        image = _image
    if _namespace:
        namespace = _namespace
    if _ingressEnabled:
        ingressEnabled = _ingressEnabled
    if _ingressHost:
        ingressHost = _ingressHost
    if _ingressTlsEnabled:
        ingressTlsEnabled = _ingressTlsEnabled
    if _ingressAnnotations:
        ingressAnnotations = _ingressAnnotations
    if _templateProfile:
        templateProfile = _templateProfile
    if _templateProfiles:
        templateProfiles = _templateProfiles
    if _extraEnvVars:
        extraEnvVars = _extraEnvVars
}

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
