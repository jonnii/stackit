import Header from '@/components/Header'
import Hero from '@/components/Hero'
import Installation from '@/components/Installation'
import QuickStart from '@/components/QuickStart'
import Commands from '@/components/Commands'
import Features from '@/components/Features'
import Documentation from '@/components/Documentation'
import Footer from '@/components/Footer'
import { Main } from '@/components/elements/main'

export default function Home() {
  return (
    <>
      <Header />
      <Main>
        <Hero />
        <Installation />
        <QuickStart />
        <Commands />
        <Features />
        <Documentation />
      </Main>
      <Footer />
    </>
  )
}