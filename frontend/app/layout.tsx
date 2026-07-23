import type { Metadata } from 'next'
import '../styles/globals.css'
import { AppShell } from './app-shell'
import { AuthProvider } from './auth-context'
import { HubProvider } from './hub-context'

export const metadata: Metadata = {
  title: 'Skill Arena',
  description: 'Competitive human skill platform',
  icons: {
    icon: '/favicon.svg',
  },
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <AuthProvider>
          <HubProvider>
            <AppShell>{children}</AppShell>
          </HubProvider>
        </AuthProvider>
      </body>
    </html>
  )
}
