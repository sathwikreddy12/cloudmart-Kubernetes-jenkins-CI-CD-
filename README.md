# CloudMart — Production Microservices on AWS EKS

A production-grade microservices platform deployed on AWS EKS, implementing industry-standard DevOps practices including containerization, CI/CD automation, infrastructure-as-code, and observability.

---

## Architecture Overview

```
                        ┌─────────────────────────────────────────────────────┐
                        │                   AWS VPC (192.168.0.0/16)          │
                        │                   EKS Cluster: cloudmart             │
                        │                                                      │
Internet ──► AWS CLB ──► Nginx Ingress Controller (2 replicas)                │
                        │        │              │              │               │
                        │  /api/users    /api/products   /api/orders           │
                        │        │              │              │               │
                        │  user-service   product-service  order-service       │
                        │  (Node.js)      (Python/FastAPI)  (Go)               │
                        │  2 replicas     2 replicas        2 replicas         │
                        │        │              │              │               │
                        │  PostgreSQL     MongoDB          Redis               │
                        │  (PVC)          (PVC)            (PVC)               │
                        │                                                      │
                        │  kube-state-metrics  HPA  Cluster Autoscaler        │
                        └─────────────────────────────────────────────────────┘
                                          │ VPC Peering
                        ┌─────────────────────────────────────────────────────┐
                        │               AWS VPC (172.31.0.0/16)               │
                        │               EC2 t3.small (Workstation)            │
                        │                                                      │
                        │  Jenkins        VictoriaMetrics    Grafana           │
                        │  (CI/CD)        (Metrics Store)    (Dashboards)      │
                        │                                                      │
                        │  Loki + Promtail    AlertManager    cAdvisor         │
                        └─────────────────────────────────────────────────────┘
```

---

## Tech Stack

| Layer | Technology |
|---|---|
| Container Orchestration | AWS EKS (Kubernetes 1.31) |
| Container Runtime | containerd 2.2.3 |
| Infrastructure Provisioning | eksctl, Terraform (Phase 8) |
| CI/CD | Jenkins (Declarative Pipeline) |
| Container Registry | AWS ECR |
| Ingress | Nginx Ingress Controller |
| Metrics | VictoriaMetrics + kube-state-metrics |
| Visualization | Grafana |
| Log Aggregation | Loki + Promtail |
| Alerting | Alertmanager |
| IaC | Terraform |

---

## Microservices

| Service | Language | Port | Responsibilities |
|---|---|---|---|
| user-service | Node.js 20 | 3001 | Registration, login, JWT auth |
| product-service | Python 3.12 / FastAPI | 3002 | Product catalog CRUD |
| order-service | Go 1.22 | 3003 | Order placement and tracking |

All services use **multi-stage Dockerfiles** — final images range from 8MB (Go) to 90MB (Python), built on Alpine/slim base images with non-root users.

---

## Repository Structure

```
cloudmart/
├── user_service/
│   ├── src/index.js
│   ├── package.json
│   └── Dockerfile
├── product_service/
│   ├── main.py
│   ├── requirements.txt
│   └── Dockerfile
├── order_service/
│   ├── main.go
│   ├── go.mod
│   └── Dockerfile
├── k8s/
│   ├── user-service.yaml
│   ├── product-service.yaml
│   ├── order-service.yaml
│   ├── ingress.yaml
│   ├── hpa.yaml
│   └── kube-state-metrics-nodeport.yaml
├── eks-cluster.yaml
└── Jenkinsfile
```

---

## Infrastructure

### EKS Cluster

Provisioned via `eksctl` using a declarative YAML config:

- **Kubernetes version:** 1.31
- **Node group:** t3.medium, min 1 / max 3 / desired 2
- **Node OS:** Amazon Linux 2023
- **Auto-provisioned:** VPC, public/private subnets across 3 AZs, IGW, NAT Gateway, route tables, IAM roles, OIDC provider, security groups

```bash
eksctl create cluster -f eks-cluster.yaml
```

### Networking

Two separate VPCs connected via **VPC Peering** for metrics scraping:

| VPC | CIDR | Purpose |
|---|---|---|
| EC2 VPC | 172.31.0.0/16 | Workstation, Jenkins, monitoring stack |
| EKS VPC | 192.168.0.0/16 | Kubernetes cluster, worker nodes |
| Peering | pcx-02c1d5ad875b5c719 | Bidirectional routing between VPCs |

Routes added to all route tables in both VPCs. Security group rules scoped to private IPs only.

