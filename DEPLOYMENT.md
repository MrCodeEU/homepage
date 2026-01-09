# Deployment Trigger Mechanism

This document describes options for triggering deployments to your homelab when data is updated or new Docker images are built.

## Overview

The GitHub Actions workflows support optional webhook triggers that can notify your homelab deployment system when:
1. **Data is updated** (new GitHub/Strava/LinkedIn data generated)
2. **Docker image is built** (new image pushed to GitHub Container Registry)

## Webhook Integration

### How It Works

Both workflows (`generate-data.yml` and `docker.yml`) include an optional step at the end:

```yaml
- name: Trigger deployment webhook (optional)
  if: success() && vars.DEPLOY_WEBHOOK_URL != ''
  run: |
    curl -X POST \
      -H "Content-Type: application/json" \
      -d '{"source": "github-actions", "action": "data-update", "timestamp": "'$(date -u +%s)'"}' \
      ${{ vars.DEPLOY_WEBHOOK_URL }} || true
```

This sends a POST request to your specified webhook URL when the workflow completes successfully.

### Webhook Payload

#### Data Update Webhook

```json
{
  "source": "github-actions",
  "action": "data-update",
  "timestamp": 1704067200
}
```

#### Docker Update Webhook

```json
{
  "source": "github-actions",
  "action": "docker-update",
  "tag": "latest",
  "timestamp": 1704067200
}
```

### Setup

1. **Configure webhook URL in GitHub:**
   - Go to Settings → Secrets and variables → Actions → Variables
   - Click "New repository variable"
   - Name: `DEPLOY_WEBHOOK_URL`
   - Value: `https://your-homelab.com/webhook/homepage`

2. **Implement webhook receiver in your homelab** (see options below)

## Deployment Options

### Option 1: Ansible Pull via Webhook

Use a simple webhook receiver that triggers Ansible to pull and deploy the latest changes.

#### Setup Steps

1. **Create webhook receiver script on your server:**

```bash
#!/bin/bash
# /opt/webhook-receiver/homepage-deploy.sh

ACTION=$1
TIMESTAMP=$2

echo "[$(date)] Received deployment trigger: action=$ACTION, timestamp=$TIMESTAMP"

# For data updates, just pull the latest data
if [ "$ACTION" = "data-update" ]; then
    cd /path/to/homepage-repo
    git pull origin main
    # Optionally restart the service to pick up new data
    docker-compose restart homepage
fi

# For Docker updates, pull new image and restart
if [ "$ACTION" = "docker-update" ]; then
    docker pull ghcr.io/mrcodeeu/homepage:latest
    docker-compose up -d homepage
fi

echo "[$(date)] Deployment completed"
```

