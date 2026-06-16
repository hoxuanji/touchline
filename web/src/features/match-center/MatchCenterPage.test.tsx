import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import MatchCenterPage from './MatchCenterPage'
import type { Match } from '../../lib/types'

function wrap(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) => <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

test('lists live matches with score', () => {
  const qc = new QueryClient()
  qc.setQueryData(['teams'], [
    { id: 1, name: 'Brazil', country: 'Brazil', group: 'F', crestUrl: '' },
    { id: 2, name: 'Spain', country: 'Spain', group: 'F', crestUrl: '' },
  ])
  qc.setQueryData<Match[]>(['fixtures'], [
    { id: 1, stage: 'group', group: 'F', venueId: 0, homeId: 1, awayId: 2, kickoffUtc: '2026-06-11T19:00:00Z', status: 'live', minute: 30, homeScore: 2, awayScore: 1 },
  ])
  render(<MatchCenterPage />, { wrapper: wrap(qc) })
  // scores are in separate elements (2 and 1 in separate spans)
  expect(screen.getByText('2')).toBeInTheDocument()
  expect(screen.getByText('1')).toBeInTheDocument()
  expect(screen.getByText("30'")).toBeInTheDocument()
  expect(screen.getByText('Brazil')).toBeInTheDocument()
})
