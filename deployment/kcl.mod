[package]
name = "deploy-claim-machinery-api"
version = "0.3.0"
description = "KCL module for deploying claim-machinery-api on Kubernetes"

[dependencies]
k8s = "1.31"

[profile]
entries = [
    "main.k"
]
