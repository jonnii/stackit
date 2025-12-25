export default function Commands() {
  return (
    <section id="commands">
      <div className="container">
        <h2>Core Commands</h2>
        <p style={{ color: 'var(--text-secondary)', marginBottom: '2rem' }}>
          Stackit provides a comprehensive set of commands for managing your stacked workflow:
        </p>

        <div className="commands-grid">
          <div className="command-card">
            <h4>stackit init</h4>
            <p>Initialize Stackit in your repository and configure the trunk branch.</p>
          </div>

          <div className="command-card">
            <h4>stackit create [name]</h4>
            <p>Create a new branch stacked on top of the current branch with your staged changes.</p>
          </div>

          <div className="command-card">
            <h4>stackit log</h4>
            <p>Display a visual representation of your stacked branches and their relationships.</p>
          </div>

          <div className="command-card">
            <h4>stackit submit</h4>
            <p>Push branches and create/update pull requests for the entire stack.</p>
          </div>

          <div className="command-card">
            <h4>stackit restack</h4>
            <p>Rebase branches in the stack to ensure each has its parent in history.</p>
          </div>

          <div className="command-card">
            <h4>stackit sync</h4>
            <p>Sync your local branches with remote changes and update your stack.</p>
          </div>

          <div className="command-card">
            <h4>stackit checkout [branch]</h4>
            <p>Switch between branches in your stack with autocomplete support.</p>
          </div>

          <div className="command-card">
            <h4>stackit merge</h4>
            <p>Merge approved pull requests and clean up merged branches.</p>
          </div>

          <div className="command-card">
            <h4>stackit squash</h4>
            <p>Squash commits in the current branch while maintaining the stack.</p>
          </div>

          <div className="command-card">
            <h4>stackit split</h4>
            <p>Split commits into separate branches for better organization.</p>
          </div>

          <div className="command-card">
            <h4>stackit absorb</h4>
            <p>Automatically absorb uncommitted changes into the appropriate commits.</p>
          </div>

          <div className="command-card">
            <h4>stackit info</h4>
            <p>Show detailed information about the current branch and its position in the stack.</p>
          </div>
        </div>
      </div>
    </section>
  )
}