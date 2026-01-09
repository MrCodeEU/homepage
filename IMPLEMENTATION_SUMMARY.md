# Implementation Summary: API Data Gathering & Workflows

This document summarizes the implementation of the GitHub, Strava, and LinkedIn API data gathering system and associated GitHub Actions workflows.

## What Was Implemented

### 1. Data Gathering Architecture ✅

**Approach:** Pre-generated static data files committed to the repository

- **Data Storage:** `backend/data/generated/*.json`
- **Data Models:** Defined in `backend/internal/models/data.go`
- **Scrapers:**
  - `backend/internal/scrapers/github.go` - GitHub portfolio projects
  - `backend/internal/scrapers/strava.go` - Strava running activities, stats, PRs
  - `backend/internal/scrapers/linkedin.go` - LinkedIn profile (basic MVP)

**Data Structures:**

- **GitHub:** Portfolio projects with .portfolio markers
- **Strava:**
  - Total stats (lifetime)
  - Year-to-date stats
  - Recent 10 activities
  - Best activities (longest distance, time, fastest pace, most elevation)
  - Personal records (5k, 10k, half marathon, marathon)
- **LinkedIn:**
  - Profile (name, headline, location, summary)
  - Work experience
  - Education
  - Skills

### 2. CLI Data Generation Tool ✅

**Location:** `backend/cmd/generate/main.go`

**Features:**
- Generate data for all sources or selectively (github, strava, linkedin)
- Save to JSON files with metadata (timestamp, version)
- Verbose logging
- Error handling with fallbacks

**Usage:**
```bash
cd backend
go run cmd/generate/main.go -output ./data/generated -sources all -verbose
```

Or via Makefile:
```bash
make generate-data
```

### 3. Backend Server Updates ✅

**Changes to `backend/cmd/server/main.go`:**
- Removed scraper initialization
- Added data loader (`internal/storage/loader.go`)
- Updated all handlers to read from pre-generated data files:
  - `/api/cv` → LinkedIn data
  - `/api/projects` → GitHub projects
  - `/api/strava` → Strava data

**New File:** `backend/internal/storage/loader.go`
- Loads pre-generated JSON files
- Converts to appropriate data structures
- Error handling for missing files

### 4. GitHub Actions Workflows ✅

#### Data Generation Workflow (`.github/workflows/generate-data.yml`)

**Triggers:**
- **Schedule:**
  - Daily at 6 AM UTC (all sources)
  - Every 4 hours (Strava only)
- **Manual:** workflow_dispatch with source selection

**Steps:**
1. Checkout repository
2. Set up Go environment
3. Determine which sources to generate
4. Run data generation with credentials from secrets
5. Commit generated data back to repo
6. Optionally trigger deployment webhook

**Secrets Required:**
- `GH_API_TOKEN` - GitHub Personal Access Token
- `GH_USERNAME` - GitHub username
- `STRAVA_CLIENT_ID` - Strava API Client ID
- `STRAVA_CLIENT_SECRET` - Strava API Client Secret
- `STRAVA_REFRESH_TOKEN` - Strava OAuth2 refresh token

#### Docker Build Workflow (`.github/workflows/docker.yml`)

**Triggers:**
- Push to `main` branch → `:latest` tag
- Push to branches containing "dev" → `:dev` tag
- Tags matching `v*` → Version tags
- Pull requests → Build only (no push)
- Manual trigger

**Features:**
- Multi-platform support (linux/amd64)
- Layer caching via GitHub Actions cache
- Automatic tag management
- Push to GitHub Container Registry (ghcr.io)
- Build summaries in workflow output
- Optional deployment webhook trigger

**Image Location:** `ghcr.io/mrcodeeu/homepage:latest`

### 5. Frontend Type Updates ✅

**Location:** `frontend/src/lib/api.ts`

**Changes:**
- Added complete TypeScript types for all data structures
- New API functions:
  - `getLinkedInData()` → LinkedInData
  - `getStravaData()` → StravaData
  - `getProjects()` → Project[]
- Maintained legacy functions for backward compatibility

**Data Types Match Backend:**
- LinkedInData, LinkedInProfile, LinkedInExperience, LinkedInEducation
- StravaData, StravaStats, StravaActivity, StravaBestRecords, StravaRecord
- Project (GitHub)

### 6. Makefile Updates ✅

**New Commands:**

