# Stackit Website

A modern React Next.js website for the Stackit CLI tool.

## Getting Started

### Prerequisites

- Node.js 18+
- npm or yarn

### Installation

```bash
npm install
```

### Development

```bash
# Start development server with hot reload
npm run dev
# or
make dev
```

### Build

```bash
# Build for production
npm run build
# or
make build
```

### Production

```bash
# Start production server
npm start
# or
make run
```

## Project Structure

```
src/
├── app/
│   ├── layout.tsx       # Root layout with metadata
│   ├── page.tsx         # Home page
│   ├── not-found.tsx    # 404 page
│   └── globals.css      # Global styles
└── components/          # React components
    ├── Header.tsx
    ├── Hero.tsx
    ├── Installation.tsx
    ├── QuickStart.tsx
    ├── Commands.tsx
    ├── Features.tsx
    ├── Documentation.tsx
    └── Footer.tsx
```

## Deployment

This is a static Next.js site configured for static export. Build with:

```bash
npm run build
```

The static files will be in the `out/` directory, ready for deployment to any static hosting service.

## Features

- ⚡ Next.js 14 with App Router
- 🎨 GitHub Dark Theme styling
- 📱 Responsive design
- 🔍 SEO optimized
- 🚀 Static site generation