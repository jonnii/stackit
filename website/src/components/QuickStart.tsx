export default function QuickStart() {
  return (
    <section id="quick-start">
      <div className="container">
        <h2>Quick Start</h2>
        <p style={{ color: 'var(--text-secondary)', marginBottom: '2rem' }}>
          Get up and running with Stackit in minutes. Here&apos;s a typical workflow:
        </p>

        <div className="workflow">
          <div className="workflow-step">
            <div className="step-number">1</div>
            <div className="step-content">
              <h4>Initialize Stackit</h4>
              <p>Set up Stackit in your repository:</p>
              <pre><code>cd your-repo
stackit init</code></pre>
            </div>
          </div>

          <div className="workflow-step">
            <div className="step-number">2</div>
            <div className="step-content">
              <h4>Create a Stacked Branch</h4>
              <p>Create a new branch stacked on top of the current branch:</p>
              <pre><code>stackit create feature-part-1
# Make your changes
git add .
stackit create feature-part-2</code></pre>
            </div>
          </div>

          <div className="workflow-step">
            <div className="step-number">3</div>
            <div className="step-content">
              <h4>View Your Stack</h4>
              <p>Visualize your stacked branches:</p>
              <pre><code>stackit log</code></pre>
            </div>
          </div>

          <div className="workflow-step">
            <div className="step-number">4</div>
            <div className="step-content">
              <h4>Submit Pull Requests</h4>
              <p>Push all branches in the stack and create/update PRs:</p>
              <pre><code>stackit submit</code></pre>
            </div>
          </div>

          <div className="workflow-step">
            <div className="step-number">5</div>
            <div className="step-content">
              <h4>Restack After Changes</h4>
              <p>Automatically rebase branches when the base changes:</p>
              <pre><code>stackit restack</code></pre>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}