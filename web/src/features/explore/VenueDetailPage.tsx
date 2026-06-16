import { useQuery } from '@tanstack/react-query'
import { useNav } from '../../app/nav'
import { useTeams } from '../../lib/useTeams'
import { flag } from '../../lib/flags'
import { api } from '../../lib/api'
import type { Match, Venue } from '../../lib/types'

const VENUE_FLAG: Record<string, string> = { USA: '🇺🇸', Canada: '🇨🇦', Mexico: '🇲🇽' }

export default function VenueDetailPage({ venueId }: { venueId: number }) {
  const { back, go } = useNav()
  const teams = useTeams()
  const { data: venues = [] } = useQuery({ queryKey: ['venues'], queryFn: api.venues })
  const { data: fixtures = [] } = useQuery({ queryKey: ['fixtures'], queryFn: api.fixtures })

  const venue = (venues as Venue[]).find(v => v.id === venueId)
  const venueMatches = (fixtures as Match[]).filter(m => m.venueId === venueId)

  if (!venue) return <div className="detail-loading">Loading…</div>

  return (
    <div className="detail">
      <button className="detail__back" onClick={back}>← Back</button>

      <div className="detail__venue-hero">
        <div className="detail__venue-icon">🏟️</div>
        <div>
          <h2 className="detail__team-title">{venue.name}</h2>
          <p className="detail__team-meta">
            {VENUE_FLAG[venue.country] ?? ''} {venue.city}, {venue.country}
          </p>
          <p className="detail__venue-cap">{venue.capacity.toLocaleString()} seats</p>
        </div>
      </div>

      <div className="detail__section">
        <p className="section-title">Matches Hosted</p>
        <div className="sched__rows">
          {venueMatches.length === 0
            ? <p className="empty">No matches at this venue yet.</p>
            : venueMatches.map(m => {
              const home = teams.get(m.homeId)
              const away = teams.get(m.awayId)
              const isFin = m.status === 'finished'
              const isLive = m.status === 'live' || m.status === 'ht'
              return (
                <div key={m.id} className="sched__row" style={{ cursor: 'pointer' }} onClick={() => go({ page: 'match', id: m.id })}>
                  <span className="sched__time">
                    {isLive ? `${m.minute}'` : isFin ? 'FT' : new Date(m.kickoffUtc).toLocaleDateString([], { month: 'short', day: 'numeric' })}
                  </span>
                  <span className="sched__team">
                    <span className="sched__flag">{flag(home?.country ?? '')}</span>
                    <span>{home?.name ?? `#${m.homeId}`}</span>
                  </span>
                  <span className="sched__vs">
                    {isFin || isLive ? <strong>{m.homeScore} – {m.awayScore}</strong> : 'vs'}
                  </span>
                  <span className="sched__team sched__team--away">
                    <span>{away?.name ?? `#${m.awayId}`}</span>
                    <span className="sched__flag">{flag(away?.country ?? '')}</span>
                  </span>
                  <span className={`sched__status ${isLive ? 'is-live' : ''}`}>{m.group ? `Grp ${m.group}` : m.stage?.toUpperCase()}</span>
                </div>
              )
            })
          }
        </div>
      </div>
    </div>
  )
}
