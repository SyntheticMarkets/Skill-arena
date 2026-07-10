import type { Metadata } from 'next'
import '../styles/globals.css'
import { AppShell } from './app-shell'

export const metadata: Metadata = {
  title: 'Skill Arena',
  description: 'Competitive human skill platform',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <AppShell>{children}</AppShell>
      </body>
    </html>
  )
}
