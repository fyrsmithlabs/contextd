# Deploying to Self-Hosted Temporal Cluster

This guide covers deploying the contextd plugin validation workflows to the self-hosted Temporal cluster at `temporal.fyrsmithlabs.ai`.

## Cluster Information

- **Frontend gRPC**: `temporal-frontend.temporal.svc:7233`
- **Web UI**: `https://temporal.fyrsmithlabs.ai`
- **Namespace**: `default` (or create a custom namespace)
- **Task Queue**: `plugin-validation-queue`

## Deployment Options

### Option 1: Kubernetes Deployment (Recommended)

Deploy the workers directly to your Kubernetes cluster:

```bash
# 1. Create secrets
kubectl create namespace contextd
kubectl create secret generic github-token -n contextd \
  --from-literal=token=ghp_your_github_token

kubectl create secret generic github-webhook-secret -n contextd \
  --from-literal=secret=your_webhook_secret

# 2. Build and push Docker images
docker build -t ghcr.io/fyrsmithlabs/contextd-plugin-validator:latest -f Dockerfile.plugin-validator .
docker build -t ghcr.io/fyrsmithlabs/contextd-github-webhook:latest -f Dockerfile.github-webhook .

docker push ghcr.io/fyrsmithlabs/contextd-plugin-validator:latest
docker push ghcr.io/fyrsmithlabs/contextd-github-webhook:latest

# 3. Deploy to cluster
kubectl apply -f deploy/k8s/plugin-validator.yaml

# 4. Verify deployment
kubectl get pods -n contextd
kubectl logs -n contextd deployment/plugin-validator-worker
kubectl logs -n contextd deployment/github-webhook
```

### Option 2: Docker Compose with Remote Temporal

Use the cluster-specific docker-compose file:

```bash
# 1. Set environment variables
export GITHUB_TOKEN=ghp_your_github_token
export GITHUB_WEBHOOK_SECRET=your_webhook_secret

# 2. Ensure your machine can reach temporal-frontend.temporal.svc:7233
# This may require VPN/Tailscale connection or SSH tunnel

# 3. Run the workers
docker-compose -f deploy/docker-compose.cluster.yml up -d

# 4. View logs
docker-compose -f deploy/docker-compose.cluster.yml logs -f
```

### Option 3: Local Development with Port Forward

For local testing against the cluster:

```bash
# 1. Port forward Temporal frontend
kubectl port-forward -n temporal svc/temporal-frontend 7233:7233

# 2. Set environment variables
export TEMPORAL_HOST=localhost:7233
export GITHUB_TOKEN=ghp_your_github_token
export GITHUB_WEBHOOK_SECRET=your_webhook_secret

# 3. Run workers locally
go run ./cmd/plugin-validator/main.go

# In another terminal
go run ./cmd/github-webhook/main.go
```

## Namespace Setup

Create a dedicated namespace for contextd workflows:

```bash
# Exec into Temporal admintools
kubectl exec -it -n temporal deployment/temporal-admintools -- /bin/bash

# Create namespace
tctl namespace register contextd-workflows \
  --description "Plugin validation workflows for contextd" \
  --retention 30

# Verify
tctl namespace describe contextd-workflows
```

Then update deployments to use the new namespace:

```yaml
env:
- name: TEMPORAL_NAMESPACE
  value: "contextd-workflows"
```

## Monitoring

### Web UI

Access workflows at: https://temporal.fyrsmithlabs.ai

Filter by task queue: `plugin-validation-queue`

### CLI

```bash
# List workflows
kubectl exec -it -n temporal deployment/temporal-admintools -- \
  tctl workflow list --query 'TaskQueue="plugin-validation-queue"'

# Show specific workflow
kubectl exec -it -n temporal deployment/temporal-admintools -- \
  tctl workflow show --workflow-id <workflow-id>

# Describe task queue
kubectl exec -it -n temporal deployment/temporal-admintools -- \
  tctl task-queue describe --task-queue plugin-validation-queue
```

### Worker Logs

```bash
# Kubernetes
kubectl logs -n contextd deployment/plugin-validator-worker -f
kubectl logs -n contextd deployment/github-webhook -f

# Docker Compose
docker-compose -f deploy/docker-compose.cluster.yml logs -f plugin-validator-worker
docker-compose -f deploy/docker-compose.cluster.yml logs -f github-webhook
```

