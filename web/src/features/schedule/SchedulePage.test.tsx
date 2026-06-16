import { render } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import SchedulePage from './SchedulePage'
import type { Match } from '../../lib/types'

function wrap(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) => <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

test('groups fixtures by date', () => {
  const qc = new QueryClient()
  qc.setQueryData<Match[]>(['fixtures'], [
    { id: 1, stage: 'group', group: 'A', venueId: 0, homeId: 1, awayId: 2, kickoffUtc: '2026-06-11T19:00:00Z', status: 'scheduled', minute: 0, homeScore: 0, awayScore: 0 },
    { id: 2, stage: 'group', group: 'A', venueId: 0, homeId: 3, awayId: 4, kickoffUtc: '2026-06-12T19:00:00Z', status: 'scheduled', minute: 0, homeScore: 0, awayScore: 0 },
  ])
  qc.setQueryData(['teams'], [])
  render(<SchedulePage />, { wrapper: wrap(qc) })
  // two distinct day sections rendered as <p> headers (not headings)
  const dayHeaders = document.querySelectorAll('.sched__day-header')
  expect(dayHeaders.length).toBeGreaterThanOrEqual(2)
})
