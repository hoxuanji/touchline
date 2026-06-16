import { useQuery } from '@tanstack/react-query'
import { api } from '../../lib/api'
import { useTeams } from '../../lib/useTeams'
import { flag } from '../../lib/flags'
import { useNav } from '../../app/nav'
import type { Match } from '../../lib/types'

const LIVE = new Set(['live', 'ht'])

function fmt(iso: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function fmtDate(iso: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' })
}

export default function MatchCenterPage() {
  const { data: matches = [], isLoading } = useQuery({ queryKey: ['fixtures'], queryFn: api.fixtures })
  const teams = useTeams()
  const { go } = useNav()

  if (isLoading) return <p className="empty">Loading matches…</p>

  const live = matches.filter((m) => LIVE.has(m.status))
  const upcoming = matches.filter((m) => m.status === 'scheduled').slice(0, 16)

  return (
    <div className="mc">
      {/* ── LIVE ── */}
      <section>
        <div className="mc__section-header">
          <h2>Live</h2>
          {live.length > 0 && <span className="mc__live-badge">Live</span>}
        </div>
        {live.length === 0
          ? <p className="empty">No matches live right now.</p>
          : (
            <div className="mc__grid">
              {live.map((m) => <ScoreCard key={m.id} m={m} teams={teams} onClick={() => go({ page: 'match', id: m.id })} />)}
            </div>
          )
        }
      </section>

      {/* ── UPCOMING ── */}
      <section>
        <div className="mc__section-header"><h2>Upcoming</h2></div>
        <div className="mc__upcoming">
          {upcoming.map((m) => {
            const home = teams.get(m.homeId)
            const away = teams.get(m.awayId)
            return (
              <div key={m.id} className="mc__upcoming-row" style={{ cursor: 'pointer' }} onClick={() => go({ page: 'match', id: m.id })}>
                <span className="mc__upcoming-time">{fmt(m.kickoffUtc)}</span>
                <span className="mc__upcoming-team">
                  <span className="mc__upcoming-flag">{flag(home?.country ?? '')}</span>
                  <span>{home?.name ?? `#${m.homeId}`}</span>
                </span>
                <span className="mc__upcoming-vs">VS</span>
                <span className="mc__upcoming-team mc__upcoming-team--away">
                  <span>{away?.name ?? `#${m.awayId}`}</span>
                  <span className="mc__upcoming-flag">{flag(away?.country ?? '')}</span>
                </span>
                <span className="mc__upcoming-group">{m.group || m.stage?.toUpperCase()}</span>
              </div>
            )
          })}
        </div>
      </section>
    </div>
  )
}

function ScoreCard({ m, teams, onClick }: { m: Match; teams: Map<number, { name: string; country: string }>; onClick: () => void }) {
  const home = teams.get(m.homeId)
  const away = teams.get(m.awayId)
  const cls = m.status === 'finished' ? 'mc__card is-finished' : m.status === 'live' || m.status === 'ht' ? 'mc__card is-live' : 'mc__card'
  return (
    <div className={cls} style={{ cursor: 'pointer' }} onClick={onClick}>
      <div className="mc__card-info">
        <span>{m.group ? `Group ${m.group}` : m.stage?.toUpperCase()}</span>
        {(m.status === 'live') && <span className="mc__card-minute">{m.minute}&apos;</span>}
        {m.status === 'ht' && <span className="mc__card-minute">HT</span>}
        {m.status === 'finished' && <span>FT</span>}
        {m.status === 'scheduled' && <span>{fmt(m.kickoffUtc)}</span>}
      </div>
      <div className="mc__team mc__team--home">
        <span className="mc__flag">{flag(home?.country ?? '')}</span>
        <span className="mc__team-name">{home?.name ?? `#${m.homeId}`}</span>
      </div>
      <div className="mc__team mc__team--away">
        <span className="mc__flag">{flag(away?.country ?? '')}</span>
        <span className="mc__team-name">{away?.name ?? `#${m.awayId}`}</span>
      </div>
      <div className="mc__score-row">
        <span className="mc__score-num">{m.homeScore}</span>
        <span className="mc__score-sep">–</span>
        <span className="mc__score-num">{m.awayScore}</span>
      </div>
      <div className="mc__card-footer">
        <span className="muted">{fmtDate(m.kickoffUtc)}</span>
      </div>
    </div>
  )
}
