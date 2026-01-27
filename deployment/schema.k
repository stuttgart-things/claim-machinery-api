"""
Schema definition for claim-machinery-api deployment configuration
"""

schema ClaimMachineryAPI:
    """Configuration for claim-machinery-api Kubernetes deployment"""

    # Metadata
    name: str = "claim-machinery-api"
    namespace: str = "default"

    # Image
    image: str = "ghcr.io/stuttgart-things/claim-machinery-api:latest"
    imagePullPolicy: str = "IfNotPresent"

    # Replicas and resources
    replicas: int = 1
    cpuRequest: str = "100m"
    cpuLimit: str = "500m"
    memoryRequest: str = "128Mi"
    memoryLimit: str = "256Mi"

    # Service
    serviceType: str = "ClusterIP"
    servicePort: int = 8080
    containerPort: int = 8080

    # Ingress
    ingressEnabled: bool = False
    ingressClassName: str = "nginx"
    ingressHost: str = "claim-machinery-api.example.com"
    ingressTlsEnabled: bool = False
    ingressTlsSecretName: str = "claim-machinery-api-tls"
    ingressAnnotations: {str:str} = {}  # Ingress-specific annotations (e.g. cert-manager.io/cluster-issuer)

    # Application config
    templatesDir: str = "/app/config/"
    templateProfilePath: str = "/app/config/profile.yaml"
    templateProfile: str = ""  # YAML content for template profile (mounted as file)
    port: str = "8080"
    logFormat: str = "text"
    debug: bool = False

    # Extra environment variables
    extraEnvVars: {str:str} = {}

    # Secrets (will be base64 encoded)
    secrets: {str:str} = {}

    # Application config
    templateProfiles: [str] = []  # list of template URLs

    # ServiceAccount
    serviceAccountAnnotations: {str:str} = {}

    # Labels and annotations
    labels: {str:str} = {}
    annotations: {str:str} = {}

    check:
        replicas >= 0, "replicas must be >= 0"
        servicePort >= 1 and servicePort <= 65535, "servicePort must be valid"
        containerPort >= 1 and containerPort <= 65535, "containerPort must be valid"
        len(templateProfiles) > 0, "templateProfiles must contain at least one entry"
