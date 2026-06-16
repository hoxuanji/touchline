import { useQuery } from '@tanstack/react-query'
import { api } from './api'
import type { Team } from './types'

export interface TeamInfo { name: string; country: string; group: string }

export function useTeams(): Map<number, TeamInfo> {
  const { data: teams = [] } = useQuery<Team[]>({ queryKey: ['teams'], queryFn: api.teams, staleTime: Infinity })
  return new Map(teams.map((t) => [t.id, { name: t.name, country: t.country, group: t.group }]))
}
