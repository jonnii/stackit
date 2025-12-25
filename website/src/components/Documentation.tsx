export default function Documentation() {
  return (
    <section id="documentation">
      <div className="container">
        <h2>Documentation</h2>
        <p style={{ color: 'var(--text-secondary)', marginBottom: '2rem' }}>
          Learn more about Stackit&apos;s features and advanced workflows:
        </p>

        <div className="commands-grid">
          <div className="command-card">
            <h4>📖 Getting Started Guide</h4>
            <p>Complete walkthrough for new users including installation, setup, and your first stack.</p>
          </div>

          <div className="command-card">
            <h4>🎓 Advanced Workflows</h4>
            <p>Learn advanced patterns like insert mode, patch staging, and complex stack management.</p>
          </div>

          <div className="command-card">
            <h4>⚙️ Configuration</h4>
            <p>Customize Stackit&apos;s behavior with repository and global configuration options.</p>
          </div>

          <div className="command-card">
            <h4>🤝 Contributing</h4>
            <p>Want to contribute? Learn about the project structure and development workflow.</p>
          </div>

          <div className="command-card">
            <h4>❓ FAQ</h4>
            <p>Common questions and troubleshooting tips for working with stacked changes.</p>
          </div>

          <div className="command-card">
            <h4>📝 Changelog</h4>
            <p>See what&apos;s new in recent releases and upcoming features on the roadmap.</p>
          </div>
        </div>

        <div style={{ marginTop: '3rem', padding: '2rem', background: 'var(--bg-secondary)', border: '1px solid var(--border)', borderRadius: '8px' }}>
          <h3>Need Help?</h3>
          <p style={{ color: 'var(--text-secondary)', marginBottom: '1rem' }}>
            Run <code className="inline-code">stackit --help</code> or <code className="inline-code">stackit [command] --help</code> to see detailed command information.
          </p>
          <p style={{ color: 'var(--text-secondary)' }}>
            Found a bug or have a feature request?
            <a href="https://github.com/jonnii/stackit/issues" style={{ color: 'var(--accent)', marginLeft: '0.25rem' }}>Open an issue on GitHub</a>.
          </p>
        </div>
      </div>
    </section>
  )
}