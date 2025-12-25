export default function Features() {
  return (
    <section id="features">
      <div className="container">
        <h2>Why Stackit?</h2>

        <div className="features">
          <div className="feature">
            <div className="feature-icon">🚀</div>
            <h3>Ship Faster</h3>
            <p>Break large changes into smaller, reviewable PRs without waiting for sequential approval.</p>
          </div>

          <div className="feature">
            <div className="feature-icon">🎯</div>
            <h3>Better Reviews</h3>
            <p>Reviewers can focus on smaller, logical changes rather than massive diffs.</p>
          </div>

          <div className="feature">
            <div className="feature-icon">🔄</div>
            <h3>Easy Rebasing</h3>
            <p>Automatically restack branches when the base changes. No manual rebasing headaches.</p>
          </div>

          <div className="feature">
            <div className="feature-icon">🎨</div>
            <h3>Visual Clarity</h3>
            <p>See your entire stack at a glance with beautiful terminal visualizations.</p>
          </div>

          <div className="feature">
            <div className="feature-icon">⚡</div>
            <h3>Intuitive CLI</h3>
            <p>Simple commands that feel natural. Built for developer productivity.</p>
          </div>

          <div className="feature">
            <div className="feature-icon">🔧</div>
            <h3>GitHub Integration</h3>
            <p>Seamlessly create and update PRs with proper dependencies and metadata.</p>
          </div>
        </div>
      </div>
    </section>
  )
}