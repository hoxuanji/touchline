import { useQuery } from '@tanstack/react-query'
import { api } from '../../lib/api'

export default function ExplorePage() {
  const { data: teams = [] } = useQuery({ queryKey: ['teams'], queryFn: api.teams })
  const { data: venues = [] } = useQuery({ queryKey: ['venues'], queryFn: api.venues })
  return (
    <div className="explore">
      <section>
        <h3>Teams</h3>
        <ul className="explore__teams">
          {teams.map((t) => <li key={t.id}>{t.name} <span className="muted">({t.group})</span></li>)}
        </ul>
      </section>
      <section>
        <h3>Venues</h3>
        <ul className="explore__venues">
          {venues.map((v) => <li key={v.id}>{v.name} — {v.city}, {v.country} ({v.capacity.toLocaleString()})</li>)}
        </ul>
      </section>
    </div>
  )
}
