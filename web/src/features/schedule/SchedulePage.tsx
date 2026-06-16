import { useQuery } from '@tanstack/react-query'
import { api } from '../../lib/api'
import { useTeams } from '../../lib/useTeams'
import { flag } from '../../lib/flags'
import type { Match } from '../../lib/types'

function dayKey(iso: string) {
  if (!iso) return 'TBD'
  const d = new Date(iso)
  return d.toLocaleDateString([], { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })
}

function timeStr(iso: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

export default function SchedulePage() {
  const { data: matches = [] } = useQuery({ queryKey: ['fixtures'], queryFn: api.fixtures })
  const teams = useTeams()

  const byDay = new Map<string, Match[]>()
  for (const m of [...matches].sort((a, b) => a.kickoffUtc.localeCompare(b.kickoffUtc))) {
    const k = dayKey(m.kickoffUtc)
    if (!byDay.has(k)) byDay.set(k, [])
    byDay.get(k)!.push(m)
  }

  return (
    <div className="sched">
      {[...byDay.entries()].map(([day, dayMatches]) => (
        <section key={day}>
          <p className="sched__day-header">{day}</p>
          <div className="sched__rows">
            {dayMatches.map((m) => {
              const home = teams.get(m.homeId)
              const away = teams.get(m.awayId)
              const isLive = m.status === 'live' || m.status === 'ht'
              const isFin = m.status === 'finished'
              return (
                <div key={m.id} className="sched__row">
                  <span className="sched__time">
                    {isLive ? (m.status === 'ht' ? 'HT' : `${m.minute}'`) : timeStr(m.kickoffUtc)}
                  </span>
                  <span className="sched__team">
                    <span className="sched__flag">{flag(home?.country ?? '')}</span>
                    <span>{home?.name ?? `#${m.homeId}`}</span>
                  </span>
                  <span className="sched__vs">
                    {isFin || isLive
                      ? <strong style={{ fontSize: '0.95rem' }}>{m.homeScore} – {m.awayScore}</strong>
                      : 'vs'}
                  </span>
                  <span className="sched__team sched__team--away">
                    <span>{away?.name ?? `#${m.awayId}`}</span>
                    <span className="sched__flag">{flag(away?.country ?? '')}</span>
                  </span>
                  <span className={`sched__status ${isLive ? 'is-live' : ''} ${isFin ? 'is-fin' : ''}`}>
                    {isLive ? 'LIVE' : isFin ? 'FT' : m.group ? `Grp ${m.group}` : m.stage?.toUpperCase()}
                  </span>
                </div>
              )
            })}
          </div>
        </section>
      ))}
    </div>
  )
}
