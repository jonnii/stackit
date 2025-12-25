import { Subheading } from '@/components/elements/subheading'
import { Text } from '@/components/elements/text'

export default function QuickStart() {
  return (
    <section id="quick-start" className="py-16">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight text-mist-900 dark:text-mist-100 sm:text-4xl">
            Quick Start
          </h2>
          <p className="mt-4 text-lg text-mist-600 dark:text-mist-400">
            Get up and running with Stackit in minutes. Here&apos;s a typical workflow:
          </p>
        </div>

        <div className="mt-16 space-y-8">
          <div className="flex gap-6">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-mist-500 text-white">1</div>
            <div>
              <Subheading className="mb-2">Initialize Stackit</Subheading>
              <Text className="mb-4">Set up Stackit in your repository:</Text>
              <pre className="rounded border border-mist-200 bg-mist-50 p-4 text-sm dark:border-mist-800 dark:bg-mist-900"><code>{`cd your-repo
stackit init`}</code></pre>
            </div>
          </div>

          <div className="flex gap-6">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-mist-500 text-white">2</div>
            <div>
              <Subheading className="mb-2">Create a Stacked Branch</Subheading>
              <Text className="mb-4">Create a new branch stacked on top of the current branch:</Text>
              <pre className="rounded border border-mist-200 bg-mist-50 p-4 text-sm dark:border-mist-800 dark:bg-mist-900"><code>{`stackit create feature-part-1
# Make your changes
git add .
stackit create feature-part-2`}</code></pre>
            </div>
          </div>

          <div className="flex gap-6">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-mist-500 text-white">3</div>
            <div>
              <Subheading className="mb-2">View Your Stack</Subheading>
              <Text className="mb-4">Visualize your stacked branches:</Text>
              <pre className="rounded border border-mist-200 bg-mist-50 p-4 text-sm dark:border-mist-800 dark:bg-mist-900"><code>stackit log</code></pre>
            </div>
          </div>

          <div className="flex gap-6">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-mist-500 text-white">4</div>
            <div>
              <Subheading className="mb-2">Submit Pull Requests</Subheading>
              <Text className="mb-4">Push all branches in the stack and create/update PRs:</Text>
              <pre className="rounded border border-mist-200 bg-mist-50 p-4 text-sm dark:border-mist-800 dark:bg-mist-900"><code>stackit submit</code></pre>
            </div>
          </div>

          <div className="flex gap-6">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-mist-500 text-white">5</div>
            <div>
              <Subheading className="mb-2">Restack After Changes</Subheading>
              <Text className="mb-4">Automatically rebase branches when the base changes:</Text>
              <pre className="rounded border border-mist-200 bg-mist-50 p-4 text-sm dark:border-mist-800 dark:bg-mist-900"><code>stackit restack</code></pre>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}