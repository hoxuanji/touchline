import type { Match, Team, Venue, Standing, MatchEvent } from '../lib/types'
import { api } from '../lib/api'

export async function getMatch(id: number): Promise<{
  match: Match; events: MatchEvent[]; homeName: string; awayName: string
}> {
  const r = await fetch(`/api/matches/${id}`, { headers: { Accept: 'application/json' } })
  return r.json()
}

export async function getNotes(subjectType: string, subjectId: number) {
  const r = await fetch(`/api/notes?subjectType=${subjectType}&subjectId=${subjectId}`, { headers: { Accept: 'application/json' } })
  return r.json()
}

export async function addNote(subjectType: string, subjectId: number, body: string) {
  return fetch('/api/notes', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ subjectType, subjectId, body }),
  })
}

export async function getFollows(): Promise<number[]> {
  const r = await fetch('/api/follows', { headers: { Accept: 'application/json' } })
  return r.json()
}

export async function follow(teamId: number) {
  return fetch(`/api/follows/${teamId}`, { method: 'POST' })
}

export async function unfollow(teamId: number) {
  return fetch(`/api/follows/${teamId}`, { method: 'DELETE' })
}

export { api }
export type { Match, Team, Venue, Standing, MatchEvent }