```bash
make generate-data    # Generate data from APIs locally
make test-real        # Run tests against real APIs
make test-real-backend
make test-real-frontend
```

**Updated Help:** Now includes data generation and real API testing sections

### 7. Documentation ✅

Created comprehensive documentation:

#### `SECRETS.md`
- Complete guide to all required secrets and environment variables
- Step-by-step instructions for obtaining API credentials:
  - GitHub Personal Access Token
  - Strava OAuth2 setup (with full authorization flow)
  - LinkedIn configuration
- GitHub Actions secrets setup
- Local development setup
- Security best practices
- Troubleshooting guide

#### `DEPLOYMENT.md`
- 5 deployment options with detailed setup instructions:
  1. Ansible Pull via Webhook
  2. AWX/Tower Workflow
  3. GitHub Repository Dispatch
  4. Watchtower (Docker auto-update)
  5. Manual Trigger
- Recommended approach for homelab
- Security considerations
- Testing instructions
- Complete example setup
- Troubleshooting guide

#### `IMPLEMENTATION_SUMMARY.md` (this file)
- Overview of all changes
- Architecture decisions
- Usage instructions

## Architecture Decisions

### Why Pre-Generated Data?

**Advantages:**
- ✅ Separates data collection from serving
- ✅ Server doesn't need API credentials
- ✅ Faster response times (no API calls)
- ✅ Continues working if APIs are down
- ✅ Easy to version control changes
- ✅ Can review data before deployment

**Trade-offs:**
- ⚠️ Data is slightly stale (but acceptable for GitHub/LinkedIn)
- ⚠️ Requires CI/CD setup for automatic updates

### Data Update Frequency

- **GitHub/LinkedIn:** Daily (changes infrequently)
- **Strava:** Every 4 hours (more dynamic, 6x per day)

Can be adjusted in `.github/workflows/generate-data.yml` cron schedules.

### LinkedIn Scraping

LinkedIn API access is highly restricted. Current implementation:
- Attempts to scrape public profile HTML
- Extracts JSON-LD structured data if available
- Falls back to basic HTML parsing
- **Recommended:** Manually edit `linkedin.json` for reliable data

## Getting Started

### 1. Set Up API Credentials

Follow `SECRETS.md` to obtain:
- GitHub Personal Access Token
- Strava OAuth2 credentials
- (Optional) LinkedIn configuration

### 2. Configure GitHub Secrets

In your GitHub repository:
1. Go to Settings → Secrets and variables → Actions
2. Add repository secrets:
   - `GH_API_TOKEN`
   - `GH_USERNAME`
   - `STRAVA_CLIENT_ID`
   - `STRAVA_CLIENT_SECRET`
   - `STRAVA_REFRESH_TOKEN`

### 3. Generate Initial Data

**Locally:**
```bash
# Set environment variables
export GITHUB_TOKEN=ghp_xxx
export GITHUB_USERNAME=mrcodeeu
export STRAVA_CLIENT_ID=xxx
export STRAVA_CLIENT_SECRET=xxx
export STRAVA_REFRESH_TOKEN=xxx

# Generate data
make generate-data

# Check generated files
ls backend/data/generated/
```

**In CI:**
- Workflows will run automatically on schedule
- Or trigger manually: `gh workflow run generate-data.yml -f sources=all`

### 4. Commit Generated Data

```bash
git add backend/data/generated/*.json
git commit -m "chore: add initial generated data"
git push origin dev/refactor-apis-and-workflows
```

### 5. Set Up Deployment (Optional)

Follow `DEPLOYMENT.md` to set up automatic deployment to your homelab.

**Quick option:** Use Watchtower for automatic Docker updates
```yaml
watchtower:
  image: containrrr/watchtower
  volumes:
    - /var/run/docker.sock:/var/run/docker.sock
  command: --interval 300 --cleanup
```

## Testing

### Test Data Generation Locally

```bash
# With verbose output
cd backend
go run cmd/generate/main.go -output ./data/generated -verbose

# Test individual sources
go run cmd/generate/main.go -sources github -verbose
go run cmd/generate/main.go -sources strava -verbose
```

### Test Backend Server

```bash
# Generate data first
make generate-data

# Start server
make dev-backend

# Test endpoints
curl http://localhost:8080/api/cv
curl http://localhost:8080/api/projects
curl http://localhost:8080/api/strava
```

### Test Docker Build

