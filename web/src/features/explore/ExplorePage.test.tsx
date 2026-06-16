import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import ExplorePage from './ExplorePage'
import type { Team, Venue } from '../../lib/types'

function wrap(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) => <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

test('lists teams and venues', () => {
  const qc = new QueryClient()
  qc.setQueryData<Team[]>(['teams'], [{ id: 1, name: 'Brazil', country: 'Brazil', group: 'F', crestUrl: '' }])
  qc.setQueryData<Venue[]>(['venues'], [{ id: 1, name: 'Estadio Azteca', city: 'Mexico City', country: 'Mexico', capacity: 87000 }])
  render(<ExplorePage />, { wrapper: wrap(qc) })
  expect(screen.getByText('Brazil')).toBeInTheDocument()
  expect(screen.getByText(/Estadio Azteca/)).toBeInTheDocument()
})
