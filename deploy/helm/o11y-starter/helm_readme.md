# o11y-starter Helm Chart

Helm chart for the **o11y-starter** service — a Go/Gin HTTP server instrumented with OpenTelemetry (traces, metrics, structured logs). The chart ships two self-contained values files: one for **minikube** and one for **AWS EKS**. Pick the file that matches your target and pass it as a single `-f` flag — no base `values.yaml` required.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
  - [minikube](#minikube)
  - [AWS EKS](#aws-eks)
- [Chart Structure](#chart-structure)
- [Values Reference](#values-reference)
  - [Image](#image)
  - [Naming](#naming)
  - [ServiceAccount](#serviceaccount)
  - [Pod Configuration](#pod-configuration)
  - [Security](#security)
  - [Service](#service)
  - [Ingress](#ingress)
  - [Gateway API](#gateway-api)
  - [Resources](#resources)
  - [Autoscaling](#autoscaling)
  - [Scheduling](#scheduling)
  - [Application Config](#application-config)
  - [ConfigMap](#configmap)
  - [Secret](#secret)
    - [type: none](#type-none)
    - [type: native](#type-native)
    - [type: external](#type-external-external-secrets-operator)
- [Secrets Workflow Guide](#secrets-workflow-guide)
- [Ingress vs Gateway API](#ingress-vs-gateway-api)
- [Upgrade Notes](#upgrade-notes)

---

## Prerequisites

| Tool | Minimum version | Notes |
|---|---|---|
| Helm | 3.12 | `helm version` |
| Kubernetes | 1.28 | Gateway API `v1` is GA from 1.28 |
| kubectl | matching cluster | — |
| **EKS only** | — | — |
| AWS Load Balancer Controller | 2.7 | Required for `ingress.className: alb` |
| AWS Gateway API Controller | 1.0 | Required for `gateway.className: amazon-vpc-lattice` |
| **External secrets only** | — | — |
| External Secrets Operator | 0.9+ | `helm repo add external-secrets https://charts.external-secrets.io` |
| **minikube only** | — | — |
| minikube | 1.32 | — |
| nginx-gateway-fabric | 1.4 | Only if using Gateway API on minikube |

---

## Quick Start

### minikube

```bash
# 1. Enable the built-in nginx ingress addon
minikube addons enable ingress

# 2. Install the chart
helm upgrade --install o11y-starter ./deploy/helm/o11y-starter \
  -f deploy/helm/o11y-starter/values-minikube.yaml \
  --namespace o11y --create-namespace

# 3. Add the host entry (if not using ingress-dns addon)
echo "$(minikube ip)  o11y-starter.local" | sudo tee -a /etc/hosts

# 4. Test
curl http://o11y-starter.local/healthz
```

To use the **Gateway API** instead of Ingress on minikube:

```bash
# Install Gateway API CRDs + nginx-gateway-fabric first
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml

helm upgrade --install o11y-starter ./deploy/helm/o11y-starter \
  -f deploy/helm/o11y-starter/values-minikube.yaml \
  --set ingress.enabled=false \
  --set gateway.enabled=true \
  --namespace o11y --create-namespace
```

### AWS EKS

```bash
# 1. Build and push the image to ECR
ECR=123456789.dkr.ecr.us-east-1.amazonaws.com/o11y-starter
docker build -t $ECR:latest .
docker push $ECR:latest

# 2. Install the chart
helm upgrade --install o11y-starter ./deploy/helm/o11y-starter \
  -f deploy/helm/o11y-starter/values-eks.yaml \
  --set image.repository=$ECR \
  --set image.tag=latest \
  --set serviceAccount.annotations."eks\.amazonaws\.com/role-arn"=arn:aws:iam::123456789:role/o11y-starter \
  --namespace o11y --create-namespace

# 3. Retrieve the ALB hostname (takes ~60 s to provision)
kubectl get ingress o11y-starter -n o11y
```

To use **VPC Lattice** (Gateway API) instead of the ALB Ingress on EKS:

```bash
helm upgrade --install o11y-starter ./deploy/helm/o11y-starter \
  -f deploy/helm/o11y-starter/values-eks.yaml \
  --set ingress.enabled=false \
  --set gateway.enabled=true \
  --set gateway.className=amazon-vpc-lattice
```

---

## Chart Structure

```
deploy/helm/o11y-starter/
├── Chart.yaml                  # chart metadata and version
├── values-minikube.yaml        # self-contained values for minikube
├── values-eks.yaml             # self-contained values for AWS EKS
└── templates/
    ├── _helpers.tpl            # named template definitions
    ├── deployment.yaml         # core workload
    ├── service.yaml            # ClusterIP Service
    ├── serviceaccount.yaml     # ServiceAccount (optional)
    ├── configmap.yaml          # non-sensitive env vars
    ├── secret.yaml             # native k8s Secret (optional)
    ├── externalsecret.yaml     # ESO ExternalSecret (optional)
    ├── ingress.yaml            # classic Ingress (optional)
    ├── gateway.yaml            # Gateway API Gateway resource (optional)
    ├── httproute.yaml          # Gateway API HTTPRoute (optional)
    ├── hpa.yaml                # HorizontalPodAutoscaler (optional)
    └── NOTES.txt               # post-install instructions
```

Each values file is **self-contained** — it carries every configurable field with its default value plus environment-specific overrides. There is no shared base file to layer on top of. To customise a deployment, edit the relevant values file or pass `--set` flags on the command line.

---

## Values Reference

All keys below are present in **both** `values-minikube.yaml` and `values-eks.yaml`. The tables note where the two files differ.

### Image

```yaml
image:
  repository: o11y-starter
  pullPolicy: IfNotPresent
  tag: ""
```

| Key | Type | Default | Description |
|---|---|---|---|
| `image.repository` | string | `o11y-starter` | Container image repository. For EKS this should be the full ECR URI, e.g. `123456789.dkr.ecr.us-east-1.amazonaws.com/o11y-starter`. |
| `image.pullPolicy` | string | `IfNotPresent` | Kubernetes image pull policy. Use `Always` in CI or when tags are mutable (e.g. `latest`). |
| `image.tag` | string | `""` | Image tag. Empty string falls back to `Chart.appVersion` (`0.1.0`). Pin this to a specific digest in production. |
| `imagePullSecrets` | list | `[]` | List of `{ name: <secret> }` objects referencing `kubernetes.io/dockerconfigjson` Secrets. Required when pulling from a private registry outside of IRSA/instance-profile scope. |

---

### Naming

```yaml
nameOverride: ""
fullnameOverride: ""
```

| Key | Type | Default | Description |
|---|---|---|---|
| `nameOverride` | string | `""` | Replaces the chart name portion in generated resource names. Useful when installing multiple instances in the same namespace. |
| `fullnameOverride` | string | `""` | Completely overrides the computed `release-chart` name. All resources (Deployment, Service, ConfigMap, Secret, Ingress, etc.) use this name. Takes precedence over `nameOverride`. |

---

### ServiceAccount

```yaml
# minikube
serviceAccount:
  create: true
  automount: true
  annotations: {}
  name: ""

# EKS
serviceAccount:
  create: true
  automount: true
  annotations:
    eks.amazonaws.com/role-arn: "arn:aws:iam::ACCOUNT_ID:role/o11y-starter-role"
  name: ""
```

| Key | Type | Default | Description |
|---|---|---|---|
| `serviceAccount.create` | bool | `true` | When `true`, a dedicated `ServiceAccount` is created for the pod. When `false`, the pod uses the `name` specified below (or `default`). |
| `serviceAccount.automount` | bool | `true` | Sets `automountServiceAccountToken` on the `ServiceAccount`. Set to `false` if the application does not need to call the Kubernetes API, which is best practice for least-privilege. |
| `serviceAccount.annotations` | map | `{}` minikube / IRSA ARN EKS | Arbitrary annotations added to the `ServiceAccount`. The primary use case on EKS is **IRSA** (IAM Roles for Service Accounts): `eks.amazonaws.com/role-arn: arn:aws:iam::ACCOUNT:role/ROLE`. This lets pods assume an IAM role without storing AWS credentials. |
| `serviceAccount.name` | string | `""` | Name of the `ServiceAccount` to use. When `create: true` and empty, the name is auto-generated from the release fullname. When `create: false`, this must reference an existing `ServiceAccount`. |

---

### Pod Configuration

```yaml
replicaCount: 1
podAnnotations: {}
podLabels: {}
```

| Key | Type | Default | Description |
|---|---|---|---|
| `replicaCount` | int | `1` | Number of pod replicas. Ignored when `autoscaling.enabled: true` — the HPA controls replica count instead. |
| `podAnnotations` | map | `{}` | Annotations placed on the pod `template` metadata. Common uses: Prometheus scrape hints (`prometheus.io/scrape: "true"`), Datadog APM injection, Linkerd injection (`linkerd.io/inject: enabled`). |
| `podLabels` | map | `{}` | Extra labels added to the pod `template` metadata alongside the standard selector labels. Useful for cost allocation, PodDisruptionBudgets, or NetworkPolicies that select by label. |

---

### Security

```yaml
podSecurityContext:
  runAsNonRoot: true
  runAsUser: 65534

securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop: ["ALL"]
```

**Pod-level security context** — applied to all containers in the pod:

| Key | Type | Default | Description |
|---|---|---|---|
| `podSecurityContext.runAsNonRoot` | bool | `true` | Kubernetes rejects the pod if the container image would run as UID 0 (root). Enforces a baseline security posture. |
| `podSecurityContext.runAsUser` | int | `65534` | UID the container process runs as. `65534` is the `nobody` user in Alpine/scratch images. Override if your image requires a specific UID. |

**Container-level security context** — applied to the single app container:

| Key | Type | Default | Description |
|---|---|---|---|
| `securityContext.allowPrivilegeEscalation` | bool | `false` | Prevents the process from gaining more privileges than its parent (blocks `setuid`, `sudo`, etc.). |
| `securityContext.readOnlyRootFilesystem` | bool | `true` | Mounts the container's root filesystem as read-only. Prevents runtime writes to the image layer. If the app needs to write temporary files, add an `emptyDir` volume mount for that path. |
| `securityContext.capabilities.drop` | list | `["ALL"]` | Drops all Linux capabilities from the container. The Go binary does not need any capabilities to serve HTTP. |

---

### Service

```yaml
service:
  type: ClusterIP
  port: 80
  targetPort: 8080
```

| Key | Type | Default | Description |
|---|---|---|---|
| `service.type` | string | `ClusterIP` | Kubernetes Service type. `ClusterIP` is the default — traffic only reachable within the cluster. Use `NodePort` for direct node access, or `LoadBalancer` to provision a cloud load balancer without an Ingress controller. |
| `service.port` | int | `80` | Port the Service listens on. This is what Ingress and HTTPRoute `backendRefs` point to. |
| `service.targetPort` | int | `8080` | Port on the container that traffic is forwarded to. Must match the application's `HTTP_ADDR` (`:8080` by default). |

---

### Ingress

```yaml
# minikube
ingress:
  enabled: true
  className: nginx
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
  hosts:
    - host: o11y-starter.local
      paths:
        - path: /
          pathType: Prefix
  tls: []

# EKS
ingress:
  enabled: true
  className: alb
  annotations:
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/target-type: ip
    alb.ingress.kubernetes.io/healthcheck-path: /healthz
  hosts:
    - host: ""
      paths:
        - path: /
          pathType: Prefix
  tls: []
```

The classic `networking.k8s.io/v1` Ingress resource. Mutually exclusive with Gateway API — enable one or the other, not both.

| Key | Type | Default | Description |
|---|---|---|---|
| `ingress.enabled` | bool | `true` (both files) | Set to `false` to disable the Ingress resource, e.g. when switching to Gateway API. |
| `ingress.className` | string | `nginx` minikube / `alb` EKS | Maps to `spec.ingressClassName`. Selects which Ingress controller processes this resource. |
| `ingress.annotations` | map | controller-specific | **minikube/nginx**: `nginx.ingress.kubernetes.io/rewrite-target: /`. **EKS/ALB**: scheme, target-type, and healthcheck-path annotations that drive ALB provisioning. |
| `ingress.hosts` | list | see above | List of host rules. Each entry has a `host` (FQDN or empty string for catch-all) and a list of `paths`. On EKS with ALB, `host` is left empty because the ALB generates its own DNS name after provisioning — read it back with `kubectl get ingress`. |
| `ingress.hosts[].paths[].path` | string | `/` | URL path prefix to match. |
| `ingress.hosts[].paths[].pathType` | string | `Prefix` | `Prefix` matches any path starting with the value. `Exact` matches only the literal path. `ImplementationSpecific` delegates matching semantics to the controller. |
| `ingress.tls` | list | `[]` | TLS configuration. Each entry takes `hosts` (list of FQDNs) and `secretName` (a `kubernetes.io/tls` Secret). Leave empty when TLS is terminated at the load balancer (typical for ALB). |

---

### Gateway API

```yaml
# minikube
gateway:
  enabled: false
  create: true
  name: ""
  existingGatewayName: ""
  className: nginx
  listeners:
    - name: http
      protocol: HTTP
      port: 80
      hostname: "o11y-starter.local"
  httpRoute:
    hostnames:
      - "o11y-starter.local"
    rules:
      - matches:
          - path:
              type: PathPrefix
              value: /

# EKS
gateway:
  enabled: false
  create: true
  name: ""
  existingGatewayName: ""
  className: amazon-vpc-lattice
  listeners:
    - name: http
      protocol: HTTP
      port: 80
      hostname: "*"
  httpRoute:
    hostnames: []
    rules:
      - matches:
          - path:
              type: PathPrefix
              value: /
```

Uses the standard `gateway.networking.k8s.io/v1` API (GA since Kubernetes 1.28). Creates a `Gateway` resource and an `HTTPRoute` that points traffic to the Service. Mutually exclusive with Ingress — enable one or the other.

| Key | Type | Default | Description |
|---|---|---|---|
| `gateway.enabled` | bool | `false` (both files) | Set to `true` to create an `HTTPRoute` (and optionally a `Gateway`). Remember to also set `ingress.enabled: false`. |
| `gateway.create` | bool | `true` | When `true`, a `Gateway` resource is created alongside the `HTTPRoute`. Set to `false` when a platform team owns a shared `Gateway` — only the `HTTPRoute` is then deployed, and it attaches to the existing Gateway via `gateway.existingGatewayName`. |
| `gateway.name` | string | `""` | Name of the `Gateway` resource to create. Defaults to the release fullname. |
| `gateway.existingGatewayName` | string | `""` | Name of an existing `Gateway` the `HTTPRoute` should attach to. Required when `gateway.create: false`. |
| `gateway.className` | string | `nginx` minikube / `amazon-vpc-lattice` EKS | `spec.gatewayClassName` — selects the Gateway controller implementation. Other common values: `istio`, `cilium`. |
| `gateway.listeners` | list | see above | List of listeners defined on the `Gateway`. Each listener specifies a `name`, `protocol` (`HTTP`/`HTTPS`/`TLS`), `port`, and optional `hostname`. `allowedRoutes.namespaces.from: Same` is set automatically, restricting attachment to routes in the same namespace. |
| `gateway.listeners[].hostname` | string | `o11y-starter.local` minikube / `"*"` EKS | The hostname this listener matches. Use `"*"` to match all hostnames — required on EKS/VPC Lattice where the DNS name is generated after provisioning. Must align with `gateway.httpRoute.hostnames`. |
| `gateway.httpRoute.hostnames` | list | `["o11y-starter.local"]` minikube / `[]` EKS | Hostnames the `HTTPRoute` matches. Must be equal to or a subset of the listener's hostname. Leave empty (`[]`) when `gateway.listeners[].hostname` is `"*"`. |
| `gateway.httpRoute.rules[].matches[].path.type` | string | `PathPrefix` | `PathPrefix` — matches any path starting with `value`. `Exact` — matches only the literal value. `RegularExpression` — matches a regex (implementation-dependent). |
| `gateway.httpRoute.rules[].matches[].path.value` | string | `/` | The path to match against incoming requests. |

---

### Resources

```yaml
resources: {}
```

| Key | Type | Default | Description |
|---|---|---|---|
| `resources` | map | `{}` | CPU and memory `requests` and `limits` for the application container. Empty means no constraints. **Always set in production** to enable the scheduler to place pods correctly and to prevent noisy-neighbour OOMKills. Autoscaling also requires `resources.requests.cpu` to compute utilisation. |

Example:

```yaml
resources:
  requests:
    cpu: 100m
    memory: 64Mi
  limits:
    cpu: 500m
    memory: 128Mi
```

---

### Autoscaling

```yaml
autoscaling:
  enabled: false
  minReplicas: 1
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80
```

| Key | Type | Default | Description |
|---|---|---|---|
| `autoscaling.enabled` | bool | `false` | Creates a `HorizontalPodAutoscaler` targeting the Deployment. When `true`, `replicaCount` is removed from the Deployment spec — the HPA controls the replica count. |
| `autoscaling.minReplicas` | int | `1` | Minimum number of running pods. Set to `2` or higher for production availability. |
| `autoscaling.maxReplicas` | int | `10` | Maximum number of pods the HPA will scale to. Set according to your node capacity and cost budget. |
| `autoscaling.targetCPUUtilizationPercentage` | int | `80` | The HPA scales up when average pod CPU utilisation across the Deployment exceeds this percentage. Requires `resources.requests.cpu` to be set — without it the HPA cannot compute utilisation. |

---

### Scheduling

```yaml
nodeSelector: {}
tolerations: []
affinity: {}
```

| Key | Type | Default | Description |
|---|---|---|---|
| `nodeSelector` | map | `{}` | Node label constraints. The pod is only scheduled onto nodes whose labels match all key/value pairs. Example: `kubernetes.io/arch: amd64`. |
| `tolerations` | list | `[]` | Allows the pod to be scheduled onto nodes with matching taints. Example: tolerate a `dedicated=observability:NoSchedule` taint on dedicated OTEL nodes. |
| `affinity` | map | `{}` | Advanced pod placement rules using `nodeAffinity`, `podAffinity`, or `podAntiAffinity`. Use `podAntiAffinity` with `requiredDuringSchedulingIgnoredDuringExecution` to spread replicas across availability zones. |

---

### Application Config

```yaml
config:
  otlpEndpoint: "otel-collector:4317"
  serviceName: "o11y-starter"
  logLevel: "info"
  httpAddr: ":8080"
```

These values are written into the `ConfigMap` and loaded into the pod via `envFrom`. They map directly to the environment variables read by `internal/config/config.go`. Both values files carry identical defaults here — override with `--set` for environment-specific collector endpoints.

| Key | Env var | Default | Description |
|---|---|---|---|
| `config.otlpEndpoint` | `OTEL_EXPORTER_OTLP_ENDPOINT` | `otel-collector:4317` | gRPC endpoint of the OpenTelemetry Collector. The service sends traces and metrics here. In-cluster format is `<service-name>:<port>`. For a Collector running outside the cluster, use a full hostname or IP. |
| `config.serviceName` | `OTEL_SERVICE_NAME` | `o11y-starter` | Service name attached to every span and metric as the `service.name` resource attribute. Appears in your tracing backend (Jaeger, Tempo, X-Ray) and metrics dashboards. |
| `config.logLevel` | `LOG_LEVEL` | `info` | Minimum log level emitted by `log/slog`. Accepted values: `debug`, `info`, `warn`, `error`. Use `debug` in development to see per-request trace/span details. |
| `config.httpAddr` | `HTTP_ADDR` | `:8080` | `host:port` the HTTP server binds to inside the container. Keep this as `:8080` to match the `EXPOSE 8080` in the Dockerfile and `service.targetPort`. |

---

### ConfigMap

```yaml
configMap:
  enabled: true
  extraData: {}
```

| Key | Type | Default | Description |
|---|---|---|---|
| `configMap.enabled` | bool | `true` | Creates a `ConfigMap` containing the `config.*` values above. The Deployment's `envFrom` references this ConfigMap, so all keys become environment variables in the pod. Disable only if you manage env vars entirely through another mechanism (e.g., a GitOps operator injecting ConfigMaps). |
| `configMap.extraData` | map | `{}` | Additional arbitrary key/value pairs merged into the ConfigMap. Use for non-sensitive feature flags, region identifiers, or any env var not covered by the core `config.*` block. Values must be strings. |

---

### Secret

```yaml
secret:
  type: none
  data: {}
  external:
    refreshInterval: "1h"
    secretStoreRef:
      name: ""
      kind: SecretStore       # minikube default
      # kind: ClusterSecretStore  # EKS default
    targetName: ""
    creationPolicy: Owner
    data: []
    dataFrom: []
```

Controls how sensitive environment variables reach the pod. Exactly one `type` is active at a time.

#### type: none

```yaml
secret:
  type: none
```

No `Secret` or `ExternalSecret` resource is created. Use this when:
- The application has no secrets at this stage.
- Secrets are injected by a sidecar, mutating webhook, or Vault agent.
- You are managing the Secret out-of-band and do not want Helm to own it.

---

#### type: native

```yaml
secret:
  type: native
  data:
    DATABASE_URL: "postgres://user:pass@host:5432/db"
    API_KEY: "supersecret"
```

Creates a standard `Opaque` Kubernetes Secret. Values are written as `stringData` (plain text in the chart; Kubernetes base64-encodes them at admission time).

| Key | Type | Default | Description |
|---|---|---|---|
| `secret.data` | map | `{}` | Key/value pairs that become Secret entries. Each key becomes an environment variable in the pod via `envFrom`. Must have at least one entry when `type: native` — the chart fails fast otherwise. |

> **Do not commit real credentials here.** Pass them at deploy time via `--set secret.data.KEY=value` or use a SOPS/Sealed Secrets workflow to encrypt values before storing in Git.

The Secret carries `helm.sh/resource-policy: keep` so that `helm uninstall` does **not** delete it, preventing accidental credential loss during a re-deploy cycle.

---

#### type: external (External Secrets Operator)

```yaml
secret:
  type: external
  external:
    refreshInterval: "1h"
    secretStoreRef:
      name: aws-secrets-manager
      kind: ClusterSecretStore
    targetName: ""
    creationPolicy: Owner
    data:
      - secretKey: DATABASE_URL
        remoteRef:
          key: myapp/prod/database
          property: url
    dataFrom:
      - extract:
          key: myapp/prod/secrets
```

Creates an `ExternalSecret` (API version `external-secrets.io/v1beta1`). The External Secrets Operator reads from the configured store and writes a standard Kubernetes `Secret`. The Deployment's `envFrom` references this auto-generated Secret by name.

| Key | Type | Default | Description |
|---|---|---|---|
| `secret.external.refreshInterval` | string | `1h` | How often ESO polls the external store for changes and reconciles the k8s Secret. Use shorter intervals (e.g. `5m`) during active rotation; longer intervals (e.g. `24h`) for stable secrets to reduce API calls and costs. |
| `secret.external.secretStoreRef.name` | string | `""` | **Required.** Name of the `SecretStore` or `ClusterSecretStore` CR that connects to your secrets backend (AWS Secrets Manager, Vault, GCP Secret Manager, Azure Key Vault, etc.). |
| `secret.external.secretStoreRef.kind` | string | `SecretStore` minikube / `ClusterSecretStore` EKS | `SecretStore` is namespace-scoped. `ClusterSecretStore` is cluster-wide and is the standard choice on EKS where a platform team manages the store credentials centrally. |
| `secret.external.targetName` | string | `""` | Name of the Kubernetes `Secret` that ESO writes. Defaults to the release fullname. The Deployment always references the correct name automatically. Override only when sharing a Secret across multiple releases. |
| `secret.external.creationPolicy` | string | `Owner` | `Owner` — ESO owns the Secret; deleted when the `ExternalSecret` is removed. `Merge` — ESO merges its keys into an existing Secret. `Orphan` — ESO creates the Secret but never deletes or updates it after initial creation. |
| `secret.external.data` | list | `[]` | Maps individual keys from the external store into Secret entries. Each item has `secretKey` (env var name in the pod), `remoteRef.key` (path in the store), optional `remoteRef.property` (a sub-field within that path, e.g. a JSON key in AWS Secrets Manager), and optional `remoteRef.version`. |
| `secret.external.dataFrom` | list | `[]` | Bulk-imports all key/value pairs at a remote path as individual Secret entries via `extract.key`. Use this when your secrets backend stores related credentials as a single JSON object — every top-level key becomes a separate env var. |

---

## Secrets Workflow Guide

### minikube — native Secret

```bash
helm upgrade --install o11y-starter ./deploy/helm/o11y-starter \
  -f deploy/helm/o11y-starter/values-minikube.yaml \
  --set secret.type=native \
  --set secret.data.DATABASE_URL="postgres://localhost:5432/dev" \
  --set secret.data.API_KEY="localkey"
```

### EKS — AWS Secrets Manager via ESO

```bash
# 1. Install ESO
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets external-secrets/external-secrets -n external-secrets --create-namespace

# 2. Create a ClusterSecretStore pointing at AWS Secrets Manager (uses IRSA)
kubectl apply -f - <<EOF
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: aws-secrets-manager
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
      auth:
        jwt:
          serviceAccountRef:
            name: o11y-starter
            namespace: o11y
EOF

# 3. Install the chart with external secrets
helm upgrade --install o11y-starter ./deploy/helm/o11y-starter \
  -f deploy/helm/o11y-starter/values-eks.yaml \
  --set secret.type=external \
  --set secret.external.secretStoreRef.name=aws-secrets-manager \
  --set secret.external.secretStoreRef.kind=ClusterSecretStore \
  --set "secret.external.data[0].secretKey=DATABASE_URL" \
  --set "secret.external.data[0].remoteRef.key=o11y-starter/prod/db" \
  --set "secret.external.data[0].remoteRef.property=url"
```

---

## Ingress vs Gateway API

| | Ingress | Gateway API |
|---|---|---|
| API | `networking.k8s.io/v1` | `gateway.networking.k8s.io/v1` |
| minikube controller | nginx (built-in addon) | nginx-gateway-fabric |
| EKS controller | AWS Load Balancer Controller (`alb`) | AWS Gateway API Controller (`amazon-vpc-lattice`) |
| Shared gateway | No — each Ingress is self-contained | Yes — `gateway.create: false` attaches an HTTPRoute to a platform-owned Gateway |
| HTTPS | `ingress.tls[]` | Listener `protocol: HTTPS` + `tls.certificateRefs` |
| Header/method routing | Annotation-based (controller-specific) | First-class `HTTPRoute` match fields |

Enable one or the other per deployment — not both simultaneously.

---

## Upgrade Notes

- **Changing `secret.type` from `native` to `external`**: the native `Secret` is not deleted by Helm (due to `resource-policy: keep`). Delete it manually before the upgrade if you want ESO to own the Secret: `kubectl delete secret <release-name> -n <namespace>`.
- **Changing hostnames** in `ingress.hosts` or `gateway.httpRoute.hostnames` is always safe — Helm patches the resource in-place.
- **Disabling `configMap.enabled`**: remove the `configMapRef` from `envFrom` before disabling, or the pod will fail to start with a missing ConfigMap error.
- **Switching from two-file installs**: if you previously ran `helm upgrade -f values.yaml -f values-eks.yaml`, drop the `-f values.yaml` flag — `values-eks.yaml` is now self-contained and the base file no longer exists.
