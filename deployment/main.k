"""
Main entry point for claim-machinery-api Kubernetes deployment
Imports all resources and exports manifests
"""

import serviceaccount
import configmap
import secret
import deploy
import service
import ingress

# Export all manifests (filter out None values)
manifests = [
    m for m in [
        serviceaccount.serviceAccount
        configmap.configMap
        secret.secret
        deploy.deployment
        service.service
        ingress.ingress
    ] if m
]