## GitHub Webhook Configuration

Configure your GitHub repository to send webhooks to the deployed service:

1. Go to repository Settings → Webhooks → Add webhook

2. Set payload URL:
   - **Kubernetes**: `https://contextd-webhook.fyrsmithlabs.ai/webhook` (requires Ingress)
   - **Local**: Use ngrok or similar tunneling service

3. Set content type: `application/json`

4. Set secret: Your `GITHUB_WEBHOOK_SECRET` value

5. Select events:
   - Pull requests (opened, synchronize, reopened)

## Troubleshooting

### Workers Not Connecting

```bash
# Check DNS resolution from worker pod
kubectl exec -it -n contextd deployment/plugin-validator-worker -- \
  nslookup temporal-frontend.temporal.svc

# Check frontend service
kubectl get svc -n temporal temporal-frontend

# Verify worker logs
kubectl logs -n contextd deployment/plugin-validator-worker | grep "temporal client connected"
```

### Workflows Not Starting

```bash
# Check task queue has workers
kubectl exec -it -n temporal deployment/temporal-admintools -- \
  tctl task-queue describe --task-queue plugin-validation-queue

# Verify workflow registration in worker logs
kubectl logs -n contextd deployment/plugin-validator-worker | grep "worker configured"

# Test workflow manually
kubectl exec -it -n temporal deployment/temporal-admintools -- \
  tctl workflow start \
    --task-queue plugin-validation-queue \
    --workflow-type PluginUpdateValidationWorkflow \
    --input '{"Owner":"fyrsmithlabs","Repo":"contextd","PRNumber":1,"HeadSHA":"abc123"}'
```

### GitHub Token Issues

```bash
# Verify secret exists
kubectl get secret github-token -n contextd

# Check token is set in pod
kubectl exec -it -n contextd deployment/plugin-validator-worker -- \
  env | grep GITHUB_TOKEN

# Test GitHub API access from pod
kubectl exec -it -n contextd deployment/github-webhook -- \
  curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user
```

## Updating Deployments

```bash
# Rebuild and push new images
docker build -t ghcr.io/fyrsmithlabs/contextd-plugin-validator:v1.2.3 -f Dockerfile.plugin-validator .
docker push ghcr.io/fyrsmithlabs/contextd-plugin-validator:v1.2.3

# Update deployment
kubectl set image deployment/plugin-validator-worker -n contextd \
  worker=ghcr.io/fyrsmithlabs/contextd-plugin-validator:v1.2.3

# Rollout status
kubectl rollout status deployment/plugin-validator-worker -n contextd

# Rollback if needed
kubectl rollout undo deployment/plugin-validator-worker -n contextd
```

## Configuration Reference

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `TEMPORAL_HOST` | Temporal frontend address | `localhost:7233` | Yes |
| `TEMPORAL_NAMESPACE` | Temporal namespace | - | No (uses default) |
| `GITHUB_TOKEN` | GitHub API token | - | Yes |
| `GITHUB_WEBHOOK_SECRET` | Webhook signature secret | - | Yes (webhook only) |
| `PORT` | Webhook server port | `3000` | No |

### Task Queues

- `plugin-validation-queue` - Main validation workflows

### Workflow Types

- `PluginUpdateValidationWorkflow` - PR plugin validation

### Activity Types

- `FetchPRFilesActivity` - Fetch PR file changes
- `CategorizeFilesActivity` - Categorize code vs plugin files
- `ValidatePluginSchemasActivity` - Validate JSON schemas
- `PostReminderCommentActivity` - Post reminder to update plugin
- `PostSuccessCommentActivity` - Post success message
- `ValidateDocumentationActivity` - AI-powered doc validation (optional)

## Best Practices

1. **Use dedicated namespace**: Create `contextd-workflows` namespace for isolation
2. **Set retention period**: 30 days is recommended for PR workflows
3. **Monitor task queue depth**: Alert if backlog grows
4. **Use multiple replicas**: Deploy 2+ worker replicas for availability
5. **Set resource limits**: Prevent runaway resource usage
6. **Enable agent validation**: Set `UseAgentValidation: true` for semantic checks
7. **Rotate secrets regularly**: Update GitHub tokens and webhook secrets
