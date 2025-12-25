import { Subheading } from '@/components/elements/subheading'
import { Text } from '@/components/elements/text'

export default function Documentation() {
  return (
    <section id="documentation" className="py-16">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight text-mist-900 dark:text-mist-100 sm:text-4xl">
            Documentation
          </h2>
          <p className="mt-4 text-lg text-mist-600 dark:text-mist-400">
            Learn more about Stackit&apos;s features and advanced workflows:
          </p>
        </div>

        <div className="mt-16 grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3">📖 Getting Started Guide</Subheading>
            <Text>Complete walkthrough for new users including installation, setup, and your first stack.</Text>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3">🎓 Advanced Workflows</Subheading>
            <Text>Learn advanced patterns like insert mode, patch staging, and complex stack management.</Text>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3">⚙️ Configuration</Subheading>
            <Text>Customize Stackit&apos;s behavior with repository and global configuration options.</Text>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3">🤝 Contributing</Subheading>
            <Text>Want to contribute? Learn about the project structure and development workflow.</Text>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3">❓ FAQ</Subheading>
            <Text>Common questions and troubleshooting tips for working with stacked changes.</Text>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3">📝 Changelog</Subheading>
            <Text>See what&apos;s new in recent releases and upcoming features on the roadmap.</Text>
          </div>
        </div>

        <div className="mt-16 rounded-lg border border-mist-200 bg-mist-50 p-8 dark:border-mist-800 dark:bg-mist-900">
          <Subheading className="mb-4">Need Help?</Subheading>
          <Text className="mb-4">
            Run <code className="rounded bg-white px-2 py-1 text-sm dark:bg-mist-800">stackit --help</code> or{' '}
            <code className="rounded bg-white px-2 py-1 text-sm dark:bg-mist-800">stackit [command] --help</code> to see detailed command information.
          </Text>
          <Text>
            Found a bug or have a feature request?{' '}
            <a
              href="https://github.com/jonnii/stackit/issues"
              className="text-mist-600 hover:text-mist-700 dark:text-mist-400 dark:hover:text-mist-300"
            >
              Open an issue on GitHub
            </a>.
          </Text>
        </div>
      </div>
    </section>
  )
}