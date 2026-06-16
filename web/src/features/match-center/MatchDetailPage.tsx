import { useQuery } from '@tanstack/react-query'
import { useNav } from '../../app/nav'
import { useTeams } from '../../lib/useTeams'
import { flag } from '../../lib/flags'
import { getMatch, addNote, getNotes } from '../../lib/dataApi'
import { useState } from 'react'
import type { MatchEvent } from '../../lib/types'

const EVENT_ICON: Record<string, string> = {
  goal: '⚽', card: '🟨', sub: '🔄', var: '📺',
}

function statusLabel(status: string, minute: number) {
  if (status === 'live') return `${minute}'`
  if (status === 'ht') return 'HT'
  if (status === 'finished') return 'FT'
  return 'Upcoming'
}

export default function MatchDetailPage({ matchId }: { matchId: number }) {
  const { back } = useNav()
  const teams = useTeams()
  const [noteText, setNoteText] = useState('')
  const [tab, setTab] = useState<'timeline' | 'stats' | 'notes'>('timeline')

  const { data, isLoading } = useQuery({
    queryKey: ['match', matchId],
    queryFn: () => getMatch(matchId),
    refetchInterval: 15_000,
  })

  const { data: notes = [], refetch: refetchNotes } = useQuery({
    queryKey: ['notes', 'match', matchId],
    queryFn: () => getNotes('match', matchId),
  })

  if (isLoading) return <div className="detail-loading">Loading match…</div>
  if (!data) return null

  const { match: m, events } = data
  const home = teams.get(m.homeId)
  const away = teams.get(m.awayId)
  const isLive = m.status === 'live' || m.status === 'ht'

  async function submitNote() {
    if (!noteText.trim()) return
    await addNote('match', matchId, noteText.trim())
    setNoteText('')
    refetchNotes()
  }

  const homeEvents = events.filter(e => e.teamId === m.homeId)
  const awayEvents = events.filter(e => e.teamId === m.awayId)
  const homeGoals = homeEvents.filter(e => e.type === 'goal').length
  const awayGoals = awayEvents.filter(e => e.type === 'goal').length
  const homeCards = homeEvents.filter(e => e.type === 'card').length
  const awayCards = awayEvents.filter(e => e.type === 'card').length

  return (
    <div className="detail">
      {/* Back */}
      <button className="detail__back" onClick={back}>← Back</button>

      {/* Scoreboard hero */}
      <div className={`detail__hero ${isLive ? 'is-live' : ''}`}>
        <div className="detail__stage">
          {m.group ? `Group ${m.group}` : m.stage?.toUpperCase()}
          {isLive && <span className="mc__live-badge" style={{ marginLeft: 8 }}>LIVE</span>}
        </div>
        <div className="detail__scoreboard">
          <div className="detail__team-col">
            <span className="detail__hero-flag">{flag(home?.country ?? '')}</span>
            <span className="detail__hero-name">{home?.name ?? `#${m.homeId}`}</span>
          </div>
          <div className="detail__score-col">
            <div className="detail__score-big">{m.homeScore} – {m.awayScore}</div>
            <div className="detail__score-status">{statusLabel(m.status, m.minute)}</div>
          </div>
          <div className="detail__team-col detail__team-col--right">
            <span className="detail__hero-flag">{flag(away?.country ?? '')}</span>
            <span className="detail__hero-name">{away?.name ?? `#${m.awayId}`}</span>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="detail__tabs">
        {(['timeline', 'stats', 'notes'] as const).map(t => (
          <button key={t} className={`detail__tab ${tab === t ? 'is-active' : ''}`} onClick={() => setTab(t)}>
            {t.charAt(0).toUpperCase() + t.slice(1)}
          </button>
        ))}
      </div>

      {/* Timeline */}
      {tab === 'timeline' && (
        <div className="detail__timeline">
          {events.length === 0
            ? <p className="empty">No events yet — check back once the match kicks off.</p>
            : events.map((e: MatchEvent) => {
              const isHome = e.teamId === m.homeId
              return (
                <div key={e.id} className={`timeline__row ${isHome ? 'is-home' : 'is-away'}`}>
                  {!isHome && <span className="timeline__spacer" />}
                  <div className="timeline__event">
                    <span className="timeline__icon">{EVENT_ICON[e.type] ?? '•'}</span>
                    <span className="timeline__detail">{e.detail || e.type}</span>
                  </div>
                  <span className="timeline__min">{e.minute}'</span>
                  {isHome && <span className="timeline__spacer" />}
                </div>
              )
            })
          }
        </div>
      )}

      {/* Stats */}
      {tab === 'stats' && (
        <div className="detail__stats">
          <StatBar label="Goals" home={homeGoals} away={awayGoals} />
          <StatBar label="Yellow Cards" home={homeCards} away={awayCards} />
        </div>
      )}

      {/* Notes */}
      {tab === 'notes' && (
        <div className="detail__notes">
          <textarea
            className="notes__input"
            placeholder="Add a match note…"
            value={noteText}
            onChange={e => setNoteText(e.target.value)}
            rows={3}
          />
          <button className="notes__submit" onClick={submitNote}>Save Note</button>
          <div className="notes__list">
            {(notes as { id: number; body: string; createdAt: string }[]).map(n => (
              <div key={n.id} className="notes__item">
                <p className="notes__body">{n.body}</p>
                <span className="notes__time">{new Date(n.createdAt).toLocaleString()}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function StatBar({ label, home, away }: { label: string; home: number; away: number }) {
  const total = home + away || 1
  const homePct = Math.round((home / total) * 100)
  const awayPct = 100 - homePct
  return (
    <div className="stat-bar">
      <div className="stat-bar__nums">
        <span className="stat-bar__val">{home}</span>
        <span className="stat-bar__label">{label}</span>
        <span className="stat-bar__val">{away}</span>
      </div>
      <div className="stat-bar__track">
        <div className="stat-bar__fill stat-bar__fill--home" style={{ width: `${homePct}%` }} />
        <div className="stat-bar__fill stat-bar__fill--away" style={{ width: `${awayPct}%` }} />
      </div>
    </div>
  )
}
