import { useEffect, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import type { Match } from './types'

export interface EventSourceLike {
  onopen: (() => void) | null
  onerror: (() => void) | null
  onmessage: ((ev: { data: string }) => void) | null
  close(): void
}

type Msg = { type: string; data: unknown }

export function useLiveStream(opts?: { makeES?: (url: string) => EventSourceLike }): { connected: boolean } {
  const qc = useQueryClient()
  const [connected, setConnected] = useState(false)
  const retryRef = useRef(0)

  useEffect(() => {
    let stopped = false
    let es: EventSourceLike | null = null

    const connect = () => {
      if (stopped) return
      // No real EventSource in jsdom/tests; only connect if a factory is given or the browser supports it.
      if (!opts?.makeES && typeof EventSource === 'undefined') return
      const make = opts?.makeES ?? ((url: string) => new EventSource(url) as unknown as EventSourceLike)
      es = make('/api/stream')
      es.onopen = () => { setConnected(true); retryRef.current = 0 }
      es.onerror = () => {
        setConnected(false)
        es?.close()
        const delay = Math.min(1000 * 2 ** retryRef.current++, 10_000)
        if (!stopped) setTimeout(connect, delay)
      }
      es.onmessage = (ev) => {
        let msg: Msg
        try { msg = JSON.parse(ev.data) } catch { return }
        if (msg.type === 'match.update') {
          const m = msg.data as Match
          qc.setQueryData<Match[]>(['fixtures'], (old) =>
            old ? old.map((x) => (x.id === m.id ? m : x)) : old)
          qc.setQueryData<Match>(['match', m.id], m)
        }
        // match.event is consumed by the Match Center slice (appends to ['events', id])
      }
    }
    connect()
    return () => { stopped = true; es?.close() }
  }, [qc, opts])

  return { connected }
}
