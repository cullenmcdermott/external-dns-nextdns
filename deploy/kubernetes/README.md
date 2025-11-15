# Kubernetes Deployment for NextDNS Webhook

This directory contains all the Kubernetes manifests needed to deploy the NextDNS webhook provider with external-dns.

## Quick Start

### 1. Create the Secret

First, copy the secret example and add your NextDNS credentials:

```bash
cp secret.yaml.example secret.yaml
# Edit secret.yaml and replace the placeholders with your actual credentials
```

### 2. Deploy to Kubernetes

#### Option A: Using kubectl

```bash
# Apply all manifests
kubectl apply -f namespace.yaml
kubectl apply -f secret.yaml
kubectl apply -f serviceaccount.yaml
kubectl apply -f clusterrole.yaml
kubectl apply -f clusterrolebinding.yaml
kubectl apply -f configmap.yaml
kubectl apply -f service.yaml
kubectl apply -f deployment.yaml
```

#### Option B: Using Kustomize

```bash
# First, update kustomization.yaml to uncomment the secret.yaml line
# Then apply:
kubectl apply -k .
```

### 3. Verify Deployment

```bash
# Check pods are running
kubectl get pods -n external-dns

# Check logs
kubectl logs -n external-dns -l app.kubernetes.io/name=external-dns -c nextdns-webhook
kubectl logs -n external-dns -l app.kubernetes.io/name=external-dns -c external-dns
```

### 4. Test with Example Ingress

```bash
# Deploy the example ingress (update the hostname first!)
kubectl apply -f example-ingress.yaml

# Check external-dns logs to see if it processes the ingress
kubectl logs -n external-dns -l app.kubernetes.io/name=external-dns -c external-dns --follow
```

## Configuration

### Environment Variables

The webhook can be configured via the ConfigMap (`configmap.yaml`) or directly in the Deployment:

| Variable | Default | Description |
|----------|---------|-------------|
| `NEXTDNS_API_KEY` | - | **Required**: Your NextDNS API key (from secret) |
| `NEXTDNS_PROFILE_ID` | - | **Required**: Your NextDNS Profile ID (from secret) |
| `SERVER_PORT` | 8888 | Webhook API port (internal) |
| `HEALTH_PORT` | 8080 | Health check port |
| `DRY_RUN` | false | If true, logs changes without applying them |
| `ALLOW_OVERWRITE` | false | If true, allows overwriting existing DNS records |
| `LOG_LEVEL` | info | Log level (debug, info, warn, error) |
| `SUPPORTED_RECORDS` | A,AAAA,CNAME | Comma-separated list of supported record types |
| `DEFAULT_TTL` | 300 | Default TTL for DNS records |
| `DOMAIN_FILTER` | "" | Comma-separated list of domains to manage |

### Domain Filtering

To limit which domains the webhook manages, set the `DOMAIN_FILTER` in the ConfigMap:

```yaml
data:
  DOMAIN_FILTER: "example.com,example.org"
```

### External-DNS Configuration

The external-dns container can be configured via the `args` section in `deployment.yaml`:

```yaml
args:
  - --source=ingress           # Watch Ingress resources
  - --source=service           # Watch Service resources
  - --provider=webhook         # Use webhook provider
  - --policy=upsert-only       # Only create/update, never delete
  - --registry=txt             # Use TXT records for ownership
  - --txt-owner-id=external-dns-nextdns
  - --log-level=info
```

## Architecture

The deployment uses a **sidecar pattern**:

- **NextDNS Webhook Container**: Exposes the webhook API on `localhost:8888`
- **External-DNS Container**: Watches Kubernetes resources and calls the webhook

Both containers run in the same pod and communicate via localhost.

## Troubleshooting

### Pods not starting

```bash
# Check pod status
kubectl describe pod -n external-dns -l app.kubernetes.io/name=external-dns

# Common issues:
# - Secret not created
# - Image pull errors
# - Resource constraints
```

### Webhook not processing records

```bash
# Check webhook logs
kubectl logs -n external-dns -l app.kubernetes.io/name=external-dns -c nextdns-webhook

# Check if external-dns can reach the webhook
kubectl logs -n external-dns -l app.kubernetes.io/name=external-dns -c external-dns | grep webhook
```

### DNS records not created in NextDNS

1. Enable dry-run mode to see what would be created:
   ```bash
   kubectl set env deployment/external-dns-nextdns DRY_RUN=true -n external-dns
   ```

2. Check the webhook logs for warnings about overwrites:
   ```bash
   kubectl logs -n external-dns -l app.kubernetes.io/name=external-dns -c nextdns-webhook | grep WARNING
   ```

3. Verify your NextDNS API credentials are correct

### Enable Debug Logging

```bash
# For the webhook
kubectl set env deployment/external-dns-nextdns LOG_LEVEL=debug -n external-dns

# For external-dns, edit the deployment and change --log-level=info to --log-level=debug
```

## Security Considerations

- The webhook runs as a non-root user (UID 65534)
- Read-only root filesystem
- All capabilities dropped
- Secrets are mounted as environment variables
- The webhook API is only accessible via localhost (not exposed outside the pod)

## Updating

To update the deployment:

```bash
# Update the image version in deployment.yaml
# Then apply the changes
kubectl apply -f deployment.yaml

# Or to update just the image:
kubectl set image deployment/external-dns-nextdns \
  nextdns-webhook=ghcr.io/cullenmcdermott/external-dns-nextdns-webhook:v1.0.0 \
  -n external-dns
```

## Cleanup

To remove the deployment:

```bash
kubectl delete -f deployment.yaml
kubectl delete -f service.yaml
kubectl delete -f configmap.yaml
kubectl delete -f secret.yaml
kubectl delete -f clusterrolebinding.yaml
kubectl delete -f clusterrole.yaml
kubectl delete -f serviceaccount.yaml
kubectl delete -f namespace.yaml
```

Or with Kustomize:

```bash
kubectl delete -k .
```
