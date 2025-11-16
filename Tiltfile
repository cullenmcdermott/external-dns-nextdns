# Tiltfile for external-dns-nextdns-webhook development

# Load extensions
load('ext://restart_process', 'docker_build_with_restart')

# Configuration
image_name = 'localhost:5005/external-dns-nextdns-webhook'
namespace = 'external-dns'

# Allow k8s contexts for local development
allow_k8s_contexts('kind-external-dns-dev')

# Ignore flox cache/log files and other non-code files
watch_settings(ignore=[
    '.flox/',
    '.git/',
    '*.log',
    '*.tmp',
    'Tiltfile.tmp.*',
    'justfile.tmp.*',
    'Dockerfile.tmp.*',
])

# Build Docker image with live reload
docker_build(
    image_name,
    '.',
    dockerfile='Dockerfile',
    # Live update: sync Go files and restart on changes
    live_update=[
        # Sync source files
        sync('./cmd', '/app/cmd'),
        sync('./internal', '/app/internal'),
        sync('./pkg', '/app/pkg'),
        # Rebuild on Go file changes
        run('cd /app && CGO_ENABLED=0 go build -o webhook ./cmd/webhook',
            trigger=['./cmd', './internal', './pkg']),
    ],
    # Build arguments
    build_args={'VERSION': 'dev-tilt'},
)

# Secret management - try 1Password first, fallback to dev secret
local_resource(
    'secrets',
    cmd='''
    # Wait for cluster to be ready (retry for up to 60 seconds)
    echo "⏳ Waiting for Kubernetes API server..."
    max_attempts=30
    attempt=0
    while [ $attempt -lt $max_attempts ]; do
      if kubectl cluster-info >/dev/null 2>&1; then
        echo "✅ Cluster is ready"
        break
      fi
      attempt=$((attempt + 1))
      sleep 2
    done

    if [ $attempt -eq $max_attempts ]; then
      echo "❌ Timeout waiting for cluster"
      exit 1
    fi

    # Ensure namespace exists first
    kubectl create namespace external-dns --dry-run=client -o yaml | kubectl apply -f - >/dev/null 2>&1

    if command -v op >/dev/null 2>&1; then
      # Try to read secrets directly - desktop app integration should handle auth
      API_KEY=$(op read "op://Private/NextDNSAPI/api_key" 2>/dev/null)
      PROFILE_ID=$(op read "op://Private/NextDNSAPI/profile_id" 2>/dev/null)

      if [ -n "$API_KEY" ] && [ -n "$PROFILE_ID" ]; then
        echo "✅ Using credentials from 1Password"
        kubectl create secret generic nextdns-webhook-secret \
          --from-literal=NEXTDNS_API_KEY="$API_KEY" \
          --from-literal=NEXTDNS_PROFILE_ID="$PROFILE_ID" \
          --namespace=external-dns \
          --dry-run=client -o yaml | kubectl apply -f -
      else
        echo ""
        echo "⚠️  Could not fetch from 1Password"
        echo "   Make sure:"
        echo "   1. 1Password desktop app is running and unlocked"
        echo "   2. You have a 'NextDNSAPI' item in your 'Private' vault with 'api_key' and 'profile_id' fields"
        echo ""
        echo "   Falling back to dev credentials for now..."
        echo ""
        kubectl apply -f deploy/kubernetes/secret-dev.yaml
      fi
    else
      echo "⚠️  1Password CLI not installed, using dev credentials"
      kubectl apply -f deploy/kubernetes/secret-dev.yaml
    fi
    ''',
    labels=['config'],
)

# Deploy Kubernetes manifests (use dev overlay to disable read-only filesystem for live updates)
yaml = kustomize('deploy/overlays/dev')

# Replace the image with our local one
yaml = blob(str(yaml).replace(
    'ghcr.io/cullenmcdermott/external-dns-nextdns-webhook:latest',
    image_name
))

k8s_yaml(yaml)

# Group supporting resources under 'config' label
k8s_resource(
    objects=[
        'external-dns:namespace',
        'external-dns:serviceaccount',
        'external-dns:clusterrole',
        'external-dns:clusterrolebinding',
    ],
    new_name='rbac',
    labels=['config'],
    resource_deps=['secrets'],
)

k8s_resource(
    objects=[
        'nextdns-webhook-config:ConfigMap:external-dns',
    ],
    new_name='config',
    labels=['config'],
)

# Main deployment
k8s_resource(
    'external-dns-nextdns',
    new_name='webhook',
    port_forwards=[
        '9080:8080',  # Health check endpoint
        '9888:8888',  # Webhook API endpoint
    ],
    resource_deps=['rbac', 'config'],
    labels=['app'],
)

# Example resources for testing external-dns
k8s_resource(
    objects=[
        'example-app:ingress:default',
        'example-loadbalancer:service:default',
    ],
    new_name='examples',
    labels=['examples'],
)

# Watch for changes in Go files
watch_file('go.mod')
watch_file('go.sum')

# Print useful information
print('''
╔════════════════════════════════════════════════════════╗
║  external-dns-nextdns-webhook - Tilt Development      ║
╠════════════════════════════════════════════════════════╣
║  Webhook API:   http://localhost:9888                  ║
║  Health Check:  http://localhost:9080/healthz          ║
║  Registry:      localhost:5005                         ║
╚════════════════════════════════════════════════════════╝

🔧 Live reload enabled for Go files
📝 Edit files in ./cmd, ./internal, or ./pkg to trigger rebuild
🌐 Tilt UI: http://localhost:10350
''')
