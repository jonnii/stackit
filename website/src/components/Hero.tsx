import { ButtonLink, PlainButtonLink } from '@/components/elements/button'
import { ArrowNarrowRightIcon } from '@/components/icons/arrow-narrow-right-icon'

export default function Hero() {
  return (
    <section id="hero" className="py-24">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <h1 className="text-4xl font-bold tracking-tight text-mist-900 dark:text-mist-100 sm:text-6xl">
            Make Stacked Changes Fast & Intuitive
          </h1>
          <p className="mt-6 text-lg leading-8 text-mist-600 dark:text-mist-400">
            A powerful CLI tool for managing stacked Git branches and pull requests. Work faster with better code organization.
          </p>
          <div className="mt-10 flex items-center justify-center gap-x-6">
            <ButtonLink href="#installation">
              Get Started
            </ButtonLink>
            <PlainButtonLink href="#features">
              Learn More <ArrowNarrowRightIcon />
            </PlainButtonLink>
          </div>
        </div>
      </div>
    </section>
  )
}