### Ingress

Single AWS Classic Load Balancer → Nginx Ingress Controller → ClusterIP services.

Path-based routing using regex capture groups:

```
/api/users(/|$)(.*)    → user-service:80
/api/products(/|$)(.*) → product-service:80
/api/orders(/|$)(.*)   → order-service:80
```

---

## CI/CD Pipeline

Jenkins declarative pipeline triggered automatically via GitHub webhook on every push to `main`.

**Stages:**

```
Checkout → Build Docker Images → Push to ECR → Deploy to EKS
```

- Images tagged with Jenkins `BUILD_NUMBER` — no `latest` tags in production
- AWS credentials managed via Jenkins credential store
- EKS authentication via `aws eks update-kubeconfig` — no static kubeconfig files
- `kubectl rollout status` gates the pipeline — deployment must succeed before marking build green
- Full pipeline completes in ~35 seconds

```groovy
stage('Deploy to EKS') {
    withAWS(credentials: 'aws-credentials', region: "${AWS_REGION}") {
        sh 'aws eks update-kubeconfig --region ${AWS_REGION} --name cloudmart'
        sh 'kubectl set image deployment/user-service ...'
        sh 'kubectl rollout status deployment/user-service'
    }
}
```

---

## Kubernetes Manifests

All deployments include:

- **Resource requests and limits** — CPU 100m/250m, Memory 128Mi/256Mi
- **Liveness probes** — restart unhealthy containers automatically
- **Readiness probes** — no traffic until container passes health check
- **2 replicas** — pods distributed across 2 nodes in separate AZs

```yaml
readinessProbe:
  httpGet:
    path: /health
    port: 3001
  initialDelaySeconds: 5
  periodSeconds: 10
livenessProbe:
  httpGet:
    path: /health
    port: 3001
  initialDelaySeconds: 15
  periodSeconds: 20
```

---

## Observability

### Metrics Pipeline

```
kube-state-metrics (EKS) → NodePort :31000
        ↓ VPC Peering (192.168.20.20:31000)
VictoriaMetrics (EC2) scrapes every 60s — 1389 metrics
        ↓
Grafana dashboard (Kubernetes Cluster Monitoring — ID 13332)
```

**Metrics collected:**
- Pod status, restarts, desired vs available replicas
- Node CPU and memory utilization
- Deployment health across all namespaces
- Container resource usage via cAdvisor

### Log Pipeline

```
Promtail (EC2) → Loki → Grafana
```

---

## Autoscaling

### HPA (Horizontal Pod Autoscaler)

Configured for all 3 services:

```yaml
minReplicas: 2
maxReplicas: 5
targetCPUUtilizationPercentage: 70
```

Pods scale out automatically when CPU exceeds 70% average across replicas.

### Cluster Autoscaler

Node group configured with min/max bounds:

```yaml
minSize: 1
maxSize: 3
desiredCapacity: 2
```

When pending pods cannot be scheduled due to resource constraints, Cluster Autoscaler provisions additional t3.medium nodes automatically.

---

## IAM Security

- Root account used only for initial setup — all operations performed via `cloudmart-admin` IAM user
- Worker nodes use instance profiles with minimum required policies (ECR read, CloudWatch write)
- EKS access entries scoped per principal — no wildcard permissions
- Security group rules scoped to specific CIDRs — no `0.0.0.0/0` on internal ports

---

## Local Development

```bash
# Build and test all services locally
docker build -t user-service:local ./user_service
docker run -d -p 3001:3001 user-service:local
curl http://localhost:3001/health

# Deploy to cluster
kubectl apply -f k8s/

# Verify
kubectl get pods -o wide
kubectl get svc
kubectl get ingress
```

---

## Cluster Lifecycle

```bash
# Provision
eksctl create cluster -f eks-cluster.yaml

# Deploy all services
kubectl apply -f k8s/

# Tear down (saves ~$5/day)
eksctl delete cluster --name cloudmart --region us-east-2
```

---



## Skills Demonstrated

`AWS EKS` `Kubernetes` `Docker` `Multi-stage Builds` `Helm` `Jenkins` `CI/CD` `GitHub Webhooks` `AWS ECR` `Nginx Ingress` `VPC Peering` `IAM` `HPA` `Cluster Autoscaler` `VictoriaMetrics` `Grafana` `Loki` `kube-state-metrics` `NodePort` `ClusterIP` `Rolling Deployments` `Health Probes` `Resource Limits` `Terraform`
