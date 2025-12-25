import { Subheading } from '@/components/elements/subheading'
import { Text } from '@/components/elements/text'

export default function Installation() {
  return (
    <section id="installation" className="py-16">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight text-mist-900 dark:text-mist-100 sm:text-4xl">
            Installation
          </h2>
        </div>

        <div className="mt-16 grid gap-8 md:grid-cols-3">
          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <div className="mb-4 flex items-center gap-3">
              <Subheading>Build from Source</Subheading>
              <span className="rounded bg-mist-500 px-2 py-1 text-xs font-medium text-white">Recommended</span>
            </div>
            <Text className="mb-4">Clone the repository and build using Go or Just:</Text>
            <pre className="rounded border border-mist-200 bg-mist-50 p-4 text-sm dark:border-mist-800 dark:bg-mist-900"><code>{`# Using Go
git clone https://github.com/jonnii/stackit.git
cd stackit
go build -o stackit ./cmd/stackit

# Or using Just (if installed)
just build`}</code></pre>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <div className="mb-4 flex items-center gap-3">
              <Subheading>Homebrew</Subheading>
              <span className="rounded bg-mist-300 px-2 py-1 text-xs font-medium text-mist-700 dark:bg-mist-700 dark:text-mist-300">Coming Soon</span>
            </div>
            <Text className="mb-4">Install via Homebrew (macOS and Linux):</Text>
            <pre className="rounded border border-mist-200 bg-mist-50 p-4 text-sm dark:border-mist-800 dark:bg-mist-900"><code>brew install stackit</code></pre>
          </div>

          <div className="rounded-lg border border-mist-200 bg-white p-6 dark:border-mist-800 dark:bg-mist-950">
            <div className="mb-4 flex items-center gap-3">
              <Subheading>Binary Release</Subheading>
              <span className="rounded bg-mist-300 px-2 py-1 text-xs font-medium text-mist-700 dark:bg-mist-700 dark:text-mist-300">Coming Soon</span>
            </div>
            <Text className="mb-4">Download pre-built binaries from GitHub releases:</Text>
            <pre className="rounded border border-mist-200 bg-mist-50 p-4 text-sm dark:border-mist-800 dark:bg-mist-900"><code>{`# Download for your platform
curl -L https://github.com/jonnii/stackit/releases/latest/download/stackit-[platform] -o stackit
chmod +x stackit`}</code></pre>
          </div>
        </div>

        <div className="mt-16">
          <Subheading>System Requirements</Subheading>
          <ul className="mt-4 list-disc pl-6 text-mist-600 dark:text-mist-400">
            <li>Go 1.25+ (for building from source)</li>
            <li>Git 2.23+</li>
            <li>GitHub CLI (optional, for enhanced PR features)</li>
          </ul>
        </div>
      </div>
    </section>
  )
}