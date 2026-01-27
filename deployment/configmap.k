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
        if len(config.templateProfiles) > 0:
            TEMPLATE_PROFILE_PATH: config.templateProfilePath
        if config.debug:
            DEBUG: "1"
        **config.extraEnvVars
    }
}

# ConfigMap for template profile (mounted as file)
profileConfigMap = {
    apiVersion: "v1"
    kind: "ConfigMap"
    metadata: {
        name: "{}-profile".format(config.name)
        namespace: config.namespace
        labels: labels.commonLabels
    }
    data: {
        "profile.yaml": """templates:
{}""".format(
            "\n".join([
                "  - {}".format(t)
                for t in config.templateProfiles
            ])
        )
    }
} if len(config.templateProfiles) > 0 else None
