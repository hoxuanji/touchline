import { renderHook, act, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useLiveStream, type EventSourceLike } from './useLiveStream'
import type { Match } from './types'

class MockES implements EventSourceLike {
  onopen: (() => void) | null = null
  onerror: (() => void) | null = null
  onmessage: ((ev: { data: string }) => void) | null = null
  close() {}
  emit(obj: unknown) { this.onmessage?.({ data: JSON.stringify(obj) }) }
  open() { this.onopen?.() }
}

function wrapper(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

test('patches fixtures cache on match.update', async () => {
  const qc = new QueryClient()
  qc.setQueryData<Match[]>(['fixtures'], [
    { id: 1, stage: 'group', group: 'A', venueId: 0, homeId: 1, awayId: 2, kickoffUtc: '', status: 'live', minute: 0, homeScore: 0, awayScore: 0 },
  ])
  let es!: MockES
  const { result } = renderHook(() => useLiveStream({ makeES: () => (es = new MockES()) }), { wrapper: wrapper(qc) })
  act(() => es.open())
  await waitFor(() => expect(result.current.connected).toBe(true))

  act(() => es.emit({ type: 'match.update', data: { id: 1, status: 'live', minute: 30, homeScore: 1, awayScore: 0, stage: 'group', group: 'A', venueId: 0, homeId: 1, awayId: 2, kickoffUtc: '' } }))

  const fixtures = qc.getQueryData<Match[]>(['fixtures'])!
  expect(fixtures.find(m => m.id === 1)!.minute).toBe(30)
  expect(fixtures.find(m => m.id === 1)!.homeScore).toBe(1)
})
