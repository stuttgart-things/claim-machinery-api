"""
Ingress for claim-machinery-api
"""

import schema
import labels

config = labels.config

ingress = {
    apiVersion: "networking.k8s.io/v1"
    kind: "Ingress"
    metadata: {
        name: config.name
        namespace: config.namespace
        labels: labels.commonLabels
        annotations: {
            "nginx.ingress.kubernetes.io/ssl-redirect": "true" if config.ingressTlsEnabled else "false"
            **config.annotations
        }
    }
    spec: {
        ingressClassName: config.ingressClassName
        if config.ingressTlsEnabled:
            tls: [
                {
                    hosts: [config.ingressHost]
                    secretName: config.ingressTlsSecretName
                }
            ]
        rules: [
            {
                host: config.ingressHost
                http: {
                    paths: [
                        {
                            path: "/"
                            pathType: "Prefix"
                            backend: {
                                service: {
                                    name: config.name
                                    port: {
                                        number: config.servicePort
                                    }
                                }
                            }
                        }
                    ]
                }
            }
        ]
    }
} if config.ingressEnabled else None
