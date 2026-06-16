import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../../lib/api'
import { useTeams } from '../../lib/useTeams'
import { flag } from '../../lib/flags'
import type { Match } from '../../lib/types'

function isoDay(iso: string) {
  if (!iso) return ''
  return iso.slice(0, 10) // "2026-06-11"
}

function timeStr(iso: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function monthLabel(iso: string) {
  return new Date(iso).toLocaleDateString([], { month: 'long', year: 'numeric' })
}

// All days in a month that contains WC2026 fixtures (June–July 2026)
function calDays(year: number, month: number): (string | null)[] {
  const first = new Date(year, month, 1).getDay() // 0=Sun
  const total = new Date(year, month + 1, 0).getDate()
  const cells: (string | null)[] = Array(first).fill(null)
  for (let d = 1; d <= total; d++) {
    cells.push(`${year}-${String(month + 1).padStart(2, '0')}-${String(d).padStart(2, '0')}`)
  }
  return cells
}

export default function SchedulePage() {
  const { data: matches = [] } = useQuery({ queryKey: ['fixtures'], queryFn: api.fixtures })
  const teams = useTeams()
  const [mode, setMode] = useState<'list' | 'cal'>('list')
  const [calMonth, setCalMonth] = useState<[number, number]>([2026, 5]) // June = month index 5

  // Sort and group by day
  const sorted = [...matches].sort((a, b) => a.kickoffUtc.localeCompare(b.kickoffUtc))
  const byDay = new Map<string, Match[]>()
  for (const m of sorted) {
    const k = isoDay(m.kickoffUtc)
    if (!k) continue
    if (!byDay.has(k)) byDay.set(k, [])
    byDay.get(k)!.push(m)
  }

  return (
    <div className="sched">
      {/* ── View toggle ── */}
      <div className="sched__toolbar">
        <button className={mode === 'list' ? 'sched__view-btn is-active' : 'sched__view-btn'} onClick={() => setMode('list')}>
          ≡ List
        </button>
        <button className={mode === 'cal' ? 'sched__view-btn is-active' : 'sched__view-btn'} onClick={() => setMode('cal')}>
          ◫ Calendar
        </button>
      </div>

      {mode === 'list' ? (
        <ListView byDay={byDay} teams={teams} />
      ) : (
        <CalView byDay={byDay} calMonth={calMonth} setCalMonth={setCalMonth} teams={teams} />
      )}
    </div>
  )
}

function ListView({ byDay, teams }: { byDay: Map<string, Match[]>; teams: ReturnType<typeof useTeams> }) {
  return (
    <>
      {[...byDay.entries()].map(([day, dayMatches]) => {
        const label = new Date(day + 'T12:00:00Z').toLocaleDateString([], { weekday: 'long', month: 'long', day: 'numeric' })
        return (
          <section key={day}>
            <p className="sched__day-header">{label}</p>
            <div className="sched__rows">
              {dayMatches.map((m) => <MatchRow key={m.id} m={m} teams={teams} />)}
            </div>
          </section>
        )
      })}
    </>
  )
}

function CalView({
  byDay, calMonth, setCalMonth, teams,
}: {
  byDay: Map<string, Match[]>
  calMonth: [number, number]
  setCalMonth: (v: [number, number]) => void
  teams: ReturnType<typeof useTeams>
}) {
  const [year, month] = calMonth
  const cells = calDays(year, month)
  const [selected, setSelected] = useState<string | null>(null)

  function prevMonth() {
    setCalMonth(month === 0 ? [year - 1, 11] : [year, month - 1])
    setSelected(null)
  }
  function nextMonth() {
    setCalMonth(month === 11 ? [year + 1, 0] : [year, month + 1])
    setSelected(null)
  }

  const selectedMatches = selected ? (byDay.get(selected) ?? []) : []

  return (
    <div className="cal">
      <div className="cal__header">
        <button className="cal__nav" onClick={prevMonth}>‹</button>
        <span className="cal__month-label">{monthLabel(`${year}-${String(month + 1).padStart(2, '0')}-01`)}</span>
        <button className="cal__nav" onClick={nextMonth}>›</button>
      </div>
      <div className="cal__grid">
        {['Sun','Mon','Tue','Wed','Thu','Fri','Sat'].map(d => (
          <div key={d} className="cal__weekday">{d}</div>
        ))}
        {cells.map((day, i) => {
          if (!day) return <div key={`e-${i}`} className="cal__cell cal__cell--empty" />
          const count = byDay.get(day)?.length ?? 0
          const isToday = day === new Date().toISOString().slice(0, 10)
          const isSelected = day === selected
          return (
            <div
              key={day}
              className={`cal__cell ${count > 0 ? 'has-matches' : ''} ${isToday ? 'is-today' : ''} ${isSelected ? 'is-selected' : ''}`}
              onClick={() => count > 0 && setSelected(isSelected ? null : day)}
            >
              <span className="cal__day-num">{parseInt(day.slice(8))}</span>
              {count > 0 && <span className="cal__dot-row">{Array(Math.min(count, 4)).fill(null).map((_, k) => <span key={k} className="cal__dot" />)}</span>}
            </div>
          )
        })}
      </div>
      {selected && selectedMatches.length > 0 && (
        <div className="cal__detail">
          <p className="sched__day-header">
            {new Date(selected + 'T12:00:00Z').toLocaleDateString([], { weekday: 'long', month: 'long', day: 'numeric' })}
          </p>
          <div className="sched__rows">
            {selectedMatches.map((m) => <MatchRow key={m.id} m={m} teams={teams} />)}
          </div>
        </div>
      )}
    </div>
  )
}

function MatchRow({ m, teams }: { m: Match; teams: ReturnType<typeof useTeams> }) {
  const home = teams.get(m.homeId)
  const away = teams.get(m.awayId)
  const isLive = m.status === 'live' || m.status === 'ht'
  const isFin = m.status === 'finished'
  return (
    <div className="sched__row">
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
}
