# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A personal homepage with auto-discovery of portfolio projects from GitHub, built with Go backend and SvelteKit frontend. The backend embeds the compiled frontend as static files, deployed as a single ~15MB Docker container.

## Development Commands

### Setup
```bash
make install              # Install all dependencies (frontend + backend)
```

### Running Locally
Development requires two terminals:
```bash
# Terminal 1 - Backend on :8080
make dev-backend

# Terminal 2 - Frontend dev server on :5173
make dev-frontend
```

Visit http://localhost:5173 for development (proxies API calls to :8080)

### Testing
```bash
make test                 # Run all tests (frontend + backend)
make test-backend         # Go tests with race detector and coverage
make test-frontend        # Vitest tests
make test-watch           # Frontend tests in watch mode

# Backend coverage report
cd backend && go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

# Single test file or function
cd backend && go test -v ./internal/scrapers -run TestGitHubScraper
cd frontend && npm run test -- src/lib/api.test.ts
```

### Linting & Formatting
```bash
make lint                 # Run all linters
make lint-backend         # golangci-lint (auto-installs if missing)
make lint-frontend        # ESLint
make format               # Format all code (Prettier + gofmt)
make check-all            # Comprehensive check: lint + test + type-check
```

### Docker
```bash
make build-docker         # Build Docker image (multi-stage: node + go)
make run-docker           # Run container on :8080
```

### Building for Production
```bash
# Frontend (creates build/ directory)
cd frontend && npm run build

# Backend (embeds static files)
cd backend && go build -o homepage ./cmd/server
```

## Architecture

### High-Level Structure

```
┌─────────────────┐
│  SvelteKit SPA  │  (Port 5173 dev, embedded in prod)
│  Static Build   │
└────────┬────────┘
         │ HTTP API calls
         ↓
┌─────────────────┐
│   Go Server     │  (Port 8080)
│  ├─ HTTP Router │
│  ├─ Scrapers    │  (GitHub API integration)
│  └─ File Cache  │  (1hr TTL, JSON storage)
└─────────────────┘
         │
         ↓
┌─────────────────┐
│  External APIs  │
│  └─ GitHub API  │  (Projects with .portfolio marker)
└─────────────────┘
```

### Backend Architecture (Go)

**Entry Point:** `backend/cmd/server/main.go`
- Embeds static frontend files via `//go:embed all:static`
- Serves static files at root `/`
- Exposes API endpoints at `/api/*`
- Uses middleware: CORS (dev only), logging

**API Endpoints:**
- `GET /api/health` - Health check
- `GET /api/cv` - CV data (currently mock)
- `GET /api/projects` - GitHub projects with portfolio markers (cached 1hr)
- `GET /api/strava` - Strava stats (currently mock)

**Internal Packages:**
- `internal/config/` - Environment-based config (Port, GitHub token/username, cache settings)
- `internal/scrapers/` - GitHub API client with portfolio discovery logic
  - Checks for `.portfolio` file (JSON) or `<!-- PORTFOLIO -->` / 🎨 in README
  - Extracts images from `.portfolio` and README markdown
  - Converts relative image paths to raw.githubusercontent.com URLs
- `internal/storage/` - File-based cache with TTL (JSON files in CACHE_DIR)

**Key Implementation Details:**
- GitHub API uses Bearer token authentication
- Cache keys are sanitized for filesystem safety
- Portfolio metadata merges .portfolio JSON with repo metadata
- Image URLs: relative paths → `https://raw.githubusercontent.com/{user}/{repo}/main/{path}`

### Frontend Architecture (SvelteKit)

**Build Configuration:**
- Uses `adapter-static` to generate static SPA
- Output: `frontend/build/` with SPA fallback to `index.html`
- Dev server: Vite on port 5173
- API calls: Dev uses `http://localhost:8080`, prod uses same-origin

