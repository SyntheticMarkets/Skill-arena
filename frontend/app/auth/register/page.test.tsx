import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'
import RegisterPage from './page'

const { push, postJSON } = vi.hoisted(() => ({ push: vi.fn(), postJSON: vi.fn() }))
vi.mock('next/navigation', () => ({ useRouter: () => ({ push }) }))
vi.mock('../../lib/api', async () => {
  const actual = await vi.importActual<typeof import('../../lib/api')>('../../lib/api')
  return { ...actual, postJSON }
})

beforeEach(() => {
  push.mockReset()
  postJSON.mockReset().mockResolvedValue({ status: 'verification_required' })
})

it('requires identity, age, password, and consent before registration', async () => {
  render(<RegisterPage />)
  const submit = screen.getByRole('button', { name: /create identity/i })
  expect(submit).toBeDisabled()
  fireEvent.change(screen.getByLabelText(/email address/i), { target: { value: 'player@example.com' } })
  fireEvent.change(screen.getByLabelText(/date of birth/i), { target: { value: '1990-01-01' } })
  fireEvent.change(screen.getByLabelText(/^password$/i), { target: { value: 'StrongPassword!42' } })
  const checks = screen.getAllByRole('checkbox')
  fireEvent.click(checks[0])
  fireEvent.click(checks[1])
  expect(submit).toBeEnabled()
  fireEvent.click(submit)
  await waitFor(() => expect(postJSON).toHaveBeenCalledWith('/api/v1/auth/register', expect.objectContaining({ email: 'player@example.com', acceptTerms: true, acceptFairPlay: true })))
  expect(push).toHaveBeenCalledWith('/auth/verification-pending?email=player%40example.com')
})
