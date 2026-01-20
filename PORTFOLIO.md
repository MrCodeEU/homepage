# Portfolio System Documentation

This document explains how to mark your repositories for inclusion in the homepage portfolio section.

## Overview

The homepage automatically discovers and displays your GitHub projects that are marked as portfolio items. Projects can be marked using either a `.portfolio` file or README markers.

## Method 1: Using `.portfolio` File (Recommended)

Create a `.portfolio` file in the root of your repository with the following JSON structure:

```json
{
  "description": "Custom description that overrides the GitHub repo description",
  "featured": true,
  "tags": ["golang", "web", "automation"],
  "images": [
    "screenshots/demo.png",
    "./assets/logo.svg",
    "https://example.com/external-image.jpg"
  ],
  "links": [
    {"name": "Live", "url": "https://myproject.com"},
    {"name": "Staging", "url": "https://staging.myproject.com"}
  ]
}
```

### Fields

- **description** (optional): Custom description to display instead of the GitHub repo description
- **featured** (optional): Set to `true` to mark this as a featured project
- **tags** (optional): Array of tags to categorize the project (merged with GitHub topics)
- **images** (optional): Array of image URLs or paths (see Image Handling below)
- **links** (optional): Array of custom links to display alongside the GitHub link (see Links below)

### Image Handling

The `images` array supports three types of image references:

1. **Relative paths** - Images stored in your repository:
   ```json
   "images": [
     "screenshots/demo.png",
     "./docs/architecture.svg",
     "banner.jpg"
   ]
   ```
   These will be automatically converted to:
   ```
   https://raw.githubusercontent.com/{username}/{repo}/main/{path}
   ```

2. **Absolute paths in repo** - Starting with `/`:
   ```json
   "images": ["/assets/logo.png"]
   ```
   The leading `/` is removed and converted to the raw GitHub URL.

3. **External URLs** - Hosted elsewhere:
   ```json
   "images": [
     "https://example.com/hosted-image.png",
     "http://cdn.example.com/banner.jpg"
   ]
   ```
   These are used as-is without modification.

### Links

The `links` array allows you to add custom buttons that appear alongside the GitHub link on project cards:

```json
"links": [
  {"name": "Live", "url": "https://myproject.com"},
  {"name": "Staging", "url": "https://staging.myproject.com"},
  {"name": "Docs", "url": "https://docs.myproject.com"}
]
```

Each link object has:
- **name**: Display text for the button (also determines the icon if no custom icon specified)
- **url**: The URL to link to
- **icon** (optional): Custom icon name (e.g., `"mdi:rocket-launch"`)

**Custom Icons:**

You can specify a custom icon using the `icon` field:

```json
"links": [
  {"name": "Launch", "url": "https://app.example.com", "icon": "mdi:rocket-launch"},
  {"name": "API", "url": "https://api.example.com", "icon": "mdi:api"}
]
```

Supported custom icon names:
- `rocket-launch` or `rocket` - Rocket icon
- `api` - API/menu icon
- `download` - Download icon
- `globe` - World/globe icon
- `flask` - Test tube/flask icon
- `book` - Book/documentation icon
- `play` - Play button icon
- `link` - Generic link icon

**Automatic Icon Detection (Fallback):**

If no custom icon is specified, icons are automatically assigned based on keywords in the link name:

| Keyword in name | Icon |
|-----------------|------|
| `live`, `prod` | Globe (world icon) |
| `staging`, `dev`, `test` | Flask (test tube) |
| `docs`, `documentation` | Book |
| `demo` | Play button |
| `api` | API/menu icon |
| `download` | Download icon |
| Other | Generic link icon |

The GitHub link is always shown automatically - you don't need to add it to the links array.

### Complete Example

```json
{
  "description": "A powerful automation tool for managing homelab infrastructure with Ansible, Docker, and continuous deployment via GitHub Actions.",
  "featured": true,
  "tags": ["ansible", "docker", "automation", "devops", "infrastructure"],
  "images": [
    "docs/screenshots/dashboard.png",
    "./assets/architecture-diagram.svg"
  ],
  "links": [
    {"name": "Live", "url": "https://myproject.com"},
    {"name": "Staging", "url": "https://dev.myproject.com"},
    {"name": "Docs", "url": "https://docs.myproject.com"}
  ]
}
```

