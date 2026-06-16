import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../../lib/api'
import { flag } from '../../lib/flags'
import { useNav } from '../../app/nav'
import type { Team, Venue } from '../../lib/types'

const VENUE_COUNTRY_FLAG: Record<string, string> = {
  USA: '🇺🇸', Canada: '🇨🇦', Mexico: '🇲🇽',
}

export default function ExplorePage() {
  const { data: teams = [] } = useQuery<Team[]>({ queryKey: ['teams'], queryFn: api.teams })
  const { data: venues = [] } = useQuery<Venue[]>({ queryKey: ['venues'], queryFn: api.venues })
  const [selectedGroup, setSelectedGroup] = useState<string>('All')
  const { go } = useNav()

  const groups = ['All', ...['A','B','C','D','E','F','G','H','I','J','K','L']]
  const filtered = selectedGroup === 'All' ? teams : teams.filter((t) => t.group === selectedGroup)

  return (
    <div className="explore">
      {/* ── Teams ── */}
      <section>
        <p className="section-title">National Teams</p>
        {/* Group filter */}
        <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap', marginBottom: '1rem' }}>
          {groups.map((g) => (
            <button
              key={g}
              onClick={() => setSelectedGroup(g)}
              style={{
                background: selectedGroup === g ? 'rgba(0,210,106,0.15)' : 'var(--panel)',
                color: selectedGroup === g ? 'var(--text)' : 'var(--muted)',
                border: `1px solid ${selectedGroup === g ? 'rgba(0,210,106,0.4)' : 'var(--line)'}`,
                borderRadius: '5px',
                padding: '0.25rem 0.65rem',
                fontSize: '0.72rem',
                cursor: 'pointer',
                letterSpacing: '0.06em',
                fontFamily: 'inherit',
              }}
            >
              {g === 'All' ? 'All' : `Grp ${g}`}
            </button>
          ))}
        </div>
        <div className="explore__cards">
          {filtered.map((t: Team) => (
            <div key={t.id} className="explore__team-card" role="button" tabIndex={0}
              onClick={() => go({ page: 'team', id: t.id })}
              onKeyDown={e => e.key === 'Enter' && go({ page: 'team', id: t.id })}>
              <span className="explore__team-flag">{flag(t.country)}</span>
              <span className="explore__team-name">{t.name}</span>
              <span className="explore__team-group">Group {t.group}</span>
            </div>
          ))}
        </div>
      </section>

      {/* ── Venues ── */}
      <section>
        <p className="section-title">Venues &amp; Stadiums</p>
        <div className="explore__venue-cards">
          {venues.map((v: Venue) => (
            <div key={v.id} className="explore__venue-card" role="button" tabIndex={0}
              onClick={() => go({ page: 'venue', id: v.id })}
              onKeyDown={e => e.key === 'Enter' && go({ page: 'venue', id: v.id })}>
              <div className="explore__venue-name">{v.name}</div>
              <div className="explore__venue-meta">
                <span className="explore__venue-country">{VENUE_COUNTRY_FLAG[v.country] ?? '🏟️'}</span>
                {v.city}, {v.country}
              </div>
              <div className="explore__venue-cap">{v.capacity.toLocaleString()} seats</div>
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}
