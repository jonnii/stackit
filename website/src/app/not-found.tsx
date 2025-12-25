import Header from '@/components/Header'
import Footer from '@/components/Footer'

export const metadata = {
  title: '404 - Page Not Found | Stackit',
  description: 'The page you\'re looking for doesn\'t exist.',
}

export default function NotFound() {
  return (
    <>
      <Header />
      <main style={{ padding: '4rem 0', textAlign: 'center' }}>
        <div className="container">
          <h1 style={{ fontSize: '4rem', marginBottom: '1rem', color: 'var(--text-primary)' }}>404</h1>
          <h2 style={{ fontSize: '2rem', marginBottom: '2rem', color: 'var(--text-secondary)' }}>Page Not Found</h2>
          <p style={{ color: 'var(--text-secondary)', marginBottom: '2rem' }}>
            The page you&apos;re looking for doesn&apos;t exist.
          </p>
          <a
            href="/"
            style={{
              display: 'inline-block',
              padding: '0.75rem 1.5rem',
              background: 'var(--accent)',
              color: 'var(--bg-primary)',
              textDecoration: 'none',
              borderRadius: '6px',
              fontWeight: '600'
            }}
          >
            Go Home
          </a>
        </div>
      </main>
      <Footer />
    </>
  )
}