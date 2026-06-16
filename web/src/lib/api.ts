import type { Match, Standing, Team, Venue } from './types'

async function get<T>(path: string): Promise<T> {
  const res = await fetch(path, { headers: { Accept: 'application/json' } })
  if (!res.ok) throw new Error(`${path} -> ${res.status}`)
  return res.json() as Promise<T>
}

export const api = {
  fixtures: () => get<Match[]>('/api/fixtures'),
  standings: () => get<Standing[]>('/api/standings'),
  teams: () => get<Team[]>('/api/teams'),
  venues: () => get<Venue[]>('/api/venues'),
}
