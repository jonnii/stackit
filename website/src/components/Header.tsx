import { PlainButtonLink } from '@/components/elements/button'
import { GitHubIcon } from '@/components/icons/social/github-icon'
import {
  NavbarLink,
  NavbarLogo,
  NavbarWithLinksActionsAndCenteredLogo,
} from '@/components/sections/navbar-with-links-actions-and-centered-logo'

export default function Header() {
  return (
    <NavbarWithLinksActionsAndCenteredLogo
      links={
        <>
          <NavbarLink href="#features">Features</NavbarLink>
          <NavbarLink href="#installation">Installation</NavbarLink>
          <NavbarLink href="#commands">Commands</NavbarLink>
          <NavbarLink href="#documentation">Docs</NavbarLink>
        </>
      }
      logo={
        <NavbarLogo href="#">
          <img
            src="/stackit-logo.svg"
            alt="Stackit"
            className="dark:hidden h-8"
          />
          <img
            src="/stackit-logo-dark.svg"
            alt="Stackit"
            className="not-dark:hidden h-8"
          />
        </NavbarLogo>
      }
      actions={
        <PlainButtonLink href="https://github.com/jonnii/stackit" target="_blank" rel="noopener noreferrer">
          <GitHubIcon />
          View on GitHub
        </PlainButtonLink>
      }
    />
  )
}