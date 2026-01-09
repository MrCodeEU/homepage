# Secrets and Environment Variables

This document describes all required secrets and environment variables for the homepage project.

## Table of Contents

1. [GitHub Configuration](#github-configuration)
2. [Strava Configuration](#strava-configuration)
3. [LinkedIn Configuration](#linkedin-configuration)
4. [GitHub Actions Secrets](#github-actions-secrets)
5. [Deployment Configuration](#deployment-configuration)
6. [Local Development Setup](#local-development-setup)

## GitHub Configuration

### GITHUB_USERNAME

- **Required:** Yes
- **Description:** Your GitHub username for fetching portfolio projects
- **Example:** `mrcodeeu`
- **Where to set:**
  - Local: Environment variable or `.env` file
  - CI/CD: GitHub Actions secret `GH_USERNAME`
  - Docker: Pass as environment variable

### GITHUB_TOKEN

- **Required:** Yes
- **Description:** GitHub Personal Access Token for API authentication
- **Permissions needed:** `public_repo` (read access to public repositories)
- **How to create:**
  1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
  2. Click "Generate new token (classic)"
  3. Give it a descriptive name (e.g., "Homepage API")
  4. Select scope: `public_repo`
  5. Click "Generate token"
  6. Copy the token immediately (you won't see it again!)
- **Example:** `ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`
- **Where to set:**
  - Local: Environment variable `GITHUB_TOKEN`
  - CI/CD: GitHub Actions secret `GH_API_TOKEN`
  - Docker: Pass as environment variable

## Strava Configuration

### STRAVA_CLIENT_ID

- **Required:** Yes (for Strava features)
- **Description:** Strava API application client ID
- **Example:** `123456`
- **How to create:**
  1. Go to [Strava API Settings](https://www.strava.com/settings/api)
  2. Create a new application if you haven't already
  3. Copy the "Client ID"
- **Where to set:**
  - Local: Environment variable `STRAVA_CLIENT_ID`
  - CI/CD: GitHub Actions secret `STRAVA_CLIENT_ID`

### STRAVA_CLIENT_SECRET

- **Required:** Yes (for Strava features)
- **Description:** Strava API application client secret
- **Example:** `xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`
- **How to create:**
  1. Same page as Client ID
  2. Copy the "Client Secret"
- **Where to set:**
  - Local: Environment variable `STRAVA_CLIENT_SECRET`
  - CI/CD: GitHub Actions secret `STRAVA_CLIENT_SECRET`

### STRAVA_REFRESH_TOKEN

- **Required:** Yes (for Strava features)
- **Description:** OAuth2 refresh token for Strava API
- **How to obtain:**
  1. After creating your Strava API application, note your Client ID and Secret
  2. Authorize your app (one-time setup):
     ```bash
     # Replace CLIENT_ID with your actual Client ID
     # Visit this URL in your browser:
     https://www.strava.com/oauth/authorize?client_id=CLIENT_ID&response_type=code&redirect_uri=http://localhost&approval_prompt=force&scope=activity:read_all
     ```
  3. Authorize the application
  4. You'll be redirected to `http://localhost/?code=AUTHORIZATION_CODE`
  5. Copy the `code` parameter from the URL
  6. Exchange the code for tokens:
     ```bash
     curl -X POST https://www.strava.com/oauth/token \
       -d client_id=YOUR_CLIENT_ID \
       -d client_secret=YOUR_CLIENT_SECRET \
       -d code=AUTHORIZATION_CODE \
       -d grant_type=authorization_code
     ```
  7. The response will include a `refresh_token` - save this!
  8. The refresh token doesn't expire unless revoked
- **Example:** `xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`
- **Where to set:**
  - Local: Environment variable `STRAVA_REFRESH_TOKEN`
  - CI/CD: GitHub Actions secret `STRAVA_REFRESH_TOKEN`

## LinkedIn Configuration

LinkedIn data scraping is currently experimental. The scraper attempts to extract data from your public LinkedIn profile.

### Profile URL

- Currently hardcoded in `backend/cmd/generate/main.go` as `https://at.linkedin.com/in/mrcodeeu`
- Can be made configurable via environment variable `LINKEDIN_PROFILE_URL`

### Manual Configuration (Recommended for MVP)

Since LinkedIn API access is restricted and web scraping is unreliable, you can manually configure your LinkedIn data:

1. Run data generation once (it will create a placeholder)
2. Edit `backend/data/generated/linkedin.json` manually with your actual data
3. Commit the updated file

## GitHub Actions Secrets

Configure these in your GitHub repository settings under Settings → Secrets and variables → Actions → Repository secrets:

### Required Secrets

| Secret Name | Description | Source |
|-------------|-------------|--------|
| `GH_API_TOKEN` | GitHub Personal Access Token | See [GITHUB_TOKEN](#github_token) |
| `GH_USERNAME` | GitHub username | Your GitHub username |
| `STRAVA_CLIENT_ID` | Strava API Client ID | See [STRAVA_CLIENT_ID](#strava_client_id) |
| `STRAVA_CLIENT_SECRET` | Strava API Client Secret | See [STRAVA_CLIENT_SECRET](#strava_client_secret) |
| `STRAVA_REFRESH_TOKEN` | Strava OAuth2 refresh token | See [STRAVA_REFRESH_TOKEN](#strava_refresh_token) |

### Optional Variables

Configure these in Settings → Secrets and variables → Actions → Repository variables:

| Variable Name | Description | Example |
|---------------|-------------|---------|
| `DEPLOY_WEBHOOK_URL` | Webhook URL to trigger deployment updates | `https://your-server.com/webhook` |

## Deployment Configuration

### Docker Environment Variables

When running the Docker container, you can pass these environment variables:

```bash
docker run -d \
  -p 8080:8080 \
  -e PORT=8080 \
  -e GITHUB_USERNAME=mrcodeeu \
  ghcr.io/mrcodeeu/homepage:latest
```

**Note:** The Docker container serves pre-generated data files. It does not need API credentials at runtime. API credentials are only needed during data generation.

### Data Generation Directory

- **Default:** `./data/generated`
- **Environment variable:** Not configurable (hardcoded in loader)
- **Docker mount:** Can mount a volume at `/app/backend/data/generated` to persist data

## Local Development Setup

### Step 1: Create a `.env` file

Create a file named `.env` in the project root (it's gitignored):

```bash
# GitHub
GITHUB_USERNAME=your-username
GITHUB_TOKEN=ghp_your_token_here

# Strava
STRAVA_CLIENT_ID=your_client_id
STRAVA_CLIENT_SECRET=your_client_secret
STRAVA_REFRESH_TOKEN=your_refresh_token

# Optional
PORT=8080
```

### Step 2: Load environment variables

```bash
# Option 1: Source the .env file
source .env

# Option 2: Use direnv (recommended)
# Install direnv: https://direnv.net/
# Then create .envrc:
echo 'dotenv' > .envrc
direnv allow

# Option 3: Export manually
export GITHUB_USERNAME=your-username
export GITHUB_TOKEN=ghp_your_token_here
# ... etc
```

### Step 3: Generate data locally

```bash
make generate-data
```

### Step 4: Run the server

```bash
make dev-backend
```

## Security Best Practices

1. **Never commit secrets to git**
   - `.env` files are gitignored
   - Always use environment variables or secret managers

2. **Rotate tokens regularly**
   - GitHub tokens can be regenerated in Settings
   - Strava refresh tokens can be revoked and recreated

3. **Use minimal permissions**
   - GitHub: Only grant `public_repo` access
   - Strava: Only request `activity:read_all` scope

4. **Monitor API usage**
   - GitHub: 5,000 requests/hour with authentication
   - Strava: 100 requests/15 minutes, 1,000 requests/day

5. **Secure webhook endpoints**
   - If using deployment webhooks, implement authentication
   - Use HTTPS only
   - Validate request signatures if possible

## Troubleshooting

### "Failed to fetch repositories" error

- Check that `GITHUB_TOKEN` is set correctly
- Verify token has `public_repo` permission
- Check GitHub API rate limits: https://api.github.com/rate_limit

### "Token exchange failed" for Strava

- Verify `STRAVA_CLIENT_ID` and `STRAVA_CLIENT_SECRET` are correct
- Check that `STRAVA_REFRESH_TOKEN` hasn't been revoked
- Re-authorize the application if needed (see setup instructions above)

### LinkedIn scraping fails

- This is expected - LinkedIn has anti-scraping measures
- Use manual configuration instead (edit `linkedin.json` directly)
- Consider implementing a manual data entry form in the future

## Future Enhancements

- Add support for `.env` file loading in Go code
- Implement secret rotation automation
- Add encrypted secret storage for sensitive data
- Create a setup wizard for first-time configuration
- Add LinkedIn API integration if access is granted
