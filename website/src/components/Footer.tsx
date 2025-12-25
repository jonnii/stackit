import { Container } from '@/components/elements/container'
import { Text } from '@/components/elements/text'

export default function Footer() {
  return (
    <footer className="border-t border-mist-200 bg-mist-50 py-12 dark:border-mist-800 dark:bg-mist-950">
      <Container>
        <Text className="text-center text-mist-600 dark:text-mist-400">
          Built with ❤️ by the Stackit community •{' '}
          <a
            href="https://github.com/jonnii/stackit"
            className="text-mist-600 hover:text-mist-700 dark:text-mist-400 dark:hover:text-mist-300"
          >
            GitHub
          </a>{' '}
          •{' '}
          <a
            href="https://github.com/jonnii/stackit/blob/main/LICENSE"
            className="text-mist-600 hover:text-mist-700 dark:text-mist-400 dark:hover:text-mist-300"
          >
            License
          </a>
        </Text>
      </Container>
    </footer>
  )
}