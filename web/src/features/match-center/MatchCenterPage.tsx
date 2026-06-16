import { useQuery } from '@tanstack/react-query'
import { api } from '../../lib/api'
import type { Match } from '../../lib/types'

const LIVE = new Set(['live', 'ht'])

export default function MatchCenterPage() {
  const { data: matches = [], isLoading } = useQuery({ queryKey: ['fixtures'], queryFn: api.fixtures })
  if (isLoading) return <p className="muted">Loading matches…</p>
  const live = matches.filter((m) => LIVE.has(m.status))
  const upcoming = matches.filter((m) => m.status === 'scheduled').slice(0, 12)
  return (
    <div className="mc">
      <h2 className="mc__h">Live</h2>
      {live.length === 0 && <p className="muted">No live matches right now.</p>}
      <ul className="mc__list">
        {live.map((m) => <MatchRow key={m.id} m={m} live />)}
      </ul>
      <h2 className="mc__h">Upcoming</h2>
      <ul className="mc__list">
        {upcoming.map((m) => <MatchRow key={m.id} m={m} />)}
      </ul>
    </div>
  )
}

function MatchRow({ m, live }: { m: Match; live?: boolean }) {
  return (
    <li className={`mc__row ${live ? 'is-live' : ''}`}>
      <span className="mc__teams">#{m.homeId} v #{m.awayId}</span>
      <span className="mc__score">{m.homeScore} – {m.awayScore}</span>
      {live ? <span className="mc__min">{m.minute}&apos;</span> : <span className="mc__time">{new Date(m.kickoffUtc).toLocaleString()}</span>}
    </li>
  )
}
