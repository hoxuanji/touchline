import { useQuery } from '@tanstack/react-query'
import { useNav } from '../../app/nav'
import { useTeams } from '../../lib/useTeams'
import { flag } from '../../lib/flags'
import { api } from '../../lib/api'
import { getFollows, follow, unfollow, getNotes, addNote } from '../../lib/dataApi'
import { useState } from 'react'
import type { Match } from '../../lib/types'

export default function TeamDetailPage({ teamId }: { teamId: number }) {
  const { back, go } = useNav()
  const teams = useTeams()
  const team = teams.get(teamId)
  const [noteText, setNoteText] = useState('')

  const { data: follows = [], refetch: refetchFollows } = useQuery({
    queryKey: ['follows'], queryFn: getFollows,
  })
  const { data: fixtures = [] } = useQuery({ queryKey: ['fixtures'], queryFn: api.fixtures })
  const { data: notes = [], refetch: refetchNotes } = useQuery({
    queryKey: ['notes', 'team', teamId], queryFn: () => getNotes('team', teamId),
  })

  const isFollowed = (follows as number[]).includes(teamId)
  const teamMatches = (fixtures as Match[]).filter(m => m.homeId === teamId || m.awayId === teamId)

  async function toggleFollow() {
    if (isFollowed) await unfollow(teamId)
    else await follow(teamId)
    refetchFollows()
  }

  async function submitNote() {
    if (!noteText.trim()) return
    await addNote('team', teamId, noteText.trim())
    setNoteText('')
    refetchNotes()
  }

  if (!team) return <div className="detail-loading">Loading…</div>

  return (
    <div className="detail">
      <button className="detail__back" onClick={back}>← Back</button>

      {/* Team hero */}
      <div className="detail__team-hero">
        <div className="detail__team-flag-xl">{flag(team.country)}</div>
        <div className="detail__team-info">
          <h2 className="detail__team-title">{team.name}</h2>
          <p className="detail__team-meta">Group {team.group} · {team.country}</p>
        </div>
        <button className={`follow-btn ${isFollowed ? 'is-following' : ''}`} onClick={toggleFollow}>
          {isFollowed ? '★ Following' : '☆ Follow'}
        </button>
      </div>

      {/* Fixtures */}
      <div className="detail__section">
        <p className="section-title">Fixtures &amp; Results</p>
        <div className="sched__rows">
          {teamMatches.slice(0, 6).map(m => {
            const isHome = m.homeId === teamId
            const opp = teams.get(isHome ? m.awayId : m.homeId)
            const isFin = m.status === 'finished'
            const isLive = m.status === 'live' || m.status === 'ht'
            return (
              <div key={m.id} className="sched__row" style={{ cursor: 'pointer' }} onClick={() => go({ page: 'match', id: m.id })}>
                <span className="sched__time">
                  {isLive ? `${m.minute}'` : isFin ? 'FT' : new Date(m.kickoffUtc).toLocaleDateString([], { month: 'short', day: 'numeric' })}
                </span>
                <span className="sched__team">
                  <span className="sched__flag">{flag(isHome ? team.country : opp?.country ?? '')}</span>
                  <span>{isHome ? team.name : opp?.name ?? '?'}</span>
                </span>
                <span className="sched__vs">
                  {isFin || isLive ? <strong>{m.homeScore} – {m.awayScore}</strong> : 'vs'}
                </span>
                <span className="sched__team sched__team--away">
                  <span>{isHome ? opp?.name ?? '?' : team.name}</span>
                  <span className="sched__flag">{flag(isHome ? opp?.country ?? '' : team.country)}</span>
                </span>
                <span className={`sched__status ${isLive ? 'is-live' : ''}`}>
                  {m.group ? `Grp ${m.group}` : m.stage?.toUpperCase()}
                </span>
              </div>
            )
          })}
        </div>
      </div>

      {/* Notes */}
      <div className="detail__section">
        <p className="section-title">Commentator Notes</p>
        <textarea className="notes__input" placeholder={`Notes on ${team.name}…`} value={noteText}
          onChange={e => setNoteText(e.target.value)} rows={3} />
        <button className="notes__submit" onClick={submitNote}>Save</button>
        <div className="notes__list">
          {(notes as { id: number; body: string; createdAt: string }[]).map(n => (
            <div key={n.id} className="notes__item">
              <p className="notes__body">{n.body}</p>
              <span className="notes__time">{new Date(n.createdAt).toLocaleString()}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
