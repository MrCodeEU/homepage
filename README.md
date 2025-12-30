# Personal Homepage

A modern personal homepage showcasing portfolio projects, CV, and activity stats.

## Features

- 🎨 **Auto-discovery** of portfolio projects from GitHub repositories
- 📊 **Dynamic content** from multiple data sources (GitHub, Strava, LinkedIn)
- 🐳 **Containerized** deployment with Docker
- 🔄 **Real-time updates** via background scrapers
- 🎯 **Portfolio markers** - Flag repos with `.portfolio` file or README markers

## Architecture

- **Frontend**: SvelteKit with TypeScript
- **Backend**: Go with embedded static files
- **Storage**: File-based cache with TTL
- **Deployment**: Single Docker container (~15MB)

## Quick Start

### Using Docker (Recommended)

```bash
docker run -d \
  -p 8080:8080 \
  -e GITHUB_USERNAME=your-username \
  -e GITHUB_TOKEN=your-github-token \
  -v homepage-data:/data \
  ghcr.io/mrcodeeu/homepage:latest
```

Visit http://localhost:8080

### Using Docker Compose

```yaml
services:
  homepage:
    image: ghcr.io/mrcodeeu/homepage:latest
    ports:
      - "8080:8080"
    environment:
      - GITHUB_USERNAME=your-username
      - GITHUB_TOKEN=your-github-token
    volumes:
      - homepage-data:/data

volumes:
  homepage-data:
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GITHUB_USERNAME` | Yes | Your GitHub username for project discovery |
| `GITHUB_TOKEN` | Yes | GitHub Personal Access Token (read-only) |
| `CACHE_DIR` | No | Cache directory (default: `/data/cache`) |

## Portfolio Markers

To feature a repository on your homepage, add one of:

### Method 1: `.portfolio` file (Recommended)

Create `.portfolio` in your repository root:

```json
{
  "description": "Custom description for your project",
  "featured": true,
  "tags": ["golang", "web", "automation"],
  "images": [
    "https://example.com/screenshot.png",
    "./assets/demo.gif"
  ]
}
```

### Method 2: README Marker

Add to your README.md:
```markdown
<!-- PORTFOLIO -->
```

Or use the emoji marker:
```markdown
🎨
```

See [PORTFOLIO.md](./PORTFOLIO.md) for detailed documentation.

## Development

### Local Setup

**Frontend**:
```bash
cd frontend
npm install
npm run dev  # http://localhost:5173
```

**Backend**:
```bash
cd backend
go run cmd/server/main.go  # http://localhost:8080
```

### Building

```bash
docker build -t homepage .
```

## Docker Images

Images are automatically built and published:

- **Production**: `ghcr.io/mrcodeeu/homepage:latest`
- **Development**: `ghcr.io/mrcodeeu/homepage:dev`

## Contributing

Contributions are welcome! Please:

1. Fork this repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file

## Credits

Built with:
- [SvelteKit](https://kit.svelte.dev/)
- [Go](https://golang.org/)
- [Docker](https://www.docker.com/)

---

🤖 This repository is automatically synced from a private monorepo.
