import { useState } from 'react'
import type { ComponentType } from 'react'
import { useLiveStream } from '../lib/useLiveStream'
import MatchCenterPage from '../features/match-center/MatchCenterPage'
import SchedulePage from '../features/schedule/SchedulePage'
import StandingsPage from '../features/standings/StandingsPage'
import ExplorePage from '../features/explore/ExplorePage'

const VIEWS = ['Match Center', 'Schedule', 'Standings', 'Explore'] as const
type View = (typeof VIEWS)[number]

const PAGES: Record<View, ComponentType> = {
  'Match Center': MatchCenterPage,
  Schedule: SchedulePage,
  Standings: StandingsPage,
  Explore: ExplorePage,
}

export default function Shell() {
  const [view, setView] = useState<View>('Match Center')
  const { connected } = useLiveStream()
  const Page = PAGES[view]
  return (
    <main className="shell">
      <header className="shell__bar">
        <span className="shell__mark" aria-hidden="true">▲</span>
        <h1 className="shell__title">TOUCHLINE</h1>
        <span className="shell__tag">WORLD CUP 2026</span>
        <nav className="shell__nav">
          {VIEWS.map((v) => (
            <button key={v} className={v === view ? 'is-active' : ''} onClick={() => setView(v)}>{v}</button>
          ))}
        </nav>
        <span className={`shell__live ${connected ? 'is-on' : ''}`} title={connected ? 'live' : 'offline'}>●</span>
      </header>
      <section className="shell__body shell__body--page">
        <Page />
      </section>
    </main>
  )
}
