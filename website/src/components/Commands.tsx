import { Subheading } from '@/components/elements/subheading'
import { Text } from '@/components/elements/text'

export default function Commands() {
  return (
    <section id="commands" className="py-16">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight text-mist-900 dark:text-mist-100 sm:text-4xl">
            Core Commands
          </h2>
          <p className="mt-4 text-lg text-mist-600 dark:text-mist-400">
            Stackit provides a comprehensive set of commands for managing your stacked workflow:
          </p>
        </div>

        <div className="mt-16 grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3 font-mono text-mist-600 dark:text-mist-400">stackit init</Subheading>
            <Text>Initialize Stackit in your repository and configure the trunk branch.</Text>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3 font-mono text-mist-600 dark:text-mist-400">stackit create [name]</Subheading>
            <Text>Create a new branch stacked on top of the current branch with your staged changes.</Text>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3 font-mono text-mist-600 dark:text-mist-400">stackit log</Subheading>
            <Text>Display a visual representation of your stacked branches and their relationships.</Text>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3 font-mono text-mist-600 dark:text-mist-400">stackit submit</Subheading>
            <Text>Push branches and create/update pull requests for the entire stack.</Text>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3 font-mono text-mist-600 dark:text-mist-400">stackit restack</Subheading>
            <Text>Rebase branches in the stack to ensure each has its parent in history.</Text>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3 font-mono text-mist-600 dark:text-mist-400">stackit sync</Subheading>
            <Text>Sync your local branches with remote changes and update your stack.</Text>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3 font-mono text-mist-600 dark:text-mist-400">stackit checkout [branch]</Subheading>
            <Text>Switch between branches in your stack with autocomplete support.</Text>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3 font-mono text-mist-600 dark:text-mist-400">stackit merge</Subheading>
            <Text>Merge approved pull requests and clean up merged branches.</Text>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3 font-mono text-mist-600 dark:text-mist-400">stackit squash</Subheading>
            <Text>Squash commits in the current branch while maintaining the stack.</Text>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3 font-mono text-mist-600 dark:text-mist-400">stackit split</Subheading>
            <Text>Split commits into separate branches for better organization.</Text>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3 font-mono text-mist-600 dark:text-mist-400">stackit absorb</Subheading>
            <Text>Automatically absorb uncommitted changes into the appropriate commits.</Text>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <Subheading className="mb-3 font-mono text-mist-600 dark:text-mist-400">stackit info</Subheading>
            <Text>Show detailed information about the current branch and its position in the stack.</Text>
          </div>
        </div>
      </div>
    </section>
  )
}