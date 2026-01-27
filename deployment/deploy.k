"""
Deployment for claim-machinery-api
"""

import schema
import labels

config = labels.config

deployment = {
    apiVersion: "apps/v1"
    kind: "Deployment"
    metadata: {
        name: config.name
        namespace: config.namespace
        labels: labels.commonLabels
        if config.annotations:
            annotations: config.annotations
    }
    spec: {
        replicas: config.replicas
        selector: {
            matchLabels: labels.selectorLabels
        }
        template: {
            metadata: {
                labels: labels.commonLabels
                if config.annotations:
                    annotations: config.annotations
            }
            spec: {
                serviceAccountName: config.name
                containers: [
                    {
                        name: config.name
                        image: config.image
                        imagePullPolicy: config.imagePullPolicy
                        ports: [
                            {
                                name: "http"
                                containerPort: config.containerPort
                                protocol: "TCP"
                            }
                        ]
                        envFrom: [
                            {
                                configMapRef: {
                                    name: config.name
                                }
                            }
                        ] + ([
                            {
                                secretRef: {
                                    name: config.name
                                }
                            }
                        ] if config.secrets else [])
                        resources: {
                            requests: {
                                cpu: config.cpuRequest
                                memory: config.memoryRequest
                            }
                            limits: {
                                cpu: config.cpuLimit
                                memory: config.memoryLimit
                            }
                        }
                        livenessProbe: {
                            httpGet: {
                                path: "/health"
                                port: "http"
                            }
                            initialDelaySeconds: 10
                            periodSeconds: 15
                            timeoutSeconds: 5
                            failureThreshold: 3
                        }
                        readinessProbe: {
                            httpGet: {
                                path: "/health"
                                port: "http"
                            }
                            initialDelaySeconds: 5
                            periodSeconds: 10
                            timeoutSeconds: 3
                            failureThreshold: 3
                        }
                        securityContext: {
                            readOnlyRootFilesystem: True
                            runAsNonRoot: True
                            allowPrivilegeEscalation: False
                            capabilities: {
                                drop: ["ALL"]
                            }
                        }
                        volumeMounts: [
                            {
                                name: "tmp"
                                mountPath: "/tmp"
                            }
                        ] + ([
                            {
                                name: "profile-config"
                                mountPath: "/app/config"
                                readOnly: True
                            }
                        ] if len(config.templateProfiles) > 0 else [])
                    }
                ]
                securityContext: {
                    runAsNonRoot: True
                    seccompProfile: {
                        type: "RuntimeDefault"
                    }
                }
                volumes: [
                    {
                        name: "tmp"
                        emptyDir: {}
                    }
                ] + ([
                    {
                        name: "profile-config"
                        configMap: {
                            name: "{}-profile".format(config.name)
                        }
                    }
                ] if len(config.templateProfiles) > 0 else [])
            }
        }
    }
}
