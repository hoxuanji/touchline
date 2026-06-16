import { useState } from 'react'
import type { ComponentType } from 'react'
import { useLiveStream } from '../lib/useLiveStream'
import { useMeta } from '../features/meta/useMeta'
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
  const { data: meta } = useMeta()
  const Page = PAGES[view]
  return (
    <main className="shell">
      <header className="shell__bar">
        <span className="shell__logo">
          <span className="shell__mark">▲</span>
          <h1 className="shell__title">TOUCHLINE</h1>
        </span>
        <span className="shell__tag">FIFA WORLD CUP 2026</span>
        <nav className="shell__nav">
          {VIEWS.map((v) => (
            <button key={v} className={v === view ? 'is-active' : ''} onClick={() => setView(v)}>{v}</button>
          ))}
        </nav>
        <span className="shell__status">
          <span className={`shell__live ${connected ? 'is-on' : ''}`}>●</span>
          {meta?.source && <span className="shell__source">{meta.source}</span>}
        </span>
      </header>
      <div className="shell__body">
        <Page />
      </div>
    </main>
  )
}