**Structure:**
- `src/lib/api.ts` - Type-safe API client with fetch wrappers
- `src/routes/` - SvelteKit file-based routing
  - `+page.svelte` - Main page
  - `+layout.svelte` - Layout wrapper
- `src/app.html` - HTML template

**Testing:**
- Vitest + jsdom for unit tests
- Testing Library for component tests
- Config: `vitest.config.ts` with jsdom environment

### Docker Build Process (Multi-Stage)

**Stage 1 - Frontend Builder:**
1. `node:20-alpine` base
2. `npm ci` → `npm run build` → outputs to `build/`

**Stage 2 - Backend Builder:**
1. `golang:1.23-alpine` base
2. Copies frontend `build/` to `backend/cmd/server/static/`
3. Go build with embedded static files (`//go:embed`)
4. CGO_ENABLED=0 for static binary

**Stage 3 - Runtime:**
1. `alpine:latest` (~15MB final image)
2. Copies binary, creates non-root user
3. Exposes port 8080, includes health check

## Portfolio System

Projects are auto-discovered from GitHub when they have:

**Option 1: .portfolio file (recommended)**
```json
{
  "description": "Custom description",
  "featured": true,
  "tags": ["golang", "web"],
  "images": ["screenshots/demo.png", "https://example.com/banner.jpg"],
  "links": [
    {"name": "Live", "url": "https://example.com"},
    {"name": "Staging", "url": "https://staging.example.com"}
  ]
}
```

**Option 2: README markers**
- `<!-- PORTFOLIO -->` HTML comment
- 🎨 emoji

**Frontend Features:**
- Image carousel with auto-switching (3s interval), pause on hover
- Link buttons with auto-detected icons based on name (Live→globe, Staging→flask, Docs→book, Demo→play)
- GitHub link always shown alongside custom links
- Badge images (shields.io) are automatically filtered from carousels

See PORTFOLIO.md for full documentation.

## Environment Variables

**Required:**
- `GITHUB_USERNAME` - GitHub username for project discovery
- `GITHUB_TOKEN` - GitHub Personal Access Token (public_repo scope)

**Optional:**
- `PORT` - Server port (default: 8080)
- `CACHE_DIR` - Cache directory (default: ./data/cache)
- `STRAVA_*` / `LINKEDIN_*` - Future integrations

## Testing Strategy

- **Backend:** Table-driven tests, httptest for handlers, coverage target >80%
- **Frontend:** Vitest with mocked fetch, Testing Library for components
- **CI:** GitHub Actions runs tests + linting + Docker build

See TESTING.md for detailed guidance.

## Common Patterns

### Adding a New API Endpoint

1. Add handler function in `backend/cmd/server/main.go`
2. Register route in `main()`: `mux.HandleFunc("/api/...", handler)`
3. Add TypeScript types and client function in `frontend/src/lib/api.ts`
4. Write tests: `backend/cmd/server/main_test.go` and `frontend/src/lib/api.test.ts`

### Adding a New Scraper

1. Implement `Scraper` interface in `backend/internal/scrapers/`:
   - `Name() string`
   - `Scrape() (any, error)` - Fetch fresh data
   - `GetCached() (any, error)` - Return cached or fresh
   - `Refresh() (any, error)` - Force refresh and update cache
2. Initialize in `main()` with cache instance
3. Add API endpoint to expose data
4. Add integration tests

### Cache Behavior

- Default TTL: 1 hour (configurable in scraper)
- Cache files: JSON with metadata (`{data, expires_at}`)
- Auto-cleanup: Expired entries deleted on read
- Manual clear: Implement refresh endpoint or use cache.Clear()

## Important Notes

- Frontend build output MUST go to `frontend/build/` (adapter-static config)
- Static files are embedded at Go compile time, not runtime
- CORS allows `http://localhost:5173` for dev, same-origin only in prod
- GitHub API rate limit: 60/hr unauthenticated, 5000/hr authenticated
- Cache directory must be writable (Docker uses volume at `/data`)