export default function Installation() {
  return (
    <section id="installation">
      <div className="container">
        <h2>Installation</h2>

        <div className="install-methods">
          <div className="install-card">
            <h3>
              Build from Source
              <span className="badge">Recommended</span>
            </h3>
            <p>Clone the repository and build using Go or Just:</p>
            <pre><code># Using Go
git clone https://github.com/jonnii/stackit.git
cd stackit
go build -o stackit ./cmd/stackit

# Or using Just (if installed)
just build</code></pre>
          </div>

          <div className="install-card">
            <h3>
              Homebrew
              <span className="badge coming-soon">Coming Soon</span>
            </h3>
            <p>Install via Homebrew (macOS and Linux):</p>
            <pre><code>brew install stackit</code></pre>
          </div>

          <div className="install-card">
            <h3>
              Binary Release
              <span className="badge coming-soon">Coming Soon</span>
            </h3>
            <p>Download pre-built binaries from GitHub releases:</p>
            <pre><code># Download for your platform
curl -L https://github.com/jonnii/stackit/releases/latest/download/stackit-[platform] -o stackit
chmod +x stackit</code></pre>
          </div>
        </div>

        <h3>System Requirements</h3>
        <ul style={{ color: 'var(--text-secondary)', marginLeft: '2rem', marginTop: '1rem' }}>
          <li>Go 1.25+ (for building from source)</li>
          <li>Git 2.23+</li>
          <li>GitHub CLI (optional, for enhanced PR features)</li>
        </ul>
      </div>
    </section>
  )
}