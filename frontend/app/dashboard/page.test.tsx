import { render, screen } from '@testing-library/react'
import { expect, it, vi } from 'vitest'
import DashboardPage from './page'

vi.mock('../hub-context', () => ({
  useHub: () => ({
    status: 'ready',
    error: '',
    reload: vi.fn(),
    data: {
      profile: { userId: 'player-1', username: 'player_one', displayName: 'Player One', country: 'ZA', language: 'en' },
      progression: {
        xp: 240, level: 2, prestige: 0, eloRating: 1200, leagueTier: 'Bronze',
        seasonPoints: 0, matchesPlayed: 1, wins: 1, losses: 0, currentStreak: 1,
        trustScore: 100, trustTier: 'trusted',
      },
      wallet: {
        currency: 'ZAR', availableBalance: 0, pendingDeposits: 0,
        pendingWithdrawals: 0, accountStatus: 'active', verificationStatus: 'unverified',
      },
      notifications: { unread: 1, total: 1 },
      objectives: [{
        id: 'practice', title: 'Complete Practice', description: 'Build verified skill evidence.',
        progress: 1, target: 1, complete: true, actionUrl: '/games',
      }],
      recommendedAction: {
        id: 'profile', label: 'Complete profile', description: 'Finish your identity.',
        actionUrl: '/profile', reason: 'A complete identity unlocks the next progression step.',
      },
      recentActivity: [],
      tournaments: [],
      challenges: [{ id: 'practice', type: 'practice', title: 'Practice', status: 'available', actionUrl: '/games' }],
      games: [{
        id: 'maze', name: 'Maze Arena', description: 'A deterministic puzzle discipline.',
        category: 'logic', version: '1.0.0', rendererKey: 'maze', modes: ['practice'],
        averageTimeSeconds: 180, capabilities: {
          practice: true, pvp: true, replay: true, tournament: true, spectator: true,
        },
        availability: 'available', rulesSummary: ['Select an arrow to submit an action.'],
      }],
      eligibility: {
        emailVerified: true, profileComplete: false, mfaEnabled: false,
        walletVisible: true, liveEligible: false, blockers: ['Complete your competitor profile.'],
      },
    },
  }),
}))

it('renders verified Hub state without invented competition data', () => {
  render(<DashboardPage />)

  expect(screen.getByRole('heading', { name: /player one/i })).toBeInTheDocument()
  expect(screen.getByRole('heading', { name: /complete profile/i })).toBeInTheDocument()
  expect(screen.getByText('Maze Arena')).toBeInTheDocument()
  expect(screen.getByText(/no tournament is currently accepting players/i)).toBeInTheDocument()
  expect(screen.getByText(/activity trail begins with your first practice session/i)).toBeInTheDocument()
  expect(screen.queryByText(/coming soon/i)).not.toBeInTheDocument()
})
