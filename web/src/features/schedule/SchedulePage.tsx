import { useQuery } from '@tanstack/react-query'
import { api } from '../../lib/api'
import type { Match } from '../../lib/types'
import { getFollows } from './api'

function dayKey(iso: string): string {
  if (!iso) return 'TBD'
  return new Date(iso).toISOString().slice(0, 10)
}

export default function SchedulePage() {
  const { data: matches = [] } = useQuery({ queryKey: ['fixtures'], queryFn: api.fixtures })
  useQuery({ queryKey: ['follows'], queryFn: getFollows })
  const byDay = new Map<string, Match[]>()
  for (const m of matches) {
    const k = dayKey(m.kickoffUtc)
    if (!byDay.has(k)) byDay.set(k, [])
    byDay.get(k)!.push(m)
  }
  const days = [...byDay.keys()].sort()
  return (
    <div className="sched">
      {days.map((d) => (
        <section key={d}>
          <h3 className="sched__day">{d}</h3>
          <ul>
            {byDay.get(d)!.map((m) => (
              <li key={m.id} className="sched__row">
                <span>{m.group || m.stage}</span> <span>#{m.homeId} v #{m.awayId}</span>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  )
}
