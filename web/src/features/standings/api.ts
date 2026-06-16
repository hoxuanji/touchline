import type { Standing } from '../../lib/types'
export async function getStandings(): Promise<Standing[]> {
  const r = await fetch('/api/standings', { headers: { Accept: 'application/json' } })
  return r.json()
}
