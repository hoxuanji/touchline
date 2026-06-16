import { useQuery } from '@tanstack/react-query'
import { useTeams } from '../../lib/useTeams'
import { flag } from '../../lib/flags'
import { getStandings } from './api'

export default function StandingsPage() {
  const { data: standings = [] } = useQuery({ queryKey: ['standings'], queryFn: getStandings })
  const teams = useTeams()
  const groups = [...new Set(standings.map((s) => s.group))].sort()

  return (
    <div className="standings">
      {groups.map((g) => {
        const rows = standings
          .filter((s) => s.group === g)
          .sort((a, b) => b.pts - a.pts || (b.gf - b.ga) - (a.gf - a.ga))
        return (
          <section key={g}>
            <h3 className="standings__group-title">Group {g}</h3>
            <table>
              <thead>
                <tr>
                  <th style={{ textAlign: 'left' }}>#  Team</th>
                  <th>P</th><th>W</th><th>D</th><th>L</th><th>GD</th><th>Pts</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((s, i) => {
                  const t = teams.get(s.teamId)
                  return (
                    <tr key={s.teamId}>
                      <td>
                        <div className="standings__team-cell">
                          <span className="standings__pos">{i + 1}</span>
                          <span className="standings__flag">{flag(t?.country ?? '')}</span>
                          <span className="standings__name">{t?.name ?? `#${s.teamId}`}</span>
                        </div>
                      </td>
                      <td>{s.played}</td>
                      <td>{s.won}</td>
                      <td>{s.drawn}</td>
                      <td>{s.lost}</td>
                      <td>{s.gf - s.ga > 0 ? `+${s.gf - s.ga}` : s.gf - s.ga}</td>
                      <td>{s.pts}</td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </section>
        )
      })}
    </div>
  )
}
