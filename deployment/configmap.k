"""
ConfigMap for claim-machinery-api
"""

import schema
import labels

config = labels.config

configMap = {
    apiVersion: "v1"
    kind: "ConfigMap"
    metadata: {
        name: config.name
        namespace: config.namespace
        labels: labels.commonLabels
    }
    data: {
        TEMPLATES_DIR: config.templatesDir
        PORT: config.port
        LOG_FORMAT: config.logFormat
        if config.templateProfilePath:
            TEMPLATE_PROFILE_PATH: config.templateProfilePath
        if config.debug:
            DEBUG: "1"
        **config.extraEnvVars
    }
}
