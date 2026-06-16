import { useQuery } from '@tanstack/react-query'
import { getStandings } from './api'

export default function StandingsPage() {
  const { data: standings = [] } = useQuery({ queryKey: ['standings'], queryFn: getStandings })
  const groups = [...new Set(standings.map((s) => s.group))].sort()
  return (
    <div className="standings">
      {groups.map((g) => (
        <section key={g}>
          <h3>Group {g}</h3>
          <table>
            <thead><tr><th>Team</th><th>P</th><th>W</th><th>D</th><th>L</th><th>GD</th><th>Pts</th></tr></thead>
            <tbody>
              {standings.filter((s) => s.group === g).map((s) => (
                <tr key={s.teamId}>
                  <td>#{s.teamId}</td><td>{s.played}</td><td>{s.won}</td><td>{s.drawn}</td><td>{s.lost}</td><td>{s.gf - s.ga}</td><td>{s.pts}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      ))}
    </div>
  )
}
