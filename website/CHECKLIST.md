# Website Launch Checklist

Use this checklist before and after deploying to production.

## Pre-Deployment

### Content Review
- [ ] Proofread all text content
- [ ] Verify all command examples are correct
- [ ] Check all links work (especially GitHub links)
- [ ] Verify installation instructions are up-to-date
- [ ] Review feature descriptions for accuracy

### Technical Setup
- [ ] Test build locally: `make build`
- [ ] Test server locally: `make run`
- [ ] Verify all static files in `public/`
- [ ] Check `go.mod` version matches available Go
- [ ] Review `railway.json` configuration
- [ ] Update `robots.txt` with production domain
- [ ] Update `sitemap.xml` with production URLs

### SEO & Social
- [ ] Update meta description
- [ ] Verify OG tags are correct
- [ ] Update Twitter Card tags
- [ ] Create/optimize og-image.png (current is SVG placeholder)
- [ ] Set canonical URL to production domain
- [ ] Verify favicon displays correctly

### Security
- [ ] Review CSP headers in server code
- [ ] Check no sensitive data in code
- [ ] Verify HTTPS will be enabled
- [ ] Test security headers with securityheaders.com

## Deployment

### Railway Setup
- [ ] Connect GitHub repository to Railway
- [ ] Verify environment variables (if any)
- [ ] Configure custom domain in Railway
- [ ] Wait for initial deployment to complete
- [ ] Check deployment logs for errors

### DNS Configuration
- [ ] Add CNAME record pointing to Railway
- [ ] Wait for DNS propagation (can take up to 48hrs)
- [ ] Verify SSL certificate is issued
- [ ] Test HTTPS redirect works

## Post-Deployment

### Functionality Tests
- [ ] Homepage loads correctly
- [ ] All sections render properly
- [ ] Navigation/anchor links work
- [ ] Code blocks display correctly
- [ ] Responsive design works on mobile
- [ ] Custom 404 page shows for bad URLs
- [ ] Favicon appears in browser tab
- [ ] Test in multiple browsers (Chrome, Firefox, Safari)

### SEO Verification
- [ ] robots.txt accessible at `/robots.txt`
- [ ] sitemap.xml accessible at `/sitemap.xml`
- [ ] Meta tags correct (test with https://metatags.io)
- [ ] Social sharing works (test on Twitter, LinkedIn)
- [ ] Submit sitemap to Google Search Console
- [ ] Submit site to Bing Webmaster Tools

### Performance
- [ ] Test load time (should be < 2s)
- [ ] Check Lighthouse score (aim for 90+)
- [ ] Verify caching headers work
- [ ] Test from different geographic locations
- [ ] Mobile performance check

### Monitoring Setup
- [ ] Add analytics (Google Analytics, Plausible, etc.)
- [ ] Set up uptime monitoring (optional)
- [ ] Configure error tracking (optional)
- [ ] Set up Railway notifications for deployments

### Final Polish
- [ ] Update GitHub repo with production URL
- [ ] Add website badge to main README
- [ ] Announce on social media
- [ ] Share with team/community

## Ongoing Maintenance

### Regular Tasks
- [ ] Monitor Railway logs weekly
- [ ] Check analytics monthly
- [ ] Update content as features are added
- [ ] Keep dependencies updated: `go get -u`
- [ ] Review and respond to issues

### When Updating Content
- [ ] Test changes locally first
- [ ] Update sitemap if adding new pages
- [ ] Verify build succeeds before pushing
- [ ] Check deployment completes successfully
- [ ] Quick smoke test after deployment

### Performance Monitoring
- [ ] Monthly Lighthouse audit
- [ ] Check Core Web Vitals
- [ ] Monitor Railway resource usage
- [ ] Optimize images if added

## Emergency Procedures

### If Site Goes Down
1. Check Railway dashboard for errors
2. Review recent deployments
3. Check Railway status page
4. Rollback to last working deployment if needed
5. Review logs for root cause

### If Build Fails
1. Check Railway build logs
2. Verify Go version compatibility
3. Test build locally
4. Check for missing files
5. Verify configuration files unchanged

### Quick Rollback
```bash
# Via Railway Dashboard
1. Go to Deployments
2. Find last working deployment
3. Click "Redeploy"

# Via CLI
railway rollback
```

## Resources

- **Railway Docs**: https://docs.railway.app
- **Go Documentation**: https://go.dev/doc/
- **SEO Testing**: https://metatags.io
- **Performance**: https://pagespeed.web.dev
- **Security Headers**: https://securityheaders.com
- **SSL Test**: https://www.ssllabs.com/ssltest/

## Notes

- Railway has a free tier with $5 credit/month
- Static sites typically use minimal resources
- Auto-deploys on every push to main branch
- Zero-downtime deployments by default