2. **Install webhook service** (using [webhook](https://github.com/adnanh/webhook)):

```bash
# Install webhook
sudo apt-get install webhook

# Create webhook config
cat > /etc/webhook/hooks.json <<EOF
[
  {
    "id": "homepage-deploy",
    "execute-command": "/opt/webhook-receiver/homepage-deploy.sh",
    "command-working-directory": "/opt/webhook-receiver",
    "pass-arguments-to-command": [
      {
        "source": "payload",
        "name": "action"
      },
      {
        "source": "payload",
        "name": "timestamp"
      }
    ],
    "response-message": "Deployment triggered successfully"
  }
]
EOF

# Start webhook service
sudo systemctl enable webhook
sudo systemctl start webhook
```

3. **Configure nginx reverse proxy:**

```nginx
# /etc/nginx/sites-available/webhook
server {
    listen 443 ssl;
    server_name webhook.your-homelab.com;

    ssl_certificate /etc/letsencrypt/live/your-homelab.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-homelab.com/privkey.pem;

    location /webhook/homepage {
        proxy_pass http://localhost:9000/hooks/homepage-deploy;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Option 2: AWX/Tower Workflow

If you're using AWX or Ansible Tower:

1. **Create a webhook-triggered job template** in AWX
2. **Configure webhook URL** to point to AWX webhook endpoint:
   ```
   https://awx.your-homelab.com/api/v2/job_templates/XX/github/
   ```
3. **Create Ansible playbook** that:
   - Pulls latest code from GitHub
   - Pulls latest Docker image from GHCR
   - Restarts services

### Option 3: GitHub Repository Dispatch

Use GitHub repository dispatch to trigger a workflow in your private homelab repo.

#### Setup Steps

1. **Create Personal Access Token** with `repo` scope for your private homelab repo

2. **Add secret to this repo:**
   - Name: `HOMELAB_REPO_TOKEN`
   - Value: Your PAT

3. **Update workflows to use repository dispatch:**

```yaml
- name: Trigger homelab deployment
  if: success()
  run: |
    curl -X POST \
      -H "Authorization: token ${{ secrets.HOMELAB_REPO_TOKEN }}" \
      -H "Accept: application/vnd.github.v3+json" \
      https://api.github.com/repos/your-username/homelab/dispatches \
      -d '{"event_type":"homepage-update","client_payload":{"action":"data-update"}}'
```

4. **In your homelab repo, create workflow:**

```yaml
# .github/workflows/homepage-deploy.yml
name: Deploy Homepage

on:
  repository_dispatch:
    types: [homepage-update]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout homelab repo
        uses: actions/checkout@v4

      - name: Run Ansible playbook
        run: |
          ansible-playbook -i inventory/production playbooks/deploy-homepage.yml
```

### Option 4: Watchtower (Docker-only)

For simple Docker deployments, use [Watchtower](https://containrrr.dev/watchtower/) to automatically pull and restart containers when images are updated.

#### Setup Steps

1. **Deploy Watchtower:**

```yaml
# docker-compose.yml
version: '3'
services:
  homepage:
    image: ghcr.io/mrcodeeu/homepage:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    labels:
      - "com.centurylinklabs.watchtower.enable=true"

  watchtower:
    image: containrrr/watchtower
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    command: --interval 300 --cleanup --label-enable
```

2. **Watchtower will automatically:**
   - Check for new images every 5 minutes
   - Pull new images
   - Restart containers with updated images
   - Clean up old images

**Note:** This handles Docker updates but not data updates. For data updates, you'd still need a webhook or scheduled git pull.

### Option 5: Manual Trigger (No Webhook)

If you prefer manual control:

1. **Remove or leave webhook URL empty** - workflows will skip the trigger step
2. **Manually pull updates:**

```bash
# Pull latest data
cd /path/to/homepage
git pull origin main

# Pull latest Docker image
docker pull ghcr.io/mrcodeeu/homepage:latest
docker-compose up -d
```

## Recommended Approach

For a homelab setup, we recommend:

1. **Use Option 4 (Watchtower)** for automatic Docker image updates
2. **Add a simple webhook receiver (Option 1)** for data updates that triggers `git pull`
3. **Set data update frequency** appropriately:
   - GitHub/LinkedIn: Once daily (changes infrequently)
   - Strava: Every 4 hours (more dynamic)

This provides a good balance of automation without requiring complex infrastructure.

## Security Considerations

1. **Webhook Authentication**
   - Consider adding HMAC signature verification
   - Use HTTPS only
   - Restrict webhook receiver to GitHub IP ranges

2. **Access Control**
   - Use firewall rules to limit webhook receiver access
   - Run webhook processes with minimal privileges
   - Use separate service accounts for deployments

3. **Monitoring**
   - Log all webhook triggers
   - Set up alerts for failed deployments
   - Monitor GitHub Actions workflow status

## Testing

### Test Data Generation Workflow

```bash
# Trigger manually
gh workflow run generate-data.yml -f sources=all

# Check logs
gh run list --workflow=generate-data.yml
gh run view <run-id> --log
```

### Test Docker Build Workflow

```bash
# Push to dev branch (triggers :dev tag)
git checkout -b dev/test-deployment
git push origin dev/test-deployment

# Check workflow
gh run list --workflow=docker.yml
```

### Test Webhook Receiver

```bash
# Send test request
curl -X POST https://webhook.your-homelab.com/webhook/homepage \
  -H "Content-Type: application/json" \
  -d '{"source":"manual-test","action":"data-update","timestamp":1704067200}'
```

## Troubleshooting

### Webhook not triggering

1. Check that `DEPLOY_WEBHOOK_URL` variable is set in GitHub
2. Verify webhook URL is accessible from GitHub
3. Check webhook receiver logs
4. Ensure HTTPS certificate is valid

### Deployment fails

1. Check webhook receiver has permissions to pull git/docker
2. Verify GitHub token or credentials are valid
3. Check disk space and Docker resources
4. Review deployment script logs

### Data not updating

1. Verify GitHub Actions workflow completed successfully
2. Check that data files were committed to the repo
3. Ensure webhook triggered git pull
4. Verify application restarted and picked up new data

## Example: Complete Homelab Setup

Here's a complete example using Option 1 (Webhook) + Option 4 (Watchtower):

```yaml
# docker-compose.yml in your homelab
version: '3.8'

services:
  homepage:
    image: ghcr.io/mrcodeeu/homepage:latest
    container_name: homepage
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      # Mount pre-generated data (updated via git pull)
      - /opt/homepage-data:/app/backend/data/generated:ro
    labels:
      - "com.centurylinklabs.watchtower.enable=true"

  watchtower:
    image: containrrr/watchtower
    container_name: watchtower
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - WATCHTOWER_CLEANUP=true
      - WATCHTOWER_LABEL_ENABLE=true
      - WATCHTOWER_INCLUDE_STOPPED=false
      - WATCHTOWER_POLL_INTERVAL=300
```

```bash
# /opt/homepage-data/update.sh
#!/bin/bash
set -e

cd /opt/homepage-data
git pull origin main

# Restart homepage to pick up new data
docker-compose -f /opt/homepage/docker-compose.yml restart homepage

echo "Homepage data updated successfully"
```

This setup provides:
- Automatic Docker image updates via Watchtower
- Data updates via webhook → git pull → restart
- Minimal manual intervention required