```bash
# Build image
make build-docker

# Run container
make run-docker

# Test
curl http://localhost:8080/api/health
```

### Test GitHub Actions Workflows

```bash
# Trigger data generation manually
gh workflow run generate-data.yml -f sources=all

# Check status
gh run list --workflow=generate-data.yml

# View logs
gh run view <run-id> --log

# Test Docker build
git push origin dev/refactor-apis-and-workflows
gh run list --workflow=docker.yml
```

## What's Not Implemented (Future Work)

### Testing

- ❌ Mocked unit tests for scrapers
- ❌ Real API integration tests
- ❌ Frontend component tests for new data

**Note:** Test infrastructure is in place (Makefile commands exist), but actual test files need to be written.

**To implement:**
1. Create test files in `backend/internal/scrapers/*_test.go`
2. Write mocked tests (default)
3. Write real API tests with `//go:build realapi` tag
4. Run with `make test` (mocked) or `make test-real` (real APIs)

### Frontend UI Updates

The frontend types are updated, but the UI components may need updates to display:
- Rich Strava data (stats, PRs, best activities)
- LinkedIn profile photos
- Activity charts/visualizations

**Current state:** API functions return new data structures, but UI still uses legacy display

### LinkedIn Improvements

- Enhanced scraping with browser automation (Playwright, Selenium)
- Manual data entry form/UI
- Alternative data sources

### Deployment Automation

- Webhook receiver implementation (examples provided in DEPLOYMENT.md)
- Ansible playbook for automated deployment
- Health check monitoring

## File Structure

```
homepage/
├── .github/
│   └── workflows/
│       ├── generate-data.yml    # Scheduled data generation
│       └── docker.yml            # Docker build & push
├── backend/
│   ├── cmd/
│   │   ├── generate/
│   │   │   └── main.go          # Data generation CLI
│   │   └── server/
│   │       └── main.go          # Updated server
│   ├── internal/
│   │   ├── models/
│   │   │   └── data.go          # Data structures
│   │   ├── scrapers/
│   │   │   ├── github.go        # GitHub scraper
│   │   │   ├── strava.go        # NEW: Strava scraper
│   │   │   └── linkedin.go      # NEW: LinkedIn scraper
│   │   └── storage/
│   │       ├── cache.go
│   │       └── loader.go        # NEW: Data loader
│   └── data/
│       └── generated/           # Pre-generated data files
│           ├── github.json
│           ├── strava.json
│           └── linkedin.json
├── frontend/
│   └── src/
│       └── lib/
│           └── api.ts           # Updated types & functions
├── Makefile                     # Updated with new commands
├── SECRETS.md                   # NEW: Secrets documentation
├── DEPLOYMENT.md                # NEW: Deployment guide
└── IMPLEMENTATION_SUMMARY.md   # NEW: This file
```

## Next Steps

1. **Merge this branch:**
   ```bash
   git push origin dev/refactor-apis-and-workflows
   # Create PR and merge to main
   ```

2. **Set up GitHub secrets** (see SECRETS.md)

3. **Trigger initial data generation:**
   ```bash
   gh workflow run generate-data.yml -f sources=all
   ```

4. **Verify Docker build:**
   ```bash
   # Will build automatically on push to main
   # Check at: ghcr.io/mrcodeeu/homepage:latest
   ```

5. **Set up deployment** (see DEPLOYMENT.md)

6. **Future: Implement tests** (when you have sample data)

7. **Future: Update frontend UI** to display rich Strava/LinkedIn data

## Questions or Issues?

- Check `SECRETS.md` for credential setup
- Check `DEPLOYMENT.md` for deployment options
- Review workflow logs: `gh run list`
- Check Docker images: https://github.com/mrcodeeu?tab=packages

## Summary

✅ **Completed:**
- Full data gathering system (GitHub, Strava, LinkedIn)
- CLI tool for local data generation
- GitHub Actions workflows (scheduled + Docker)
- Backend refactored to use pre-generated data
- Frontend types updated
- Comprehensive documentation
- Makefile commands for easy usage

⚠️ **Partially Completed:**
- LinkedIn scraper (MVP, may need manual data)
- Frontend UI (types updated, but components may need updates)

❌ **Not Implemented:**
- Unit tests (mocked & real API tests)
- Frontend component updates for rich data display
- Deployment automation scripts

**Overall:** The core infrastructure is complete and ready to use! 🎉