## Method 2: Using README Markers

If you don't want to create a `.portfolio` file, you can mark your repository by adding one of these markers to your README:

### HTML Comment Marker
```markdown
<!-- PORTFOLIO -->
```

### Unicode Marker
```markdown
🎨
```

When using README markers:
- The GitHub repository description will be used
- Featured status defaults to `false`
- Tags come from GitHub topics only
- Images are automatically extracted from `![alt](url)` markdown syntax in the README

## How It Works

1. The scraper scans your GitHub repositories
2. For each repo, it checks for:
   - A `.portfolio` file in the root
   - OR `<!-- PORTFOLIO -->` comment in README
   - OR 🎨 emoji in README
3. If found, the repo is included in the portfolio
4. Images are collected from:
   - The `images` array in `.portfolio` (if present)
   - Any `![alt](url)` markdown images in the README
5. All relative image paths are converted to absolute GitHub raw URLs
6. Badge images (shields.io) are filtered out
7. Duplicate images are removed
8. Links from `.portfolio` are added as buttons alongside the GitHub link

## Testing Locally

To test your portfolio configuration locally:

1. Add the `.portfolio` file or README marker to your repository
2. Set environment variables:
   ```bash
   export GITHUB_USERNAME=your-username
   export GITHUB_TOKEN=your-github-token
   ```
3. Run the backend:
   ```bash
   cd apps/homepage
   make dev-backend
   ```
4. Check the API response:
   ```bash
   curl http://localhost:8080/api/projects | jq
   ```

## Image Carousel

When a project has multiple images, they are displayed in an auto-rotating carousel:

- **Auto-switching**: Images rotate every 3 seconds
- **Pause on hover**: Carousel pauses when you hover over it
- **Manual navigation**: Arrow buttons and dot indicators for manual control
- **Image counter**: Shows current position (e.g., "2 / 5")

**Note:** Badge images from shields.io are automatically filtered out from the carousel to only show actual project screenshots.

## Image Best Practices

1. **Use local images for privacy**: Store screenshots and project images in your repo rather than external services
2. **Organize images**: Create a dedicated directory like `screenshots/`, `docs/images/`, or `assets/`
3. **Optimize sizes**: Keep images reasonably sized (< 500KB) for faster loading
4. **Use descriptive names**: `feature-dashboard.png` is better than `img1.png`
5. **Supported formats**: PNG, JPG, GIF, SVG, WebP
6. **Avoid badges in images array**: Shield.io badges are auto-filtered, but it's cleaner to only include actual screenshots

## Example Directory Structure

```
your-repo/
├── .portfolio                 # Portfolio configuration
├── README.md
├── screenshots/
│   ├── main-interface.png     # Referenced as "screenshots/main-interface.png"
│   ├── mobile-view.png
│   └── settings.png
├── docs/
│   └── architecture.svg       # Referenced as "docs/architecture.svg"
└── src/
    └── ...
```

## Cache Behavior

- Portfolio data is cached for 1 hour by default
- The cache refreshes automatically when expired
- Manual refresh endpoint: `GET /api/refresh/github` (if implemented)

## Troubleshooting

**My repo isn't showing up:**
- Ensure the repository is **public**
- Verify the `.portfolio` file has valid JSON syntax
- Check that the README marker is on its own line
- Confirm `GITHUB_USERNAME` matches your GitHub username

**Images aren't loading:**
- Verify the file path is correct (case-sensitive)
- Ensure the image exists in the `main` branch
- Check that the image file is committed and pushed
- Test the raw GitHub URL directly in your browser

**Custom description not appearing:**
- Ensure the `description` field in `.portfolio` is not empty
- Check JSON syntax with a validator
- Verify the `.portfolio` file is in the repository root
