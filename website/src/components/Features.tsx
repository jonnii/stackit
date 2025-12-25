import { Subheading } from '@/components/elements/subheading'
import { Text } from '@/components/elements/text'

export default function Features() {
  return (
    <section id="features" className="py-16">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight text-mist-900 dark:text-mist-100 sm:text-4xl">
            Why Stackit?
          </h2>
        </div>

        <div className="mt-16 grid gap-8 md:grid-cols-2 lg:grid-cols-3">
          <div className="text-center">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-mist-100 text-3xl dark:bg-mist-900">🚀</div>
            <Subheading className="mb-3">Ship Faster</Subheading>
            <Text>Break large changes into smaller, reviewable PRs without waiting for sequential approval.</Text>
          </div>

          <div className="text-center">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-mist-100 text-3xl dark:bg-mist-900">🎯</div>
            <Subheading className="mb-3">Better Reviews</Subheading>
            <Text>Reviewers can focus on smaller, logical changes rather than massive diffs.</Text>
          </div>

          <div className="text-center">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-mist-100 text-3xl dark:bg-mist-900">🔄</div>
            <Subheading className="mb-3">Easy Rebasing</Subheading>
            <Text>Automatically restack branches when the base changes. No manual rebasing headaches.</Text>
          </div>

          <div className="text-center">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-mist-100 text-3xl dark:bg-mist-900">🎨</div>
            <Subheading className="mb-3">Visual Clarity</Subheading>
            <Text>See your entire stack at a glance with beautiful terminal visualizations.</Text>
          </div>

          <div className="text-center">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-mist-100 text-3xl dark:bg-mist-900">⚡</div>
            <Subheading className="mb-3">Intuitive CLI</Subheading>
            <Text>Simple commands that feel natural. Built for developer productivity.</Text>
          </div>

          <div className="text-center">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-mist-100 text-3xl dark:bg-mist-900">🔧</div>
            <Subheading className="mb-3">GitHub Integration</Subheading>
            <Text>Seamlessly create and update PRs with proper dependencies and metadata.</Text>
          </div>
        </div>
      </div>
    </section>
  )
}