import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import WalletPage from './page'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('../lib/api', async () => {
  const actual = await vi.importActual<typeof import('../lib/api')>('../lib/api')
  return { ...actual, apiFetch }
})

const overview = {
  wallet: {
    currency: 'ZAR', availableMinor: 12_345, pendingDepositMinor: 0, pendingWithdrawalMinor: 2_000,
    lockedMinor: 0, lifetimeDepositMinor: 20_000, lifetimeWithdrawalMinor: 5_000,
    updatedAt: '2026-07-23T12:00:00Z',
  },
  assessment: {
    status: 'complete', country: 'ZA', occupation: 'employed', sourceOfFunds: 'salary',
    riskClassification: 'standard', verificationStatus: 'verified', responsibleGamingStatus: 'active',
  },
  limits: {
    currency: 'ZAR', dailyDepositMinor: 50_000, monthlyDepositMinor: 500_000,
    dailyWithdrawalMinor: 25_000, monthlyWithdrawalMinor: 250_000,
    depositUsedTodayMinor: 0, depositUsedMonthMinor: 20_000,
    withdrawalUsedTodayMinor: 2_000, withdrawalUsedMonthMinor: 5_000,
  },
  verificationStatus: 'approved',
  paymentMethods: [{ id: 'card', type: 'card', displayName: 'Card', currency: 'ZAR', available: true }],
  deposits: [],
  withdrawals: [{
    id: 'withdrawal-1', amountMinor: 2_000, feeMinor: 0, currency: 'ZAR', method: 'card',
    status: 'pending_review',
    requestedAt: '2026-07-23T12:00:00Z', updatedAt: '2026-07-23T12:00:00Z',
  }],
}

describe('Financial wallet', () => {
  afterEach(cleanup)

  beforeEach(() => {
    apiFetch.mockReset()
    apiFetch.mockImplementation((path: string) => {
      if (path === '/api/v1/financial/overview') return Promise.resolve(overview)
      if (path === '/api/v1/financial/transactions') return Promise.resolve([])
      if (path === '/api/v1/financial/evidence') return Promise.resolve([])
      return Promise.resolve({})
    })
  })

  it('renders backend balances, lifecycle state, and assessment status', async () => {
    render(<WalletPage />)
    expect(await screen.findByText('Your money, with every stage visible.')).toBeInTheDocument()
    const formattedBalance = new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency: 'ZAR',
    }).format(123.45)
    expect(screen.getByText((_, element) => (
      element?.tagName === 'STRONG' && element.textContent === formattedBalance
    ))).toBeInTheDocument()
    expect(screen.getByText('Pending Review')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Assessment' }))
    expect(screen.getByRole('heading', { name: 'Complete' })).toBeInTheDocument()
  })

  it('submits integer minor units with an idempotency key', async () => {
    render(<WalletPage />)
    await screen.findByText('Your money, with every stage visible.')
    await userEvent.click(screen.getByRole('button', { name: 'Withdraw' }))
    await userEvent.type(screen.getByLabelText('Amount'), '20.50')
    await userEvent.click(screen.getByRole('button', { name: 'Submit withdrawal' }))
    await waitFor(() => expect(apiFetch).toHaveBeenCalledWith(
      '/api/v1/financial/withdrawals',
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({ 'Idempotency-Key': expect.stringMatching(/^wallet-/) }),
        body: JSON.stringify({ amountMinor: 2050, currency: 'ZAR', method: 'card' }),
      }),
    ))
  })
})
