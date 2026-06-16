import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import StandingsPage from './StandingsPage'
import type { Standing } from '../../lib/types'

function wrap(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) => <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

test('renders a group table with points', () => {
  const qc = new QueryClient()
  qc.setQueryData<Standing[]>(['standings'], [
    { group: 'A', teamId: 1, played: 1, won: 1, drawn: 0, lost: 0, gf: 2, ga: 0, pts: 3, form: '' },
    { group: 'A', teamId: 2, played: 1, won: 0, drawn: 0, lost: 1, gf: 0, ga: 2, pts: 0, form: '' },
  ])
  render(<StandingsPage />, { wrapper: wrap(qc) })
  expect(screen.getByText('Group A')).toBeInTheDocument()
  expect(screen.getAllByRole('row').length).toBeGreaterThanOrEqual(3)
})
