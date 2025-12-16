# Stackit Website

Official website for Stackit, deployed on Railway.

## Structure

```
website/
├── cmd/
│   └── server/          # Go web server
│       └── main.go
├── public/              # Static assets
│   ├── index.html       # Main homepage
│   ├── 404.html         # Custom 404 page
│   ├── favicon.svg      # Site favicon
│   ├── robots.txt       # Search engine directives
│   └── sitemap.xml      # Site map for SEO
├── go.mod               # Go module definition
├── Procfile             # Railway process file
├── railway.json         # Railway configuration
└── nixpacks.toml        # Nixpacks build configuration
```

## Local Development

### Prerequisites

- Go 1.25+
- Make (optional)

### Running Locally

```bash
cd website

# Build the server
go build -o server ./cmd/server

# Run the server
./server

# Or run directly
go run ./cmd/server/main.go
```

The server will start on `http://localhost:8080` (or the port specified by the `PORT` environment variable).

## Deployment

### Railway

This site is configured for automatic deployment on Railway. Simply:

1. Connect your GitHub repository to Railway
2. Railway will automatically detect the configuration
3. The site will build and deploy

### Environment Variables

- `PORT` - Server port (Railway sets this automatically)

### Build Process

Railway uses the configuration in `railway.json` and `nixpacks.toml` to:
1. Install Go 1.25
2. Build the server binary
3. Start the web server serving static files from `public/`

## Features

- ✅ Custom Go web server (no external dependencies)
- ✅ Security headers (CSP, X-Frame-Options, etc.)
- ✅ Custom 404 page
- ✅ SEO optimization (meta tags, sitemap, robots.txt)
- ✅ Social sharing (Open Graph, Twitter Cards)
- ✅ Request logging
- ✅ Static asset caching
- ✅ Mobile responsive design

## Making Changes

### Updating Content

Edit files in the `public/` directory. The server serves these files directly.

### Updating the Server

Edit `cmd/server/main.go` to modify server behavior, add routes, or adjust middleware.

### SEO

- Update `public/sitemap.xml` when adding new pages
- Update meta tags in HTML files for better social sharing
- Keep `public/robots.txt` updated with crawl directives

## Performance

- Static assets cached for 1 year (HTML cached for 1 hour)
- Gzip compression (handled by Railway/CDN)
- Minimal dependencies for fast cold starts

## Security

The server includes security headers:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Content-Security-Policy`
- `Referrer-Policy: strict-origin-when-cross-origin`

## Support

For issues with the website, please open an issue on the main [Stackit repository](https://github.com/jonnii/stackit).
