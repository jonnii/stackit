# Deployment Guide

## Railway Deployment

### Prerequisites

1. A [Railway](https://railway.app) account
2. GitHub repository connected to Railway
3. (Optional) Custom domain

### Initial Setup

1. **Connect Repository**
   - Log in to Railway
   - Click "New Project"
   - Select "Deploy from GitHub repo"
   - Choose your Stackit repository

2. **Configure Service**
   - Railway will auto-detect the configuration from `railway.json` at the repository root
   - The build will:
     - Change to the `website/cmd/server` directory
     - Build the Go server and place the binary at the repository root
     - Start serving from the `website/` directory

3. **Environment Variables**
   - Railway automatically sets `PORT`
   - No additional variables required
   - Optional: Add analytics IDs if needed

4. **Custom Domain** (Optional)
   - Go to your service settings
   - Click "Settings" → "Domains"
   - Add your custom domain (e.g., `stackit.dev`)
   - Update DNS records as instructed
   - Update `public/sitemap.xml` and `public/robots.txt` with new domain

### Deployment Process

Railway deploys automatically on every push to your main branch:

1. Push changes to GitHub
2. Railway detects the push
3. Builds the project using `railway.json` configuration
4. Deploys the new version with zero downtime
5. Health check ensures the service is running

### Manual Deployment

Using Railway CLI:

```bash
# Install Railway CLI
npm i -g @railway/cli

# Login
railway login

# Link to project
railway link

# Deploy
cd website
railway up
```

### Build Configuration

The deployment uses two configuration files:

1. **`railway.json`** - Railway-specific build and deploy settings (uses Railpack)
2. **`Procfile`** - Process definition (web server)

### Monitoring

- **Logs**: Available in Railway dashboard under "Deployments"
- **Metrics**: CPU, Memory, Network usage in Railway dashboard
- **Health**: Railway automatically monitors service health

### Rolling Back

If a deployment fails:

1. Go to Railway dashboard
2. Click "Deployments"
3. Find the last working deployment
4. Click "Redeploy"

## Alternative Platforms

### Vercel

While Vercel is primarily for Node.js, you can deploy using a custom runtime:

1. Add `vercel.json`:
```json
{
  "builds": [
    {
      "src": "website/cmd/server/main.go",
      "use": "@vercel/go"
    }
  ],
  "routes": [
    {
      "src": "/(.*)",
      "dest": "/website/cmd/server/main.go"
    }
  ]
}
```

### Fly.io

1. Install flyctl
2. Create `fly.toml`:
```toml
app = "stackit"

[build]
  builder = "paketobuildpacks/builder:base"
  buildpacks = ["gcr.io/paketo-buildpacks/go"]

[[services]]
  internal_port = 8080
  protocol = "tcp"

  [[services.ports]]
    port = 80
    handlers = ["http"]
  
  [[services.ports]]
    port = 443
    handlers = ["tls", "http"]
```

3. Deploy: `flyctl deploy`

### Render

1. Create new "Web Service"
2. Connect GitHub repository
3. Configure:
   - **Build Command**: `cd website && go build -o server ./cmd/server`
   - **Start Command**: `cd website && ./server`
   - **Environment**: Go

### Heroku

1. Add `heroku.yml`:
```yaml
build:
  docker:
    web: Dockerfile
```

2. Create `Dockerfile`:
```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY website/ .
RUN go build -o server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
COPY --from=builder /app/public ./public
CMD ["./server"]
```

3. Deploy: `git push heroku main`

## Post-Deployment Checklist

- [ ] Website loads at production URL
- [ ] All pages accessible (including 404)
- [ ] Favicon displays correctly
- [ ] Meta tags correct (check with https://metatags.io)
- [ ] Mobile responsive (test on devices)
- [ ] SSL certificate active (HTTPS)
- [ ] robots.txt accessible
- [ ] sitemap.xml accessible
- [ ] Update sitemap and robots.txt with production domain
- [ ] Test social sharing (Twitter, LinkedIn, Discord)
- [ ] Set up monitoring/analytics (optional)
- [ ] Configure custom domain DNS
- [ ] Test all internal links

## Troubleshooting

### Build Fails

- Railpack automatically detects the Go version from `website/go.mod`
- Verify all files are committed and pushed
- Check Railway build logs for specific errors

### Server Won't Start

- Ensure PORT environment variable is read correctly
- Check server logs in Railway dashboard
- Verify `public/` directory structure is correct

### 404 Errors

- Ensure all static files are in `website/public/`
- Check file paths are case-sensitive
- Verify server is serving from correct directory

### Performance Issues

- Enable CDN in Railway settings
- Optimize images (if added)
- Check cache headers are working
- Monitor resource usage in dashboard

## Security

- HTTPS is automatic on Railway
- Security headers configured in server
- No sensitive data in environment (for this static site)
- Regular dependency updates: `go get -u && go mod tidy`

## Cost Optimization

Railway pricing:
- Free tier: $5 credit per month
- Static site typically uses minimal resources
- Estimated cost: Free to $5/month depending on traffic

Tips:
- Images should be optimized
- Enable caching (already configured)
- Use Railway's CDN for static assets
