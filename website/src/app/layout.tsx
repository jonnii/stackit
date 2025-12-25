import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'Stackit - Make stacked changes fast & intuitive',
  description: 'A powerful CLI tool for managing stacked Git branches and pull requests. Work faster with better code organization and ship smaller, reviewable PRs.',
  keywords: 'git, stacked changes, stacked diffs, pull requests, github, cli, developer tools, code review',
  authors: [{ name: 'Stackit' }],
  metadataBase: new URL('https://stackit.dev'),
  openGraph: {
    title: 'Stackit - Make stacked changes fast & intuitive',
    description: 'A powerful CLI tool for managing stacked Git branches and pull requests. Ship faster with better code reviews.',
    url: 'https://stackit.dev/',
    siteName: 'Stackit',
    images: [
      {
        url: '/og-image.png',
        width: 1200,
        height: 630,
        alt: 'Stackit - Make stacked changes fast & intuitive',
      },
    ],
    locale: 'en_US',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Stackit - Make stacked changes fast & intuitive',
    description: 'A powerful CLI tool for managing stacked Git branches and pull requests. Ship faster with better code reviews.',
    images: ['/og-image.png'],
  },
  robots: {
    index: true,
    follow: true,
  },
}

export const viewport = {
  width: 'device-width',
  initialScale: 1,
  themeColor: '#0d1117',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}